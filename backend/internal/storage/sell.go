package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/jackc/pgx/v5"
)

// SellSignalExtras holds the SELL-specific JSONB/numeric columns added by
// migration 012_sell_signals, kept separate from domain.Signal so the BUY
// engine's SaveSignal path never needs to know about SELL-only fields.
type SellSignalExtras struct {
	SellScore             float64
	SellRuleScore         float64
	SellModelProbability  float64
	SellBaseThreshold     float64
	SellFinalThreshold    float64
	TradeFlowSnapshot     interface{}
	BearishStructure      interface{}
	SpoofAnalysis         interface{}
	InvalidationCondition interface{}
	InvalidationReason    string
}

// SaveSellSignal persists a SELL-family domain.Signal (SELL_SETUP,
// SELL_CONFIRMED, TAKE_PROFIT_SUGGESTED, AVOID_ENTRY, EXIT_WARNING) into the
// same `signals` table the BUY engine uses, additionally populating the
// SELL-only columns from migration 012 so the Terminal/Diagnostic UI can
// query full trade-flow/structure/spoof evidence without re-deriving it.
func (r *Repository) SaveSellSignal(ctx context.Context, signal domain.Signal, extras SellSignalExtras) error {
	reasonsJSON, _ := json.Marshal(signal.Reasons)
	riskJSON, _ := json.Marshal(signal.RiskFlags)
	featureJSON, _ := json.Marshal(signal.Features)
	versionJSON, _ := json.Marshal(signal.Version)
	evidenceJSON, _ := json.Marshal(signal.Evidence)
	thresholdJSON, _ := json.Marshal(signal.Threshold)
	missingFeaturesJSON, _ := json.Marshal(signal.MissingFeatures)
	blockedReasonsJSON, _ := json.Marshal(signal.BlockedReasons)

	tradeFlowJSON, _ := json.Marshal(extras.TradeFlowSnapshot)
	structureJSON, _ := json.Marshal(extras.BearishStructure)
	spoofJSON, _ := json.Marshal(extras.SpoofAnalysis)
	invalidationJSON, _ := json.Marshal(extras.InvalidationCondition)
	supportingJSON, _ := json.Marshal(signal.Evidence.SupportingEvidence)
	contradictingJSON, _ := json.Marshal(signal.Evidence.ContradictingEvidence)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO signals (
			id, symbol, signal_type, status, primary_timeframe,
			entry_price, invalidation_price, target_price_1, target_price_2,
			rule_score, ai_review, reasons, risk_flags, feature_snapshot,
			signal_version, evidence, threshold_detail, data_quality_score,
			data_quality_status, data_source, missing_features, blocked_reasons,
			created_at, expires_at,
			sell_score, sell_rule_score, sell_model_probability,
			sell_base_threshold, sell_final_threshold,
			trade_flow_snapshot, bearish_structure_snapshot, spoof_analysis,
			supporting_evidence, contradicting_evidence,
			invalidation_condition, invalidation_reason
		) VALUES (
			$1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,
			'{}'::jsonb,$11::jsonb,$12::jsonb,$13::jsonb,
			$14::jsonb,$15::jsonb,$16::jsonb,$17,
			$18,$19,$20::jsonb,$21::jsonb,
			$22,$23,
			$24,$25,$26,$27,$28,
			$29::jsonb,$30::jsonb,$31::jsonb,
			$32::jsonb,$33::jsonb,
			$34::jsonb,$35
		)
		ON CONFLICT (id) DO NOTHING
	`,
		signal.ID, signal.Symbol, signal.Type, signal.Status, signal.PrimaryTimeframe,
		signal.EntryPrice, signal.Invalidation, signal.Target1, signal.Target2,
		signal.RuleScore,
		reasonsJSON, riskJSON, featureJSON,
		versionJSON, evidenceJSON, thresholdJSON, signal.DataQualityScore,
		signal.DataQualityStatus, signal.DataSource, missingFeaturesJSON, blockedReasonsJSON,
		signal.CreatedAt, signal.ExpiresAt,
		extras.SellScore, extras.SellRuleScore, extras.SellModelProbability,
		extras.SellBaseThreshold, extras.SellFinalThreshold,
		tradeFlowJSON, structureJSON, spoofJSON,
		supportingJSON, contradictingJSON,
		invalidationJSON, extras.InvalidationReason,
	)
	return err
}

// SellCandidate is a SELL-family signal awaiting directional-accuracy
// evaluation. Target1/Invalidation come straight from the `signals` table
// row this signal wrote (see signals/sell/protective_sell.go's
// priceLevels), so the outcome evaluator never needs to re-run the SELL
// engine's evidence logic — it purely checks whether price actually moved
// the direction the signal claimed.
type SellCandidate struct {
	SignalID     string
	Symbol       string
	SignalType   string
	EntryPrice   float64
	Target1      float64
	Invalidation float64
	CreatedAt    time.Time
}

// ActiveSellCandidates returns SELL-family signals created within the last
// 25 hours that have not yet been closed, mirroring ActiveCandidates' BUY
// retention window so the 24h horizon is always fully observed before a
// signal drops out of tracking.
func (r *Repository) ActiveSellCandidates(ctx context.Context) ([]SellCandidate, error) {
	cutoff := time.Now().Add(-25 * time.Hour)
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, symbol, signal_type, entry_price, target_price_1, invalidation_price, created_at
		FROM signals
		WHERE signal_type IN ('SELL_SETUP', 'SELL_CONFIRMED') AND created_at > $1 AND closed_at IS NULL
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []SellCandidate
	for rows.Next() {
		var c SellCandidate
		if err := rows.Scan(&c.SignalID, &c.Symbol, &c.SignalType, &c.EntryPrice, &c.Target1, &c.Invalidation, &c.CreatedAt); err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
	}
	return candidates, rows.Err()
}

// CloseSellSignal marks a SELL-family signal as closed once directional
// invalidation is confirmed by live price action, mirroring how
// UpdateOutcome closes BUY signals on target/invalidation hit.
func (r *Repository) CloseSellSignal(ctx context.Context, signalID, status, invalidationReason string, closedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE signals SET status = $2, closed_at = $3, invalidation_reason = $4
		WHERE id = $1::uuid AND closed_at IS NULL
	`, signalID, status, closedAt, invalidationReason)
	return err
}

