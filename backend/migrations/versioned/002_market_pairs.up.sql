CREATE TABLE IF NOT EXISTS market_pairs (
    symbol VARCHAR(20) PRIMARY KEY, rank_position INTEGER, rank_score NUMERIC(12, 6), tier SMALLINT,
    qualified BOOLEAN NOT NULL DEFAULT FALSE, quote_volume_24h NUMERIC(30, 8), spread_bps NUMERIC(12, 4),
    depth_score NUMERIC(12, 6), activity_score NUMERIC(12, 6), selection_reason JSONB,
    rejection_reason VARCHAR(255), is_active BOOLEAN NOT NULL DEFAULT TRUE,
    universe_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_market_pairs_qualified_rank ON market_pairs (qualified, rank_position);
CREATE INDEX IF NOT EXISTS idx_market_pairs_tier_active ON market_pairs (tier, is_active);
CREATE INDEX IF NOT EXISTS idx_market_pairs_quote_volume ON market_pairs (quote_volume_24h DESC);
