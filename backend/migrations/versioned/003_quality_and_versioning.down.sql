ALTER TABLE signals DROP COLUMN IF EXISTS data_quality_score;
ALTER TABLE signals DROP COLUMN IF EXISTS threshold_detail;
ALTER TABLE signals DROP COLUMN IF EXISTS evidence;
ALTER TABLE signals DROP COLUMN IF EXISTS signal_version;
DROP TABLE IF EXISTS data_quality_history;
