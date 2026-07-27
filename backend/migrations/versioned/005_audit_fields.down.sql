ALTER TABLE signals DROP COLUMN IF EXISTS blocked_reasons;
ALTER TABLE signals DROP COLUMN IF EXISTS missing_features;
ALTER TABLE signals DROP COLUMN IF EXISTS data_source;
ALTER TABLE signals DROP COLUMN IF EXISTS data_quality_status;
