package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/execution_simulation"
	"github.com/example/crypto-spot-signal/internal/market"
	"github.com/example/crypto-spot-signal/internal/outcome"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, databaseURL string) (*Repository, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	config.MaxConns = 12
	config.MinConns = 2
	config.MaxConnLifetime = 45 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Repository{pool: pool}, nil
}

func (r *Repository) Close() {
	if r != nil && r.pool != nil {
		r.pool.Close()
	}
}

func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}

// --- Outcome and Simulation Methods ---

// ActiveCandidates keeps signals one extra hour so the 24h horizon is observed and persisted.
func (r *Repository) ActiveCandidates(ctx context.Context) ([]outcome.Candidate, error) {
	cutoff := time.Now().Add(-25 * time.Hour)
	rows, err := r.pool.Query(ctx, `
		SELECT id, symbol, entry_price, target_price_1, target_price_2, invalidation_price, created_at
		FROM signals
		WHERE created_at > $1
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []outcome.Candidate
	for rows.Next() {
		var c outcome.Candidate
		if err := rows.Scan(
			&c.SignalID, &c.Symbol, &c.EntryPrice,
			&c.Target1, &c.Target2, &c.Invalidation, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
	}
	return candidates, nil
}

// SaveOutcome persists multi-horizon returns to signal_outcomes_v2.
func (r *Repository) SaveOutcome(ctx context.Context, result outcome.Result) error {
	returnsJSON, _ := json.Marshal(result.Returns)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO signal_outcomes_v2 (
			signal_id, symbol, evaluated_at, returns,
			max_favorable_pct, max_adverse_pct,
			target_hit, target_hit_at, invalidation_hit, invalidation_hit_at
		) VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (signal_id) DO UPDATE SET
			evaluated_at = EXCLUDED.evaluated_at,
			returns = signal_outcomes_v2.returns || EXCLUDED.returns,
			max_favorable_pct = EXCLUDED.max_favorable_pct,
			max_adverse_pct = EXCLUDED.max_adverse_pct,
			target_hit = EXCLUDED.target_hit,
			target_hit_at = EXCLUDED.target_hit_at,
			invalidation_hit = EXCLUDED.invalidation_hit,
			invalidation_hit_at = EXCLUDED.invalidation_hit_at
	`,
		result.SignalID, result.Symbol, result.EvaluatedAt, returnsJSON,
		result.MaximumFavorablePct, result.MaximumAdversePct,
		result.TargetHit, result.TargetHitAt, result.InvalidationHit, result.InvalidationHitAt,
	)
	return err
}