// SellSignalDetail bundles a domain.Signal row with its SELL-only columns
// (migration 012), for the SELL-family list/detail API and the Terminal/
// Pair Diagnostic UI. It never gets returned for BUY-family signal types.
type SellSignalDetail struct {
	domain.Signal
	SellScore             float64         `json:"sellScore"`
	SellRuleScore         float64         `json:"sellRuleScore"`
	SellBaseThreshold     float64         `json:"sellBaseThreshold"`
	SellFinalThreshold    float64         `json:"sellFinalThreshold"`
	TradeFlowSnapshot     json.RawMessage `json:"tradeFlowSnapshot"`
	BearishStructure      json.RawMessage `json:"bearishStructureSnapshot"`
	SpoofAnalysis         json.RawMessage `json:"spoofAnalysis"`
	SupportingEvidence    []string        `json:"supportingEvidence"`
	ContradictingEvidence []string        `json:"contradictingEvidence"`
	InvalidationReason    *string         `json:"invalidationReason"`
}

var sellSignalTypes = []string{
	domain.SellSignalSetup, domain.SellSignalConfirmed,
	domain.TakeProfitSuggested, domain.AvoidEntrySignal, domain.ExitWarningSignal,
}

// ListSellSignals returns SELL-family signals (all five types) ordered by
// recency, optionally filtered to one symbol. It is the SELL-side
// counterpart to ListSignalsFiltered, kept in its own function rather than
// overloading SignalFilter because SELL signals expose a materially
// different set of columns (trade flow, structure, spoof, invalidation
// reason) that BUY signals don't have.
func (r *Repository) ListSellSignals(ctx context.Context, symbol string, limit int) ([]SellSignalDetail, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	args := []interface{}{sellSignalTypes, limit}
	symbolClause := ""
	if symbol != "" {
		symbolClause = "AND symbol = $3"
		args = append(args, symbol)
	}
	query := `
		SELECT id::text, symbol, signal_type, status, primary_timeframe,
			entry_price, invalidation_price, target_price_1, target_price_2,
			rule_score, reasons, risk_flags, feature_snapshot,
			signal_version, evidence, threshold_detail, data_quality_score,
			data_quality_status, data_source, missing_features, blocked_reasons,
			created_at, expires_at,
			sell_score, sell_rule_score, sell_base_threshold, sell_final_threshold,
			trade_flow_snapshot, bearish_structure_snapshot, spoof_analysis,
			supporting_evidence, contradicting_evidence, invalidation_reason
		FROM signals
		WHERE signal_type = ANY($1) ` + symbolClause + `
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SellSignalDetail
	for rows.Next() {
		detail, err := scanSellSignal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, detail)
	}
	return out, rows.Err()
}

// GetSellSignal returns one SELL-family signal by ID, or pgx.ErrNoRows.
func (r *Repository) GetSellSignal(ctx context.Context, id string) (SellSignalDetail, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, symbol, signal_type, status, primary_timeframe,
			entry_price, invalidation_price, target_price_1, target_price_2,
			rule_score, reasons, risk_flags, feature_snapshot,
			signal_version, evidence, threshold_detail, data_quality_score,
			data_quality_status, data_source, missing_features, blocked_reasons,
			created_at, expires_at,
			sell_score, sell_rule_score, sell_base_threshold, sell_final_threshold,
			trade_flow_snapshot, bearish_structure_snapshot, spoof_analysis,
			supporting_evidence, contradicting_evidence, invalidation_reason
		FROM signals WHERE id = $1::uuid
	`, id)
	return scanSellSignal(row)
}

