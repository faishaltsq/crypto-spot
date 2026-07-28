-- 012_sell_signals.down.sql

ALTER TABLE signals 
DROP COLUMN IF EXISTS sell_score,
DROP COLUMN IF EXISTS sell_rule_score,
DROP COLUMN IF EXISTS sell_model_probability,
DROP COLUMN IF EXISTS sell_base_threshold,
DROP COLUMN IF EXISTS sell_final_threshold,
DROP COLUMN IF EXISTS trade_flow_snapshot,
DROP COLUMN IF EXISTS bearish_structure_snapshot,
DROP COLUMN IF EXISTS spoof_analysis,
DROP COLUMN IF EXISTS supporting_evidence,
DROP COLUMN IF EXISTS contradicting_evidence,
DROP COLUMN IF EXISTS passed_rules,
DROP COLUMN IF EXISTS failed_rules,
DROP COLUMN IF EXISTS blocked_rules,
DROP COLUMN IF EXISTS invalidation_condition,
DROP COLUMN IF EXISTS invalidation_reason,
DROP COLUMN IF EXISTS directional_outcome,
DROP COLUMN IF EXISTS avoid_entry_outcome,
DROP COLUMN IF EXISTS exit_warning_outcome,
DROP COLUMN IF EXISTS take_profit_outcome;

DROP TABLE IF EXISTS sell_signal_outcomes;
