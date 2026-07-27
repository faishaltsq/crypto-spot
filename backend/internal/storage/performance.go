package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/example/crypto-spot-signal/internal/performance"
)

// PerformanceReport reads execution simulations, never raw outcome returns, for proof-of-edge reporting.
func (r *Repository) PerformanceReport(ctx context.Context, filters performance.Filters) (performance.Report, error) {
	clauses, args := []string{"1=1"}, []interface{}{}
	add := func(condition string, value interface{}) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(condition, len(args)))
	}
	if filters.DateFrom != nil {
		add("s.created_at >= $%d", *filters.DateFrom)
	}
	if filters.DateTo != nil {
		add("s.created_at <= $%d", *filters.DateTo)
	}
	if filters.Pair != "" {
		add("s.symbol = $%d", filters.Pair)
	}
	if filters.Tier != "" {
		add("COALESCE(s.feature_snapshot->>'tier', '') = $%d", filters.Tier)
	}
	if filters.Timeframe != "" {
		add("s.primary_timeframe = $%d", filters.Timeframe)
	}
	if filters.SignalStatus != "" {
		add("CASE WHEN p.signal_id IS NULL THEN 'PENDING_SIMULATION' WHEN p.simulation_status = 'INCOMPLETE' THEN 'INCOMPLETE_SIMULATION' WHEN p.simulation_status = 'PARTIAL_FILL' THEN 'PARTIAL_FILL' WHEN p.net_return IS NOT NULL AND p.gross_return IS NOT NULL THEN 'EVALUATED' ELSE 'INCOMPLETE_SIMULATION' END = $%d", filters.SignalStatus)
	}
	if filters.ScoreBucket != "" {
		add("CASE WHEN s.rule_score >= 70 AND s.rule_score < 75 THEN '70-74' WHEN s.rule_score >= 75 AND s.rule_score < 80 THEN '75-79' WHEN s.rule_score >= 80 AND s.rule_score < 85 THEN '80-84' WHEN s.rule_score >= 85 AND s.rule_score < 90 THEN '85-89' WHEN s.rule_score >= 90 AND s.rule_score <= 100 THEN '90-100' ELSE 'OUT_OF_RANGE' END = $%d", filters.ScoreBucket)
	}
	if filters.MarketRegime != "" {
		add("COALESCE(s.feature_snapshot->>'marketRegime', 'UNAVAILABLE') = $%d", filters.MarketRegime)
	}
	if filters.RuleVersion != "" {
		add("COALESCE(s.signal_version->>'ruleVersion', '') = $%d", filters.RuleVersion)
	}
	if filters.ModelVersion != "" {
		add("COALESCE(s.signal_version->>'modelVersion', '') = $%d", filters.ModelVersion)
	}
	if filters.AIDecision != "" {
		add("COALESCE(s.ai_review->>'decision', '') = $%d", filters.AIDecision)
	}
	if filters.Notional != "" {
		add("COALESCE(p.notional::text, '') = $%d", filters.Notional)
	}
	query := fmt.Sprintf(`SELECT s.created_at, s.symbol, COALESCE(s.feature_snapshot->>'tier',''), s.primary_timeframe,
		s.status, COALESCE(s.feature_snapshot->>'marketRegime','UNAVAILABLE'), COALESCE(s.signal_version->>'ruleVersion','UNAVAILABLE'),
		COALESCE(s.signal_version->>'modelVersion','UNAVAILABLE'), COALESCE(s.ai_review->>'decision','UNAVAILABLE'),
		COALESCE(s.ai_review->>'provider','UNAVAILABLE'), COALESCE(s.data_quality_status::text,'UNAVAILABLE'),
		COALESCE(s.threshold_detail->>'spoofStatusAtSignal','UNAVAILABLE'), s.rule_score, p.notional,
		p.gross_return, p.net_return, p.entry_fee, p.exit_fee, p.entry_slippage, p.exit_slippage,
		o.returns, COALESCE(o.max_favorable_pct, 0), COALESCE(o.max_adverse_pct, 0), COALESCE(o.target_hit, false), COALESCE(o.invalidation_hit, false),
		CASE WHEN p.signal_id IS NULL THEN 'PENDING_SIMULATION' WHEN p.simulation_status = 'INCOMPLETE' THEN 'INCOMPLETE_SIMULATION' WHEN p.simulation_status = 'PARTIAL_FILL' THEN 'PARTIAL_FILL' WHEN p.net_return IS NOT NULL AND p.gross_return IS NOT NULL THEN 'EVALUATED' ELSE 'INCOMPLETE_SIMULATION' END,
		COALESCE(EXTRACT(EPOCH FROM p.simulated_at - s.created_at), 0)
		FROM signals s LEFT JOIN paper_execution_simulations p ON p.signal_id = s.id
		LEFT JOIN signal_outcomes_v2 o ON o.signal_id = s.id WHERE %s ORDER BY s.created_at, p.notional`, strings.Join(clauses, " AND "))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return performance.Report{}, err
	}
	defer rows.Close()
	samples := []performance.Sample{}
	for rows.Next() {
		var sample performance.Sample
		var notional *float64
		var duration float64
		var returnsJSON []byte
		if err := rows.Scan(&sample.CreatedAt, &sample.Pair, &sample.Tier, &sample.Timeframe, &sample.SignalStatus, &sample.MarketRegime, &sample.RuleVersion, &sample.ModelVersion, &sample.AIDecision, &sample.AIProvider, &sample.DataQuality, &sample.SpoofRisk, &sample.Score, &notional, &sample.GrossReturn, &sample.NetReturn, &sample.EntryFee, &sample.ExitFee, &sample.EntrySlippage, &sample.ExitSlippage, &returnsJSON, &sample.MFE, &sample.MAE, &sample.TargetHit, &sample.InvalidationHit, &sample.SimulationStatus, &duration); err != nil {
			return performance.Report{}, err
		}
		if notional != nil {
			sample.Notional = *notional
		}
		sample.Duration = time.Duration(duration * float64(time.Second))
		sample.HorizonReturns = parsePerformanceHorizons(returnsJSON)
		samples = append(samples, sample)
	}
	if err := rows.Err(); err != nil {
		return performance.Report{}, err
	}
	return performance.Build(samples), nil
}

func parsePerformanceHorizons(raw []byte) map[string]performance.HorizonSample {
	if len(raw) == 0 {
		return nil
	}
	var decoded map[string]struct {
		ReturnPct    float64  `json:"returnPct"`
		NetReturnPct *float64 `json:"netReturnPct"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}
	result := make(map[string]performance.HorizonSample, len(decoded))
	for horizon, value := range decoded {
		if value.NetReturnPct == nil {
			continue
		}
		result[horizon] = performance.HorizonSample{GrossReturn: value.ReturnPct / 100, NetReturn: *value.NetReturnPct / 100}
	}
	return result
}
