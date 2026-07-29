package signals

import (
	"fmt"
	"time"

	"github.com/example/crypto-spot-signal/internal/config"
)

// EngineConfig is the resolved runtime configuration for the BUY engine.
// It holds ONLY the fields the BUY engine actually consumes. BUY routes
// score and data-quality decisions through the dynamic threshold
// calculator (signals/threshold), so it deliberately does not carry
// SetupScore / MinRuleScore / MinDataQuality gate fields the way the SELL
// engine does — those have no consumption point in BUY's threshold-based
// architecture and were dead config before this refactor.
//
// ConfirmScore, MaxSpoofScore, MinTrendAlignment, PairCooldown and
// MaxNewPerMinute are live gates read on the hot path. The three
// MaxActive* limits are carried for parity with the SELL engine's config
// surface and Admin Settings, but are not yet enforced on the hot path
// (no engine enforces them today); real enforcement is deferred to a
// later fase that will cover BUY and SELL symmetrically.
type EngineConfig struct {
	ConfirmScore      float64
	MaxSpoofScore     float64
	MinTrendAlignment float64

	MaxActiveGlobal     int
	MaxActivePerPair    int
	MaxActivePerCluster int
	MaxNewPerMinute     int
	PairCooldown        time.Duration
}

func (c EngineConfig) Validate() error {
	if c.ConfirmScore < 0 {
		return configError("ConfirmScore must be non-negative")
	}
	if c.MaxSpoofScore < 0 || c.MaxSpoofScore > 100 {
		return configError("MaxSpoofScore must be between 0 and 100")
	}
	if c.MinTrendAlignment < -1 || c.MinTrendAlignment > 1 {
		return configError("MinTrendAlignment must be between -1 and 1")
	}
	if c.MaxActiveGlobal < 0 || c.MaxActivePerPair < 0 || c.MaxActivePerCluster < 0 || c.MaxNewPerMinute < 0 {
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

// FromAppConfig resolves the flat config.Config signal fields into the BUY
// EngineConfig, mirroring sell.FromAppConfig. This is the single mapping
// point between stored/env configuration and the engine's live gates, used
// both at startup (New) and on live reload (SetConfig) so Admin Settings
// changes actually take effect at runtime.
func FromAppConfig(c *config.Config) EngineConfig {
	return EngineConfig{
		ConfirmScore:        c.SignalConfirmScore,
		MaxSpoofScore:       c.SignalMaxSpoofScore,
		MinTrendAlignment:   c.SignalMinTrendAlignment,
		MaxActiveGlobal:     c.SignalMaxActiveGlobal,
		MaxActivePerPair:    c.SignalMaxActivePerPair,
		MaxActivePerCluster: c.SignalMaxActiveCluster,
		MaxNewPerMinute:     c.SignalMaxNewPerMinute,
		PairCooldown:        c.SignalPairCooldown,
	}
}
