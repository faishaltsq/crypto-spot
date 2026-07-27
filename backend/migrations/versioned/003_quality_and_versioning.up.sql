CREATE TABLE IF NOT EXISTS data_quality_history (
    symbol VARCHAR(50) NOT NULL, evaluated_at TIMESTAMPTZ NOT NULL, score DOUBLE PRECISION NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'BLOCKED', reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
    signal_allowed BOOLEAN NOT NULL DEFAULT FALSE, PRIMARY KEY (symbol, evaluated_at)
);
SELECT create_hypertable('data_quality_history', 'evaluated_at', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_data_quality_symbol_time ON data_quality_history (symbol, evaluated_at DESC);
CREATE INDEX IF NOT EXISTS idx_data_quality_status ON data_quality_history (status, evaluated_at DESC);
ALTER TABLE signals ADD COLUMN IF NOT EXISTS signal_version JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE signals ADD COLUMN IF NOT EXISTS evidence JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE signals ADD COLUMN IF NOT EXISTS threshold_detail JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE signals ADD COLUMN IF NOT EXISTS data_quality_score DOUBLE PRECISION NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_signals_data_quality ON signals (data_quality_score DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_signals_version ON signals USING gin (signal_version);
