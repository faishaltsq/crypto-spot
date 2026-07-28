-- 012_sell_signals.up.sql

ALTER TABLE signals 
ADD COLUMN IF NOT EXISTS sell_score DOUBLE PRECISION,
ADD COLUMN IF NOT EXISTS sell_rule_score DOUBLE PRECISION,
ADD COLUMN IF NOT EXISTS sell_model_probability DOUBLE PRECISION,
ADD COLUMN IF NOT EXISTS sell_base_threshold DOUBLE PRECISION,
ADD COLUMN IF NOT EXISTS sell_final_threshold DOUBLE PRECISION,
ADD COLUMN IF NOT EXISTS trade_flow_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
ADD COLUMN IF NOT EXISTS bearish_structure_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
ADD COLUMN IF NOT EXISTS spoof_analysis JSONB NOT NULL DEFAULT '{}'::jsonb,
ADD COLUMN IF NOT EXISTS supporting_evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS contradicting_evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS passed_rules JSONB NOT NULL DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS failed_rules JSONB NOT NULL DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS blocked_rules JSONB NOT NULL DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS invalidation_condition JSONB NOT NULL DEFAULT '{}'::jsonb,
ADD COLUMN IF NOT EXISTS invalidation_reason VARCHAR(64),
ADD COLUMN IF NOT EXISTS directional_outcome JSONB NOT NULL DEFAULT '{}'::jsonb,
ADD COLUMN IF NOT EXISTS avoid_entry_outcome JSONB NOT NULL DEFAULT '{}'::jsonb,
ADD COLUMN IF NOT EXISTS exit_warning_outcome JSONB NOT NULL DEFAULT '{}'::jsonb,
ADD COLUMN IF NOT EXISTS take_profit_outcome JSONB NOT NULL DEFAULT '{}'::jsonb;

-- New outcome table specifically for SELL directional accuracy (not short profit)
CREATE TABLE IF NOT EXISTS sell_signal_outcomes (
    signal_id UUID PRIMARY KEY REFERENCES signals(id) ON DELETE CASCADE,
    symbol VARCHAR(32) NOT NULL,
    evaluated_at TIMESTAMPTZ NOT NULL,
    directional_return DOUBLE PRECISION NOT NULL DEFAULT 0,
    directional_accuracy BOOLEAN NOT NULL DEFAULT false,
    max_downside_move DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_adverse_upside_move DOUBLE PRECISION NOT NULL DEFAULT 0,
    support_reclaim BOOLEAN NOT NULL DEFAULT false,
    breakdown_follow_through BOOLEAN NOT NULL DEFAULT false,
    invalidated BOOLEAN NOT NULL DEFAULT false,
    time_to_invalidation INTERVAL,
    avoid_entry_effectiveness DOUBLE PRECISION,
    exit_warning_effectiveness DOUBLE PRECISION,
    take_profit_effectiveness DOUBLE PRECISION
);
CREATE INDEX IF NOT EXISTS idx_sell_signal_outcomes_symbol ON sell_signal_outcomes(symbol);
CREATE INDEX IF NOT EXISTS idx_sell_signal_outcomes_evaluated_at ON sell_signal_outcomes(evaluated_at DESC);
