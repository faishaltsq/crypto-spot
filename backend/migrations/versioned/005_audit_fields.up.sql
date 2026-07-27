ALTER TABLE signals ADD COLUMN IF NOT EXISTS data_quality_status VARCHAR(20) NOT NULL DEFAULT 'UNAVAILABLE';
ALTER TABLE signals ADD COLUMN IF NOT EXISTS data_source VARCHAR(20) NOT NULL DEFAULT 'GATE';
ALTER TABLE signals ADD COLUMN IF NOT EXISTS missing_features JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE signals ADD COLUMN IF NOT EXISTS blocked_reasons JSONB NOT NULL DEFAULT '[]'::jsonb;
CREATE INDEX IF NOT EXISTS idx_signals_dq_status ON signals (data_quality_status);
CREATE INDEX IF NOT EXISTS idx_signals_source ON signals (data_source);
