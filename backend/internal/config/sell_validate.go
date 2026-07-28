package config

import (
	"fmt"
	"time"
)

func durationMinutes(key string, fallbackMinutes int) time.Duration {
	return time.Duration(getInt(key, fallbackMinutes)) * time.Minute
}

// validateSellConfig fails startup fast on invalid SELL/take-profit config,
// mirroring the single-guard-clause style used for the BUY signal engine in
// config.go and signals/config.go.
func validateSellConfig(c SellConfig) error {
	if c.ConfirmScore < c.SetupScore {
		return fmt.Errorf("invalid sell engine configuration: SELL_CONFIRM_SCORE must be >= SELL_SETUP_SCORE")
	}
	if c.SetupScore < 0 || c.ConfirmScore < 0 || c.MinRuleScore < 0 {
		return fmt.Errorf("invalid sell engine configuration: sell scores must be non-negative")
	}
	if c.MinModelProbability < 0 || c.MinModelProbability > 1 {
		return fmt.Errorf("invalid sell engine configuration: SELL_MIN_MODEL_PROBABILITY must be between 0 and 1")
	}
	if c.MinDataQuality < 0 || c.MinDataQuality > 100 {
		return fmt.Errorf("invalid sell engine configuration: SELL_MIN_DATA_QUALITY must be between 0 and 100")
	}
	if c.MaxSpoofScore < 0 || c.MaxSpoofScore > 100 {
		return fmt.Errorf("invalid sell engine configuration: SELL_MAX_SPOOF_SCORE must be between 0 and 100")
	}
	if c.MinTradeflowScore < 0 || c.MinTradeflowScore > 100 {
		return fmt.Errorf("invalid sell engine configuration: SELL_MIN_TRADEFLOW_SCORE must be between 0 and 100")
	}
	if c.MinTimeframeAlignment < 0 || c.MinTimeframeAlignment > 100 {
		return fmt.Errorf("invalid sell engine configuration: SELL_MIN_TIMEFRAME_ALIGNMENT must be between 0 and 100")
	}
	if c.MaxActiveGlobal < 0 || c.MaxActivePerPair < 0 || c.MaxActivePerCluster < 0 || c.MaxNewPerMinute < 0 {
		return fmt.Errorf("invalid sell engine configuration: sell active-signal limits must be non-negative")
	}
	if c.PairCooldown < 0 {
		return fmt.Errorf("invalid sell engine configuration: SELL_PAIR_COOLDOWN_MINUTES must be non-negative")
	}
	if c.MinTradeCount < 0 || c.MinTradeNotionalUSDT < 0 || c.MinObservationSeconds < 0 {
		return fmt.Errorf("invalid sell engine configuration: sell minimum-sample thresholds must be non-negative")
	}
	if c.TakeProfitConfirmScore < c.TakeProfitSetupScore {
		return fmt.Errorf("invalid sell engine configuration: TAKE_PROFIT_CONFIRM_SCORE must be >= TAKE_PROFIT_SETUP_SCORE")
	}
	if c.TakeProfitMinOverextension < 0 || c.TakeProfitMinOverextension > 100 {
		return fmt.Errorf("invalid sell engine configuration: TAKE_PROFIT_MIN_OVEREXTENSION_SCORE must be between 0 and 100")
	}
	if c.TakeProfitMinExhaustion < 0 || c.TakeProfitMinExhaustion > 100 {
		return fmt.Errorf("invalid sell engine configuration: TAKE_PROFIT_MIN_EXHAUSTION_SCORE must be between 0 and 100")
	}
	return nil
}