func scanSellSignal(row rowScanner) (SellSignalDetail, error) {
	var detail SellSignalDetail
	var reasonsJSON, riskJSON, featureJSON []byte
	var versionJSON, evidenceJSON, thresholdJSON []byte
	var missingFeaturesJSON, blockedReasonsJSON []byte
	var supportingJSON, contradictingJSON []byte

	err := row.Scan(
		&detail.ID, &detail.Symbol, &detail.Type, &detail.Status, &detail.PrimaryTimeframe,
		&detail.EntryPrice, &detail.Invalidation, &detail.Target1, &detail.Target2,
		&detail.RuleScore, &reasonsJSON, &riskJSON, &featureJSON,
		&versionJSON, &evidenceJSON, &thresholdJSON, &detail.DataQualityScore,
		&detail.DataQualityStatus, &detail.DataSource, &missingFeaturesJSON, &blockedReasonsJSON,
		&detail.CreatedAt, &detail.ExpiresAt,
		&detail.SellScore, &detail.SellRuleScore, &detail.SellBaseThreshold, &detail.SellFinalThreshold,
		&detail.TradeFlowSnapshot, &detail.BearishStructure, &detail.SpoofAnalysis,
		&supportingJSON, &contradictingJSON, &detail.InvalidationReason,
	)
	if err != nil {
		return SellSignalDetail{}, err
	}
	_ = json.Unmarshal(reasonsJSON, &detail.Reasons)
	_ = json.Unmarshal(riskJSON, &detail.RiskFlags)
	_ = json.Unmarshal(featureJSON, &detail.Features)
	_ = json.Unmarshal(versionJSON, &detail.Version)
	_ = json.Unmarshal(evidenceJSON, &detail.Evidence)
	_ = json.Unmarshal(thresholdJSON, &detail.Threshold)
	_ = json.Unmarshal(missingFeaturesJSON, &detail.MissingFeatures)
	_ = json.Unmarshal(blockedReasonsJSON, &detail.BlockedReasons)
	_ = json.Unmarshal(supportingJSON, &detail.SupportingEvidence)
	_ = json.Unmarshal(contradictingJSON, &detail.ContradictingEvidence)
	return detail, nil
}

