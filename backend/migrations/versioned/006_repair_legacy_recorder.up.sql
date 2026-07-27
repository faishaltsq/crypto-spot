-- Repairs databases initialized by legacy 004_recorder_and_outcomes.sql, which created then dropped these tables.
CREATE TABLE IF NOT EXISTS market_events (
    event_id UUID NOT NULL, event_type VARCHAR(32) NOT NULL, exchange VARCHAR(32) NOT NULL,
    symbol VARCHAR(32) NOT NULL, exchange_timestamp TIMESTAMPTZ NOT NULL,
    received_timestamp TIMESTAMPTZ NOT NULL, processed_timestamp TIMESTAMPTZ NOT NULL,
    connection_id VARCHAR(64) NOT NULL, sequence BIGINT NOT NULL, payload BYTEA NOT NULL,
    schema_version INT NOT NULL DEFAULT 1, compressed BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (exchange_timestamp, symbol, event_id)
);
CREATE INDEX IF NOT EXISTS idx_market_events_symbol_time ON market_events(symbol, exchange_timestamp DESC);
CREATE TABLE IF NOT EXISTS signal_outcomes_v2 (
    signal_id UUID PRIMARY KEY REFERENCES signals(id) ON DELETE CASCADE, symbol VARCHAR(32) NOT NULL,
    evaluated_at TIMESTAMPTZ NOT NULL, returns JSONB NOT NULL, max_favorable_pct DOUBLE PRECISION NOT NULL,
    max_adverse_pct DOUBLE PRECISION NOT NULL, target_hit BOOLEAN NOT NULL DEFAULT false,
    target_hit_at TIMESTAMPTZ, invalidation_hit BOOLEAN NOT NULL DEFAULT false, invalidation_hit_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_signal_outcomes_v2_symbol ON signal_outcomes_v2(symbol);
CREATE INDEX IF NOT EXISTS idx_signal_outcomes_v2_favorable ON signal_outcomes_v2(max_favorable_pct DESC);
CREATE TABLE IF NOT EXISTS signal_simulations (
    signal_id UUID PRIMARY KEY REFERENCES signals(id) ON DELETE CASCADE, symbol VARCHAR(32) NOT NULL,
    simulated_at TIMESTAMPTZ NOT NULL, base_entry_price DOUBLE PRECISION NOT NULL, fees JSONB NOT NULL,
    slippage_by_notional JSONB NOT NULL, capacity JSONB NOT NULL, avg_slippage_bps DOUBLE PRECISION NOT NULL,
    total_cost_bps DOUBLE PRECISION NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_signal_simulations_symbol ON signal_simulations(symbol);
CREATE INDEX IF NOT EXISTS idx_signal_simulations_cost ON signal_simulations(total_cost_bps);
