package threshold

import (
	"encoding/json"
	"fmt"
	"os"
)

type VolatilityAdjustment struct {
	MinPercentile float64 `json:"minPercentile"`
	Adjustment    float64 `json:"adjustment"`
}

type Config struct {
	Version          string                  `json:"version"`
	BaseThreshold    float64                 `json:"baseThreshold"`
	Tier             map[Tier]float64        `json:"tier"`
	Regime           map[Regime]float64      `json:"regime"`
	Volatility       []VolatilityAdjustment  `json:"volatility"`
	Spoof            map[SpoofRisk]float64   `json:"spoof"`
	Liquidity        map[Liquidity]float64   `json:"liquidity"`
	Correlation      map[Correlation]float64 `json:"correlation"`
	BlockRegimes     map[Regime]bool         `json:"blockRegimes"`
	BlockSpoof       map[SpoofRisk]bool      `json:"blockSpoof"`
	BlockLiquidity   map[Liquidity]bool      `json:"blockLiquidity"`
	BlockDataQuality map[DataQuality]bool    `json:"blockDataQuality"`
}

func DefaultConfig(base float64) Config {
	return Config{
		Version: "threshold-v1", BaseThreshold: base,
		Tier:             map[Tier]float64{TierA: 0, TierB: 3, TierC: 7},
		Regime:           map[Regime]float64{StrongUptrend: 0, WeakUptrend: 2, Ranging: 4, HighVolatility: 5, MarketSellOff: 10, PumpCondition: 8, StrongDowntrend: 15, Undetermined: 5},
		Volatility:       []VolatilityAdjustment{{MinPercentile: 0, Adjustment: 0}, {MinPercentile: 80, Adjustment: 2}},
		Spoof:            map[SpoofRisk]float64{SpoofLow: 0, SpoofModerate: 4, SpoofHigh: 0},
		Liquidity:        map[Liquidity]float64{LiquidityHealthy: 0, LiquidityModerate: 3, LiquidityLow: 0},
		Correlation:      map[Correlation]float64{CorrelationIndependent: 0, CorrelationModerateBurst: 2, CorrelationHighCluster: 5},
		BlockRegimes:     map[Regime]bool{StrongDowntrend: true},
		BlockSpoof:       map[SpoofRisk]bool{SpoofHigh: true},
		BlockLiquidity:   map[Liquidity]bool{LiquidityLow: true},
		BlockDataQuality: map[DataQuality]bool{DataQualityDegraded: true, DataQualityStale: true, DataQualityUnsynced: true, DataQualityIncomplete: true, DataQualityBlocked: true},
	}
}

// LoadConfig permits one versioned JSON document to override every adjustment.
// Keeping policy in one environment value avoids scattered threshold constants.
func LoadConfig(base float64) (Config, error) {
	cfg := DefaultConfig(base)
	raw := os.Getenv("SIGNAL_THRESHOLD_CONFIG_JSON")
	if raw == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse SIGNAL_THRESHOLD_CONFIG_JSON: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
