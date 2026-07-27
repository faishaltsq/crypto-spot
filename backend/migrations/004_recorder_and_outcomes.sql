-- Migration 004: Market Recorder & Signal Outcomes
-- Up

-- 1. Market Events table for the Recorder
CREATE TABLE market_events (
    event_id UUID NOT NULL,
    event_type VARCHAR(32) NOT NULL,
    exchange VARCHAR(32) NOT NULL,
    symbol VARCHAR(32) NOT NULL,
    exchange_timestamp TIMESTAMPTZ NOT NULL,
    received_timestamp TIMESTAMPTZ NOT NULL,
    processed_timestamp TIMESTAMPTZ NOT NULL,
    connection_id VARCHAR(64) NOT NULL,
    sequence BIGINT NOT NULL,
    payload BYTEA NOT NULL,
    schema_version INT NOT NULL DEFAULT 1,
    compressed BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (exchange_timestamp, symbol, event_id)
);

-- Note: We use exchange_timestamp as part of the primary key for TimescaleDB chunking.
-- If TimescaleDB is enabled:
-- SELECT create_hypertable('market_events', 'exchange_timestamp', chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE);
-- SELECT add_retention_policy('market_events', INTERVAL '7 days');

CREATE INDEX idx_market_events_symbol_time ON market_events(symbol, exchange_timestamp DESC);

-- 2. Signal Outcomes (Multi-Horizon Evaluation)
CREATE TABLE signal_outcomes_v2 (
    signal_id UUID PRIMARY KEY REFERENCES signals(id) ON DELETE CASCADE,
    symbol VARCHAR(32) NOT NULL,
    evaluated_at TIMESTAMPTZ NOT NULL,
    returns JSONB NOT NULL,
    max_favorable_pct DOUBLE PRECISION NOT NULL,
    max_adverse_pct DOUBLE PRECISION NOT NULL,
    target_hit BOOLEAN NOT NULL DEFAULT false,
    target_hit_at TIMESTAMPTZ,
    invalidation_hit BOOLEAN NOT NULL DEFAULT false,
    invalidation_hit_at TIMESTAMPTZ
);

CREATE INDEX idx_signal_outcomes_v2_symbol ON signal_outcomes_v2(symbol);
CREATE INDEX idx_signal_outcomes_v2_favorable ON signal_outcomes_v2(max_favorable_pct DESC);

-- 3. Execution Simulation (Fees and Slippage)
CREATE TABLE signal_simulations (
    signal_id UUID PRIMARY KEY REFERENCES signals(id) ON DELETE CASCADE,
    symbol VARCHAR(32) NOT NULL,
    simulated_at TIMESTAMPTZ NOT NULL,
    base_entry_price DOUBLE PRECISION NOT NULL,
    fees JSONB NOT NULL,
    slippage_by_notional JSONB NOT NULL,
    capacity JSONB NOT NULL,
    avg_slippage_bps DOUBLE PRECISION NOT NULL,
    total_cost_bps DOUBLE PRECISION NOT NULL
);

CREATE INDEX idx_signal_simulations_symbol ON signal_simulations(symbol);
CREATE INDEX idx_signal_simulations_cost ON signal_simulations(total_cost_bps);

-- Down
DROP TABLE IF EXISTS signal_simulations;
DROP TABLE IF EXISTS signal_outcomes_v2;
DROP TABLE IF EXISTS market_events;