func (r *Repository) SaveEntrySimulations(ctx context.Context, rows []execution_simulation.Simulation) error {
	batch := &pgx.Batch{}
	for _, row := range rows {
		batch.Queue(`INSERT INTO paper_execution_simulations (
			signal_id, notional, reference_price, estimated_entry_price, entry_fee, entry_slippage, entry_slippage_bps,
			maximum_supported_notional, depth_coverage, liquidity_confidence, filled_notional, unfilled_notional,
			partial_fill, simulation_status, simulated_at
		) VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (signal_id, notional) DO NOTHING`,
			row.SignalID, row.Notional, row.ReferencePrice, row.EstimatedEntryPrice, row.EntryFee, row.EntrySlippage,
			row.EntrySlippageBPS, row.MaximumSupportedNotional, row.DepthCoverage, row.LiquidityConfidence,
			row.FilledNotional, row.UnfilledNotional, row.PartialFill, row.Status, row.SimulatedAt)
	}
	results := r.pool.SendBatch(ctx, batch)
	defer results.Close()
	for range rows {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) SimulationsForSignal(ctx context.Context, signalID string) ([]execution_simulation.Simulation, error) {
	rows, err := r.pool.Query(ctx, `SELECT id::text, signal_id::text, notional, reference_price, estimated_entry_price,
		entry_fee, entry_slippage, entry_slippage_bps, maximum_supported_notional, depth_coverage, liquidity_confidence,
		filled_notional, unfilled_notional, partial_fill, simulation_status, simulated_at, created_at, updated_at
		FROM paper_execution_simulations WHERE signal_id = $1::uuid ORDER BY notional`, signalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var simulations []execution_simulation.Simulation
	for rows.Next() {
		var row execution_simulation.Simulation
		if err := rows.Scan(&row.ID, &row.SignalID, &row.Notional, &row.ReferencePrice, &row.EstimatedEntryPrice,
			&row.EntryFee, &row.EntrySlippage, &row.EntrySlippageBPS, &row.MaximumSupportedNotional, &row.DepthCoverage,
			&row.LiquidityConfidence, &row.FilledNotional, &row.UnfilledNotional, &row.PartialFill, &row.Status,
			&row.SimulatedAt, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, err
		}
		if row.EstimatedEntryPrice != nil && row.FilledNotional != nil {
			row.BaseFilled = *row.FilledNotional / *row.EstimatedEntryPrice
		}
		simulations = append(simulations, row)
	}
	return simulations, rows.Err()
}

func (r *Repository) UpdateExitSimulations(ctx context.Context, rows []execution_simulation.Simulation) error {
	batch := &pgx.Batch{}
	for _, row := range rows {
		batch.Queue(`UPDATE paper_execution_simulations SET estimated_exit_price=$2, exit_fee=$3, exit_slippage=$4,
			exit_slippage_bps=$5, gross_return=$6, net_return=$7, partial_fill=$8, simulation_status=$9,
			simulated_at=$10, updated_at=NOW() WHERE id=$1::uuid`, row.ID, row.EstimatedExitPrice, row.ExitFee,
			row.ExitSlippage, row.ExitSlippageBPS, row.GrossReturn, row.NetReturn, row.PartialFill, row.Status, row.SimulatedAt)
	}
	results := r.pool.SendBatch(ctx, batch)
	defer results.Close()
	for range rows {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) SaveCandle(ctx context.Context, candle domain.Candle) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO candles (
			symbol, timeframe, open_time, open, high, low, close,
			base_volume, quote_volume, is_closed, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())
		ON CONFLICT (symbol, timeframe, open_time) DO UPDATE SET
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			base_volume = EXCLUDED.base_volume,
			quote_volume = EXCLUDED.quote_volume,
			is_closed = EXCLUDED.is_closed,
			updated_at = NOW()
	`,
		candle.Symbol,
		candle.Timeframe,
		candle.OpenTime,
		candle.Open,
		candle.High,
		candle.Low,
		candle.Close,
		candle.BaseVolume,
		candle.QuoteVolume,
		candle.Closed,
	)
	return err
}

func (r *Repository) SaveFeature(ctx context.Context, feature domain.FeatureSnapshot) error {
	trendJSON, _ := json.Marshal(feature.TrendByTimeframe)
	reasonsJSON, _ := json.Marshal(feature.Reasons)
	riskJSON, _ := json.Marshal(feature.RiskFlags)
	featureJSON, _ := json.Marshal(feature)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO market_features (
			symbol, calculated_at, price, spread_bps, bid_depth_quote,
			ask_depth_quote, orderbook_imbalance, spoof_score,
			relative_volume_1m, volume_delta_ratio_1m, trend_alignment,
			rule_score, data_quality_score, status, trend_by_timeframe,
			reasons, risk_flags, feature_payload
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,
			$15::jsonb,$16::jsonb,$17::jsonb,$18::jsonb
		)
		ON CONFLICT (symbol, calculated_at) DO UPDATE SET
			price = EXCLUDED.price,
			rule_score = EXCLUDED.rule_score,
			feature_payload = EXCLUDED.feature_payload
	`,
		feature.Symbol,
		feature.CalculatedAt,
		feature.Price,
		feature.SpreadBPS,
		feature.BidDepthQuote,
		feature.AskDepthQuote,
		feature.OrderbookImbalance,
		feature.SpoofScore,
		feature.RelativeVolume1m,
		feature.VolumeDeltaRatio1m,
		feature.TrendAlignment,
		feature.RuleScore,
		feature.DataQualityScore,
		feature.Status,
		trendJSON,
		reasonsJSON,
		riskJSON,
		featureJSON,
	)
	return err
}

func (r *Repository) SaveSignal(ctx context.Context, signal domain.Signal) error {
	reasonsJSON, _ := json.Marshal(signal.Reasons)
	riskJSON, _ := json.Marshal(signal.RiskFlags)
	featureJSON, _ := json.Marshal(signal.Features)
	aiJSON, _ := json.Marshal(signal.AI)
	versionJSON, _ := json.Marshal(signal.Version)
	evidenceJSON, _ := json.Marshal(signal.Evidence)
	thresholdJSON, _ := json.Marshal(signal.Threshold)
	missingFeaturesJSON, _ := json.Marshal(signal.MissingFeatures)
	blockedReasonsJSON, _ := json.Marshal(signal.BlockedReasons)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO signals (
			id, symbol, signal_type, status, primary_timeframe,
			entry_price, invalidation_price, target_price_1, target_price_2,
			rule_score, ai_review, reasons, risk_flags, feature_snapshot,
			signal_version, evidence, threshold_detail, data_quality_score,
			data_quality_status, data_source, missing_features, blocked_reasons,
			created_at, expires_at
		) VALUES (
			$1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,
			$11::jsonb,$12::jsonb,$13::jsonb,$14::jsonb,
			$15::jsonb,$16::jsonb,$17::jsonb,$18,
			$19,$20,$21::jsonb,$22::jsonb,
			$23,$24
		)
		ON CONFLICT (id) DO NOTHING
	`,
		signal.ID,
		signal.Symbol,
		signal.Type,
		signal.Status,
		signal.PrimaryTimeframe,
		signal.EntryPrice,
		signal.Invalidation,
		signal.Target1,
		signal.Target2,
		signal.RuleScore,
		aiJSON,
		reasonsJSON,
		riskJSON,
		featureJSON,
		versionJSON,
		evidenceJSON,
		thresholdJSON,
		signal.DataQualityScore,
		signal.DataQualityStatus,
		signal.DataSource,
		missingFeaturesJSON,
		blockedReasonsJSON,
		signal.CreatedAt,
		signal.ExpiresAt,
	)
	return err
}

func (r *Repository) ListSignals(ctx context.Context, limit int) ([]domain.Signal, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, symbol, signal_type, status, primary_timeframe,
			entry_price, invalidation_price, target_price_1, target_price_2,
			rule_score, ai_review, reasons, risk_flags, feature_snapshot,
			signal_version, evidence, threshold_detail, data_quality_score,
			data_quality_status, data_source, missing_features, blocked_reasons,
			created_at, expires_at
		FROM signals
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	signals := make([]domain.Signal, 0, limit)
	for rows.Next() {
		signal, err := scanSignal(rows)
		if err != nil {
			return nil, err
		}
		signals = append(signals, signal)
	}
	return signals, rows.Err()
}

type SignalFilter struct {
	Limit       int
	Offset      int
	Status      *string
	Symbol      *string
	SignalType  *string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	ScoreMin    *float64
	ScoreMax    *float64
	OrderBy     string
}

type CompareHistory struct {
	NetReturn     *float64
	WinRate       *float64
	SampleCount   int
	NetExpectancy *float64
	MFE           *float64
	MAE           *float64
}

type CompareSignalState struct {
	Active           bool
	DynamicThreshold *float64
}

// CompareHistory aggregates only evaluated outcome values for one requested horizon.
func (r *Repository) CompareHistory(ctx context.Context, symbol, timeframe string, since time.Time) (CompareHistory, error) {
	var value CompareHistory
	err := r.pool.QueryRow(ctx, `
		WITH evaluated AS (
			SELECT (o.returns->$3->>'netReturnPct')::double precision AS net_return,
				o.max_favorable_pct, o.max_adverse_pct
			FROM signal_outcomes_v2 o
			JOIN signals s ON s.id = o.signal_id
			WHERE o.symbol = $1 AND s.created_at >= $2
			AND o.returns ? $3 AND (o.returns->$3->>'netReturnPct') IS NOT NULL
		)
		SELECT COUNT(*), AVG(net_return), AVG(CASE WHEN net_return > 0 THEN 1.0 ELSE 0.0 END),
			AVG(net_return), AVG(max_favorable_pct), AVG(max_adverse_pct)
		FROM evaluated`, symbol, since, timeframe).Scan(&value.SampleCount, &value.NetReturn, &value.WinRate, &value.NetExpectancy, &value.MFE, &value.MAE)
	return value, err
}

func (r *Repository) CompareSignalState(ctx context.Context, symbol string) (CompareSignalState, bool, error) {
	var value CompareSignalState
	var status string
	var thresholdJSON []byte
	err := r.pool.QueryRow(ctx, `SELECT status, threshold_detail FROM signals WHERE symbol = $1 ORDER BY created_at DESC LIMIT 1`, symbol).Scan(&status, &thresholdJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return value, false, nil
	}
	if err != nil {
		return value, false, err
	}
	var threshold struct {
		FinalThreshold *float64 `json:"finalThreshold"`
	}
	_ = json.Unmarshal(thresholdJSON, &threshold)
	value.Active = status == "CONFIRMED" || status == "ACTIVE"
	value.DynamicThreshold = threshold.FinalThreshold
	return value, true, nil
}

func (r *Repository) ListSignalsFiltered(ctx context.Context, filter SignalFilter) ([]domain.Signal, int, error) {
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 100
	}
	if filter.OrderBy == "" {
		filter.OrderBy = "created_at DESC"
	}

	allowedOrder := map[string]bool{
		"created_at DESC": true, "created_at ASC": true,
		"rule_score DESC": true, "rule_score ASC": true,
		"symbol ASC": true, "symbol DESC": true,
	}
	if !allowedOrder[filter.OrderBy] {
		filter.OrderBy = "created_at DESC"
	}

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	if filter.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.Symbol != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("symbol = $%d", argIdx))
		args = append(args, *filter.Symbol)
		argIdx++
	}
	if filter.SignalType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("signal_type = $%d", argIdx))
		args = append(args, *filter.SignalType)
		argIdx++
	}
	if filter.CreatedFrom != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.CreatedFrom)
		argIdx++
	}
	if filter.CreatedTo != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.CreatedTo)
		argIdx++
	}
	if filter.ScoreMin != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("rule_score >= $%d", argIdx))
		args = append(args, *filter.ScoreMin)
		argIdx++
	}
	if filter.ScoreMax != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("rule_score <= $%d", argIdx))
		args = append(args, *filter.ScoreMax)
		argIdx++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM signals %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, filter.Limit, filter.Offset)
	query := fmt.Sprintf(`
		SELECT id::text, symbol, signal_type, status, primary_timeframe,
			entry_price, invalidation_price, target_price_1, target_price_2,
			rule_score, ai_review, reasons, risk_flags, feature_snapshot,
			signal_version, evidence, threshold_detail, data_quality_score,
			data_quality_status, data_source, missing_features, blocked_reasons,
			created_at, expires_at
		FROM signals
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, filter.OrderBy, argIdx, argIdx+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	signals := make([]domain.Signal, 0, filter.Limit)
	for rows.Next() {
		signal, err := scanSignal(rows)
		if err != nil {
			return nil, 0, err
		}
		signals = append(signals, signal)
	}
	if err := r.attachSimulationsToSignals(ctx, signals); err != nil {
		return nil, 0, err
	}
	return signals, total, rows.Err()
}

func (r *Repository) GetSignal(ctx context.Context, id string) (domain.Signal, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, symbol, signal_type, status, primary_timeframe,
			entry_price, invalidation_price, target_price_1, target_price_2,
			rule_score, ai_review, reasons, risk_flags, feature_snapshot,
			signal_version, evidence, threshold_detail, data_quality_score,
			data_quality_status, data_source, missing_features, blocked_reasons,
			created_at, expires_at
		FROM signals
		WHERE id = $1::uuid
	`, id)
	signal, err := scanSignal(row)
	if err != nil {
		return signal, err
	}
	return signal, r.attachSimulations(ctx, &signal)
}

func (r *Repository) attachSimulations(ctx context.Context, signal *domain.Signal) error {
	rows, err := r.pool.Query(ctx, `SELECT notional, entry_fee, exit_fee, entry_slippage, exit_slippage,
		entry_slippage_bps, exit_slippage_bps, gross_return, net_return, simulation_status
		FROM paper_execution_simulations WHERE signal_id = $1::uuid ORDER BY notional`, signal.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var simulation domain.PaperSimulation
		if err := rows.Scan(&simulation.Notional, &simulation.EntryFee, &simulation.ExitFee, &simulation.EntrySlippage,
			&simulation.ExitSlippage, &simulation.EntrySlippageBPS, &simulation.ExitSlippageBPS, &simulation.GrossReturn,
			&simulation.NetReturn, &simulation.SimulationStatus); err != nil {
			return err
		}
		signal.Simulations = append(signal.Simulations, simulation)
	}
	return rows.Err()
}

func (r *Repository) attachSimulationsToSignals(ctx context.Context, signals []domain.Signal) error {
	if len(signals) == 0 {
		return nil
	}
	ids := make([]string, len(signals))
	byID := make(map[string]*domain.Signal, len(signals))
	for i := range signals {
		ids[i], byID[signals[i].ID] = signals[i].ID, &signals[i]
	}
	rows, err := r.pool.Query(ctx, `SELECT signal_id::text, notional, entry_fee, exit_fee, entry_slippage, exit_slippage,
		entry_slippage_bps, exit_slippage_bps, gross_return, net_return, simulation_status
		FROM paper_execution_simulations WHERE signal_id = ANY($1::uuid[]) ORDER BY signal_id, notional`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var simulation domain.PaperSimulation
		if err := rows.Scan(&id, &simulation.Notional, &simulation.EntryFee, &simulation.ExitFee, &simulation.EntrySlippage,
			&simulation.ExitSlippage, &simulation.EntrySlippageBPS, &simulation.ExitSlippageBPS, &simulation.GrossReturn,
			&simulation.NetReturn, &simulation.SimulationStatus); err != nil {
			return err
		}
		if signal := byID[id]; signal != nil {
			signal.Simulations = append(signal.Simulations, simulation)
		}
	}
	return rows.Err()
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanSignal(row rowScanner) (domain.Signal, error) {
	var signal domain.Signal
	var aiJSON, reasonsJSON, riskJSON, featureJSON []byte
	var versionJSON, evidenceJSON, thresholdJSON []byte
	var missingFeaturesJSON, blockedReasonsJSON []byte
	err := row.Scan(
		&signal.ID,
		&signal.Symbol,
		&signal.Type,
		&signal.Status,
		&signal.PrimaryTimeframe,
		&signal.EntryPrice,
		&signal.Invalidation,
		&signal.Target1,
		&signal.Target2,
		&signal.RuleScore,
		&aiJSON,
		&reasonsJSON,
		&riskJSON,
		&featureJSON,
		&versionJSON,
		&evidenceJSON,
		&thresholdJSON,
		&signal.DataQualityScore,
		&signal.DataQualityStatus,
		&signal.DataSource,
		&missingFeaturesJSON,
		&blockedReasonsJSON,
		&signal.CreatedAt,
		&signal.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return signal, pgx.ErrNoRows
		}
		return signal, err
	}
	_ = json.Unmarshal(aiJSON, &signal.AI)
	_ = json.Unmarshal(reasonsJSON, &signal.Reasons)
	_ = json.Unmarshal(riskJSON, &signal.RiskFlags)
	_ = json.Unmarshal(featureJSON, &signal.Features)
	_ = json.Unmarshal(versionJSON, &signal.Version)
	_ = json.Unmarshal(evidenceJSON, &signal.Evidence)
	_ = json.Unmarshal(thresholdJSON, &signal.Threshold)
	if missingFeaturesJSON != nil {
		_ = json.Unmarshal(missingFeaturesJSON, &signal.MissingFeatures)
	}
	if blockedReasonsJSON != nil {
		_ = json.Unmarshal(blockedReasonsJSON, &signal.BlockedReasons)
	}
	signal.Enrich()
	return signal, nil
}

func (r *Repository) ListOutcomeCandidates(ctx context.Context) ([]domain.OutcomeCandidate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, symbol, entry_price, target_price_1,
			invalidation_price, created_at, expires_at
		FROM signals
		WHERE created_at >= NOW() - INTERVAL '6 hours'
		  AND closed_at IS NULL
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.OutcomeCandidate, 0)
	for rows.Next() {
		var item domain.OutcomeCandidate
		if err := rows.Scan(
			&item.ID,
			&item.Symbol,
			&item.EntryPrice,
			&item.Target1,
			&item.Invalidation,
			&item.CreatedAt,
			&item.ExpiresAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// UpdateOutcome persists the latest return metrics for a BUY signal and,
// if the signal just crossed into a terminal lifecycle state (target hit,
// invalidation hit, or expiry), updates its status. Returns the new
// terminal status ("" if the signal is still active) so the caller can
// broadcast a realtime lifecycle event — callers must never let a BUY
// signal transition out of "active" silently, or the Terminal/Signals
// page will disagree on active counts.
func (r *Repository) UpdateOutcome(
	ctx context.Context,
	item domain.OutcomeCandidate,
	currentPrice float64,
	now time.Time,
) (string, error) {
	if item.EntryPrice <= 0 || currentPrice <= 0 {
		return "", nil
	}
	returnValue := (currentPrice - item.EntryPrice) / item.EntryPrice
	age := now.Sub(item.CreatedAt)

	var return5m, return15m, return1h, return4h *float64
	if age >= 5*time.Minute {
		value := returnValue
		return5m = &value
	}
	if age >= 15*time.Minute {
		value := returnValue
		return15m = &value
	}
	if age >= time.Hour {
		value := returnValue
		return1h = &value
	}
	if age >= 4*time.Hour {
		value := returnValue
		return4h = &value
	}

	targetHit := currentPrice >= item.Target1
	invalidationHit := currentPrice <= item.Invalidation

	_, err := r.pool.Exec(ctx, `
		INSERT INTO signal_outcomes (
			signal_id, return_5m, return_15m, return_1h, return_4h,
			max_favorable_excursion, max_adverse_excursion,
			target_hit, invalidation_hit, evaluated_at
		) VALUES (
			$1::uuid,$2,$3,$4,$5,
			GREATEST($6::double precision, 0),
			LEAST($6::double precision, 0),
			$7,$8,$9
		)
		ON CONFLICT (signal_id) DO UPDATE SET
			return_5m = COALESCE(signal_outcomes.return_5m, EXCLUDED.return_5m),
			return_15m = COALESCE(signal_outcomes.return_15m, EXCLUDED.return_15m),
			return_1h = COALESCE(signal_outcomes.return_1h, EXCLUDED.return_1h),
			return_4h = COALESCE(signal_outcomes.return_4h, EXCLUDED.return_4h),
			max_favorable_excursion = GREATEST(
				COALESCE(signal_outcomes.max_favorable_excursion, 0),
				EXCLUDED.max_favorable_excursion
			),
			max_adverse_excursion = LEAST(
				COALESCE(signal_outcomes.max_adverse_excursion, 0),
				EXCLUDED.max_adverse_excursion
			),
			target_hit = COALESCE(signal_outcomes.target_hit, FALSE) OR EXCLUDED.target_hit,
			invalidation_hit = COALESCE(signal_outcomes.invalidation_hit, FALSE) OR EXCLUDED.invalidation_hit,
			evaluated_at = EXCLUDED.evaluated_at
	`,
		item.ID,
		return5m,
		return15m,
		return1h,
		return4h,
		returnValue,
		targetHit,
		invalidationHit,
		now,
	)
	if err != nil {
		return "", err
	}

	if targetHit || invalidationHit || now.After(item.ExpiresAt) || age >= 4*time.Hour {
		status := "CLOSED"
		if invalidationHit {
			status = "INVALIDATED"
		} else if now.After(item.ExpiresAt) && !targetHit {
			status = "EXPIRED"
		}
		tag, err := r.pool.Exec(ctx, `
			UPDATE signals
			SET status = $2, closed_at = $3
			WHERE id = $1::uuid AND closed_at IS NULL
		`, item.ID, status, now)
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() > 0 {
			return status, nil
		}
	}
	return "", nil
}

func (r *Repository) PerformanceSummary(ctx context.Context) (domain.PerformanceSummary, error) {
	var summary domain.PerformanceSummary
	err := r.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM signals) AS total_signals,
			COUNT(*) FILTER (WHERE returns ? '1h' AND (returns->'1h'->>'netReturnPct') IS NOT NULL) AS evaluated_signals,
			COUNT(*) FILTER (WHERE target_hit = TRUE AND returns ? '1h' AND (returns->'1h'->>'netReturnPct') IS NOT NULL) AS target_hits,
			COUNT(*) FILTER (WHERE invalidation_hit = TRUE AND returns ? '1h' AND (returns->'1h'->>'netReturnPct') IS NOT NULL) AS invalidation_hits,
			COALESCE(AVG((returns->'5m'->>'netReturnPct')::double precision) / 100, 0),
			COALESCE(AVG((returns->'15m'->>'netReturnPct')::double precision) / 100, 0),
			COALESCE(AVG((returns->'1h'->>'netReturnPct')::double precision) / 100, 0),
			COALESCE(AVG((returns->'4h'->>'netReturnPct')::double precision) / 100, 0),
			COALESCE(AVG(max_favorable_pct), 0),
			COALESCE(AVG(max_adverse_pct), 0)
		FROM signal_outcomes_v2
	`).Scan(
		&summary.TotalSignals,
		&summary.EvaluatedSignals,
		&summary.TargetHits,
		&summary.InvalidationHits,
		&summary.AverageReturn5m,
		&summary.AverageReturn15m,
		&summary.AverageReturn1h,
		&summary.AverageReturn4h,
		&summary.AverageMFE,
		&summary.AverageMAE,
	)
	if err != nil {
		return summary, err
	}
	if summary.EvaluatedSignals > 0 {
		summary.TargetHitRate = float64(summary.TargetHits) / float64(summary.EvaluatedSignals)
	}
	return summary, nil
}

