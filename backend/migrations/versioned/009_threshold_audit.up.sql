CREATE INDEX IF NOT EXISTS idx_signals_threshold_version
ON signals ((threshold_detail->>'thresholdVersion'));

CREATE INDEX IF NOT EXISTS idx_signals_threshold_blocked
ON signals (((threshold_detail->>'blockedByThreshold')::boolean));
