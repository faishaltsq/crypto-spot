package sell

import (
	"time"

	"github.com/example/crypto-spot-signal/internal/config"
	"github.com/example/crypto-spot-signal/internal/signals/threshold"
)

// EngineConfig is the resolved runtime configuration for the SELL engine,
// converted once from config.SellConfig at startup (see cmd/server/main.go).
type EngineConfig struct {
	Enabled bool

	RequireClosedCandle         bool
	RequireExecutedTradeConfirm bool
	RequireOrderbookSync        bool
	RequireFailedReclaim        bool

	SetupScore            float64
	ConfirmScore          float64
	MinRuleScore          float64
	MinModelProbability   float64
	MinDataQuality        float64
	MaxSpoofScore         float64
	MinTradeflowScore     float64
	MinTimeframeAlignment float64

	PairCooldown        time.Duration
	MaxActiveGlobal     int
	MaxActivePerPair    int
	MaxActivePerCluster int
	MaxNewPerMinute     int

	TakeProfitEnabled              bool
	TakeProfitSetupScore           float64
	TakeProfitConfirmScore         float64
	TakeProfitMinOverextension     float64
	TakeProfitMinExhaustion        float64
	TakeProfitRequireCVDDivergence bool

	// ThresholdConfig reuses the existing direction-agnostic dynamic
	// threshold calculator (signals/threshold) so tier/regime/volatility/
	// spoof/liquidity/correlation adjustments apply identically to SELL.
	ThresholdConfig threshold.Config
}

// FromAppConfig converts the loaded config.SellConfig into an EngineConfig,
// wiring in a dedicated threshold.Config so SELL threshold adjustments never
// share mutable state with the BUY engine's threshold.Config.
func FromAppConfig(c config.SellConfig) EngineConfig {
	return EngineConfig{
		Enabled:                      c.Enabled,
		RequireClosedCandle:          c.RequireClosedCandle,
		RequireExecutedTradeConfirm:  c.RequireExecutedTradeConfirm,
		RequireOrderbookSync:         c.RequireOrderbookSync,
		RequireFailedReclaim:         c.RequireFailedReclaim,
		SetupScore:                   c.SetupScore,
		ConfirmScore:                 c.ConfirmScore,
		MinRuleScore:                 c.MinRuleScore,
		MinModelProbability:          c.MinModelProbability,
		MinDataQuality:               c.MinDataQuality,
		MaxSpoofScore:                c.MaxSpoofScore,
		MinTradeflowScore:            c.MinTradeflowScore,
		MinTimeframeAlignment:        c.MinTimeframeAlignment,
		PairCooldown:                 c.PairCooldown,
		MaxActiveGlobal:              c.MaxActiveGlobal,
		MaxActivePerPair:             c.MaxActivePerPair,
		MaxActivePerCluster:          c.MaxActivePerCluster,
		MaxNewPerMinute:              c.MaxNewPerMinute,
		TakeProfitEnabled:            c.TakeProfitEnabled,
		TakeProfitSetupScore:         c.TakeProfitSetupScore,
		TakeProfitConfirmScore:       c.TakeProfitConfirmScore,
		TakeProfitMinOverextension:   c.TakeProfitMinOverextension,
		TakeProfitMinExhaustion:      c.TakeProfitMinExhaustion,
		TakeProfitRequireCVDDivergence: c.TakeProfitRequireCVDDivergence,
		ThresholdConfig:              threshold.DefaultConfig(c.ConfirmScore),
	}
}
