CREATE TABLE ai_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id UUID REFERENCES signals(id) ON DELETE SET NULL,
    pair VARCHAR(32) NOT NULL,
    timeframe VARCHAR(16) NOT NULL,
    provider VARCHAR(32) NOT NULL,
    model VARCHAR(128) NOT NULL,
    decision VARCHAR(16) NOT NULL CHECK (decision IN ('CONFIRM', 'REJECT', 'WAIT', 'UNAVAILABLE')),
    confidence DOUBLE PRECISION NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    summary VARCHAR(600) NOT NULL,
    supporting_reason_codes JSONB NOT NULL DEFAULT '[]'::jsonb,
    contradicting_reason_codes JSONB NOT NULL DEFAULT '[]'::jsonb,
    risk_flags JSONB NOT NULL DEFAULT '[]'::jsonb,
    latency_ms INTEGER NOT NULL DEFAULT 0 CHECK (latency_ms >= 0),
    fallback BOOLEAN NOT NULL DEFAULT FALSE,
    fallback_reason VARCHAR(64),
    provider_error_code VARCHAR(64),
    prompt_version VARCHAR(64) NOT NULL,
    schema_version VARCHAR(64) NOT NULL,
    reviewed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ai_reviews_signal_id ON ai_reviews(signal_id);
CREATE INDEX idx_ai_reviews_reviewed_at ON ai_reviews(reviewed_at DESC);