func (r *Repository) SaveMarketSnapshot(ctx context.Context, snapshot market.PairSnapshot) error {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	recordedAt := time.Now().UTC().Truncate(time.Second)
	_, err = transaction.Exec(ctx, `
		INSERT INTO orderbook_metrics (
			symbol, recorded_at, mid_price, spread_bps, bid_depth_quote,
			ask_depth_quote, imbalance, spoof_score, removal_quote,
			last_update_id, is_synced
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (symbol, recorded_at) DO UPDATE SET
			mid_price = EXCLUDED.mid_price,
			spread_bps = EXCLUDED.spread_bps,
			bid_depth_quote = EXCLUDED.bid_depth_quote,
			ask_depth_quote = EXCLUDED.ask_depth_quote,
			imbalance = EXCLUDED.imbalance,
			spoof_score = EXCLUDED.spoof_score,
			removal_quote = EXCLUDED.removal_quote,
			last_update_id = EXCLUDED.last_update_id,
			is_synced = EXCLUDED.is_synced
	`,
		snapshot.Symbol,
		recordedAt,
		snapshot.Book.MidPrice,
		snapshot.Book.SpreadBPS,
		snapshot.Book.BidDepthQuote,
		snapshot.Book.AskDepthQuote,
		snapshot.Book.Imbalance,
		snapshot.Book.SpoofScore,
		snapshot.Book.RemovalQuote,
		snapshot.Book.LastUpdateID,
		snapshot.Book.Synced,
	)
	if err != nil {
		return err
	}

	for timeframe, values := range snapshot.Candles {
		if len(values) == 0 {
			continue
		}
		// Persist recent candles. Upserts keep the currently open candle current
		// and make reconnect backfills idempotent.
		start := len(values) - 3
		if start < 0 {
			start = 0
		}
		for _, candle := range values[start:] {
			_, err = transaction.Exec(ctx, `
				INSERT INTO candles (
					symbol, timeframe, open_time, open, high, low, close,
					base_volume, quote_volume, is_closed, updated_at
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())
				ON CONFLICT (symbol, timeframe, open_time) DO UPDATE SET
					open = EXCLUDED.open,
					high = EXCLUDED.high,
					low = EXCLUDED.low,
					close = EXCLUDED.close,
					base_volume = EXCLUDED.base_volume,
					quote_volume = EXCLUDED.quote_volume,
					is_closed = EXCLUDED.is_closed,
					updated_at = NOW()
			`,
				snapshot.Symbol,
				timeframe,
				candle.OpenTime,
				candle.Open,
				candle.High,
				candle.Low,
				candle.Close,
				candle.BaseVolume,
				candle.QuoteVolume,
				candle.Closed,
			)
			if err != nil {
				return err
			}
		}
	}

	return transaction.Commit(ctx)
}