// GetSellSignalOutcome returns the directional-accuracy outcome for a SELL
// signal, or pgx.ErrNoRows if it hasn't been evaluated yet.
func (r *Repository) GetSellSignalOutcome(ctx context.Context, signalID string) (SellSignalOutcome, error) {
	var o SellSignalOutcome
	row := r.pool.QueryRow(ctx, `
		SELECT signal_id::text, symbol, evaluated_at, directional_return, directional_accuracy,
			max_downside_move, max_adverse_upside_move, support_reclaim,
			breakdown_follow_through, invalidated,
			avoid_entry_effectiveness, exit_warning_effectiveness, take_profit_effectiveness
		FROM sell_signal_outcomes WHERE signal_id = $1::uuid
	`, signalID)
	err := row.Scan(
		&o.SignalID, &o.Symbol, &o.EvaluatedAt, &o.DirectionalReturn, &o.DirectionalAccuracy,
		&o.MaxDownsideMove, &o.MaxAdverseUpsideMove, &o.SupportReclaim,
		&o.BreakdownFollowThrough, &o.Invalidated,
		&o.AvoidEntryEffectiveness, &o.ExitWarningEffectiveness, &o.TakeProfitEffectiveness,
	)
	return o, err
}

// LatestBuySignalState looks up the most recent BUY-family signal for a
// symbol so the SELL engine's orchestrator (main.go's scannerLoop) can build
// an accurate sell.ActiveBuyContext without the SELL engine ever touching
// storage directly. Only BUY_SETUP/BUY_CONFIRMED_CANDIDATE/BUY_CONFIRMED
// types are considered; SELL-family signal types are ignored here.
func (r *Repository) LatestBuySignalState(ctx context.Context, symbol string) (id, signalType, status string, entryPrice float64, createdAt time.Time, found bool, err error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, signal_type, status, entry_price, created_at
		FROM signals
		WHERE symbol = $1 AND signal_type IN ('BUY_SETUP', 'BUY_CONFIRMED') AND closed_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`, symbol)
	if scanErr := row.Scan(&id, &signalType, &status, &entryPrice, &createdAt); scanErr != nil {
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return "", "", "", 0, time.Time{}, false, nil
		}
		return "", "", "", 0, time.Time{}, false, scanErr
	}
	return id, signalType, status, entryPrice, createdAt, true, nil
}

// SellSignalOutcome mirrors migration 012's sell_signal_outcomes table.
type SellSignalOutcome struct {
	SignalID                  string
	Symbol                    string
	EvaluatedAt               time.Time
	DirectionalReturn         float64
	DirectionalAccuracy       bool
	MaxDownsideMove           float64
	MaxAdverseUpsideMove      float64
	SupportReclaim            bool
	BreakdownFollowThrough    bool
	Invalidated               bool
	AvoidEntryEffectiveness   *float64
	ExitWarningEffectiveness  *float64
	TakeProfitEffectiveness   *float64
}

// SaveSellSignalOutcome upserts one directional-accuracy outcome row.
func (r *Repository) SaveSellSignalOutcome(ctx context.Context, o SellSignalOutcome) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sell_signal_outcomes (
			signal_id, symbol, evaluated_at, directional_return, directional_accuracy,
			max_downside_move, max_adverse_upside_move, support_reclaim,
			breakdown_follow_through, invalidated,
			avoid_entry_effectiveness, exit_warning_effectiveness, take_profit_effectiveness
		) VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (signal_id) DO UPDATE SET
			evaluated_at = EXCLUDED.evaluated_at,
			directional_return = EXCLUDED.directional_return,
			directional_accuracy = EXCLUDED.directional_accuracy,
			max_downside_move = EXCLUDED.max_downside_move,
			max_adverse_upside_move = EXCLUDED.max_adverse_upside_move,
			support_reclaim = EXCLUDED.support_reclaim,
			breakdown_follow_through = EXCLUDED.breakdown_follow_through,
			invalidated = EXCLUDED.invalidated,
			avoid_entry_effectiveness = EXCLUDED.avoid_entry_effectiveness,
			exit_warning_effectiveness = EXCLUDED.exit_warning_effectiveness,
			take_profit_effectiveness = EXCLUDED.take_profit_effectiveness
	`,
		o.SignalID, o.Symbol, o.EvaluatedAt, o.DirectionalReturn, o.DirectionalAccuracy,
		o.MaxDownsideMove, o.MaxAdverseUpsideMove, o.SupportReclaim,
		o.BreakdownFollowThrough, o.Invalidated,
		o.AvoidEntryEffectiveness, o.ExitWarningEffectiveness, o.TakeProfitEffectiveness,
	)
	return err
}
