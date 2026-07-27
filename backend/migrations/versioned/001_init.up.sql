CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS candles (
    symbol VARCHAR(50) NOT NULL, timeframe VARCHAR(20) NOT NULL, open_time TIMESTAMPTZ NOT NULL,
    open DOUBLE PRECISION NOT NULL, high DOUBLE PRECISION NOT NULL, low DOUBLE PRECISION NOT NULL,
    close DOUBLE PRECISION NOT NULL, base_volume DOUBLE PRECISION NOT NULL DEFAULT 0,
    quote_volume DOUBLE PRECISION NOT NULL DEFAULT 0, is_closed BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), PRIMARY KEY (symbol, timeframe, open_time)
);
SELECT create_hypertable('candles', 'open_time', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_candles_symbol_timeframe_time ON candles (symbol, timeframe, open_time DESC);

CREATE TABLE IF NOT EXISTS orderbook_metrics (
    symbol VARCHAR(50) NOT NULL, recorded_at TIMESTAMPTZ NOT NULL, mid_price DOUBLE PRECISION NOT NULL DEFAULT 0,
    spread_bps DOUBLE PRECISION NOT NULL DEFAULT 0, bid_depth_quote DOUBLE PRECISION NOT NULL DEFAULT 0,
    ask_depth_quote DOUBLE PRECISION NOT NULL DEFAULT 0, imbalance DOUBLE PRECISION NOT NULL DEFAULT 0,
    spoof_score DOUBLE PRECISION NOT NULL DEFAULT 0, removal_quote DOUBLE PRECISION NOT NULL DEFAULT 0,
    last_update_id BIGINT NOT NULL DEFAULT 0, is_synced BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (symbol, recorded_at)
);
SELECT create_hypertable('orderbook_metrics', 'recorded_at', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_orderbook_metrics_symbol_time ON orderbook_metrics (symbol, recorded_at DESC);

CREATE TABLE IF NOT EXISTS market_features (
    symbol VARCHAR(50) NOT NULL, calculated_at TIMESTAMPTZ NOT NULL, price DOUBLE PRECISION NOT NULL DEFAULT 0,
    spread_bps DOUBLE PRECISION NOT NULL DEFAULT 0, bid_depth_quote DOUBLE PRECISION NOT NULL DEFAULT 0,
    ask_depth_quote DOUBLE PRECISION NOT NULL DEFAULT 0, orderbook_imbalance DOUBLE PRECISION NOT NULL DEFAULT 0,
    spoof_score DOUBLE PRECISION NOT NULL DEFAULT 0, relative_volume_1m DOUBLE PRECISION NOT NULL DEFAULT 0,
    volume_delta_ratio_1m DOUBLE PRECISION NOT NULL DEFAULT 0, trend_alignment DOUBLE PRECISION NOT NULL DEFAULT 0,
    rule_score DOUBLE PRECISION NOT NULL DEFAULT 0, data_quality_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    status VARCHAR(40) NOT NULL, trend_by_timeframe JSONB NOT NULL DEFAULT '{}'::jsonb,
    reasons JSONB NOT NULL DEFAULT '[]'::jsonb, risk_flags JSONB NOT NULL DEFAULT '[]'::jsonb,
    feature_payload JSONB NOT NULL, PRIMARY KEY (symbol, calculated_at)
);
SELECT create_hypertable('market_features', 'calculated_at', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_market_features_symbol_time ON market_features (symbol, calculated_at DESC);
CREATE INDEX IF NOT EXISTS idx_market_features_score_time ON market_features (rule_score DESC, calculated_at DESC);

CREATE TABLE IF NOT EXISTS signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), symbol VARCHAR(50) NOT NULL, signal_type VARCHAR(40) NOT NULL,
    status VARCHAR(40) NOT NULL, primary_timeframe VARCHAR(20) NOT NULL, entry_price DOUBLE PRECISION NOT NULL,
    invalidation_price DOUBLE PRECISION NOT NULL, target_price_1 DOUBLE PRECISION NOT NULL,
    target_price_2 DOUBLE PRECISION NOT NULL, rule_score DOUBLE PRECISION NOT NULL,
    ai_review JSONB NOT NULL DEFAULT '{}'::jsonb, reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
    risk_flags JSONB NOT NULL DEFAULT '[]'::jsonb, feature_snapshot JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL, expires_at TIMESTAMPTZ NOT NULL, closed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_signals_created_at ON signals (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_signals_symbol_created ON signals (symbol, created_at DESC);
CREATE TABLE IF NOT EXISTS signal_outcomes (
    signal_id UUID PRIMARY KEY REFERENCES signals(id) ON DELETE CASCADE, return_5m DOUBLE PRECISION,
    return_15m DOUBLE PRECISION, return_1h DOUBLE PRECISION, return_4h DOUBLE PRECISION,
    max_favorable_excursion DOUBLE PRECISION, max_adverse_excursion DOUBLE PRECISION,
    target_hit BOOLEAN, invalidation_hit BOOLEAN, evaluated_at TIMESTAMPTZ
);
