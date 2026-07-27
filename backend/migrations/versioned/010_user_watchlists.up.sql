CREATE TABLE user_watchlists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    name VARCHAR(120) NOT NULL CHECK (char_length(trim(name)) > 0),
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, name)
);

CREATE UNIQUE INDEX idx_user_watchlists_default ON user_watchlists (user_id) WHERE is_default;

CREATE TABLE user_watchlist_pairs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    watchlist_id UUID NOT NULL REFERENCES user_watchlists(id) ON DELETE CASCADE,
    symbol VARCHAR(32) NOT NULL,
    position INTEGER NOT NULL DEFAULT 0 CHECK (position >= 0),
    is_favorite BOOLEAN NOT NULL DEFAULT FALSE,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    preferred_timeframe VARCHAR(16) NOT NULL DEFAULT '15m',
    minimum_signal_score DOUBLE PRECISION NOT NULL DEFAULT 85 CHECK (minimum_signal_score BETWEEN 0 AND 100),
    risk_levels TEXT[] NOT NULL DEFAULT ARRAY['LOW', 'MEDIUM'],
    signal_types TEXT[] NOT NULL DEFAULT ARRAY['BUY_CONFIRMED'],
    notification_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    tags TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (watchlist_id, symbol)
);

CREATE INDEX idx_user_watchlist_pairs_watchlist_position ON user_watchlist_pairs (watchlist_id, position);

CREATE TABLE user_pair_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    watchlist_pair_id UUID NOT NULL REFERENCES user_watchlist_pairs(id) ON DELETE CASCADE,
    note TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, watchlist_pair_id)
);

CREATE TABLE user_pair_alert_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    watchlist_pair_id UUID NOT NULL REFERENCES user_watchlist_pairs(id) ON DELETE CASCADE,
    notification_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    minimum_signal_score DOUBLE PRECISION NOT NULL DEFAULT 85 CHECK (minimum_signal_score BETWEEN 0 AND 100),
    preferred_timeframe VARCHAR(16) NOT NULL DEFAULT '15m',
    risk_levels TEXT[] NOT NULL DEFAULT ARRAY['LOW', 'MEDIUM'],
    signal_types TEXT[] NOT NULL DEFAULT ARRAY['BUY_CONFIRMED'],
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    timezone VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, watchlist_pair_id)
);
