CREATE TABLE system_settings (
    setting_key VARCHAR(100) PRIMARY KEY,
    setting_value JSONB NOT NULL,
    reload_mode VARCHAR(32) NOT NULL CHECK (reload_mode IN ('immediate', 'new_signal', 'resubscription', 'restart')),
    restart_required BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE system_setting_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version INTEGER NOT NULL UNIQUE,
    settings JSONB NOT NULL,
    changed_by VARCHAR(128) NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE system_setting_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    setting_key VARCHAR(100) NOT NULL,
    previous_value JSONB,
    new_value JSONB NOT NULL,
    changed_by VARCHAR(128) NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason TEXT NOT NULL DEFAULT '',
    result VARCHAR(32) NOT NULL,
    effective_at TIMESTAMPTZ,
    restart_required BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_system_setting_audit_logs_changed_at ON system_setting_audit_logs (changed_at DESC);

CREATE TABLE user_preferences (
    user_id UUID PRIMARY KEY,
    preferences JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
