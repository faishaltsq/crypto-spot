package config

import "time"

// SellConfig holds every environment-driven tunable for the SELL signal engine
// (protective sell, take-profit, avoid-entry). It is loaded and validated at
// startup, mirroring the style of the existing BUY EngineConfig validation.
type SellConfig struct {
	Enabled                       bool
	RequireClosedCandle           bool
	RequireExecutedTradeConfirm   bool
	RequireOrderbookSync          bool
	RequireFailedReclaim          bool

	SetupScore           float64
	ConfirmScore         float64
	MinRuleScore         float64
	MinModelProbability  float64
	MinDataQuality       float64
	MaxSpoofScore        float64
	MinTradeflowScore    float64
	MinTimeframeAlignment float64

	PairCooldown        time.Duration
	MaxActiveGlobal     int
	MaxActivePerPair    int
	MaxActivePerCluster int
	MaxNewPerMinute     int

	MinTradeCount        int
	MinTradeNotionalUSDT float64
	MinObservationSeconds int

	TakeProfitEnabled              bool
	TakeProfitSetupScore           float64
	TakeProfitConfirmScore         float64
	TakeProfitMinOverextension     float64
	TakeProfitMinExhaustion        float64
	TakeProfitRequireCVDDivergence bool
}
