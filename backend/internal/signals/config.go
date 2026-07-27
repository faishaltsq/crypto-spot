package signals

import (
	"fmt"
	"time"
)

type EngineConfig struct {
	SetupScore             float64
	ConfirmScore           float64
	MinRuleScore           float64
	MinModelProbability    float64
	MinDataQuality         float64
	MaxSpoofScore          float64
	MaxActiveGlobal        int
	MaxActivePerPair       int
	MaxActivePerCluster    int
	GlobalSignalRatePerMin int
	PairCooldown           time.Duration
}

func (c EngineConfig) Validate() error {
	if c.ConfirmScore < c.SetupScore {
		return configError("ConfirmScore must be at least SetupScore")
	}
	if c.SetupScore < 0 || c.ConfirmScore < 0 || c.MinRuleScore < 0 {
		return configError("signal scores must be non-negative")
	}
	if c.MinModelProbability < 0 || c.MinModelProbability > 1 {
		return configError("MinModelProbability must be between 0 and 1")
	}
	if c.MinDataQuality < 0 || c.MinDataQuality > 100 {
		return configError("MinDataQuality must be between 0 and 100")
	}
	if c.MaxSpoofScore < 0 || c.MaxSpoofScore > 100 {
		return configError("MaxSpoofScore must be between 0 and 100")
	}
	if c.MaxActiveGlobal < 0 || c.MaxActivePerPair < 0 || c.MaxActivePerCluster < 0 || c.GlobalSignalRatePerMin < 0 {
		return configError("signal limits must be non-negative")
	}
	if c.PairCooldown < 0 {
		return configError("PairCooldown must be non-negative")
	}
	return nil
}

func configError(message string) error {
	return fmt.Errorf("invalid signal engine config: %s", message)
}
