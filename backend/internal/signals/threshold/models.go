package threshold

type Tier string

const (
	TierA Tier = "A"
	TierB Tier = "B"
	TierC Tier = "C"
)

type Regime string

const (
	StrongUptrend   Regime = "STRONG_UPTREND"
	WeakUptrend     Regime = "WEAK_UPTREND"
	Ranging         Regime = "RANGING"
	HighVolatility  Regime = "HIGH_VOLATILITY"
	MarketSellOff   Regime = "MARKET_SELL_OFF"
	PumpCondition   Regime = "PUMP_CONDITION"
	StrongDowntrend Regime = "STRONG_DOWNTREND"
	Undetermined    Regime = "UNDETERMINED"
)

type SpoofRisk string

const (
	SpoofLow      SpoofRisk = "LOW"
	SpoofModerate SpoofRisk = "MODERATE"
	SpoofHigh     SpoofRisk = "HIGH"
)

type Liquidity string

const (
	LiquidityHealthy  Liquidity = "HEALTHY"
	LiquidityModerate Liquidity = "MODERATE"
	LiquidityLow      Liquidity = "LOW"
)

type Correlation string

const (
	CorrelationIndependent   Correlation = "INDEPENDENT"
	CorrelationModerateBurst Correlation = "MODERATE_BURST"
	CorrelationHighCluster   Correlation = "HIGH_CLUSTER"
)

type DataQuality string

const (
	DataQualityValid      DataQuality = "VALID"
	DataQualityDegraded   DataQuality = "DEGRADED"
	DataQualityStale      DataQuality = "STALE"
	DataQualityUnsynced   DataQuality = "UNSYNCED"
	DataQualityIncomplete DataQuality = "INCOMPLETE"
	DataQualityBlocked    DataQuality = "BLOCKED"
)

type Input struct {
	Tier                 Tier
	Regime               Regime
	VolatilityPercentile float64
	SpoofRisk            SpoofRisk
	Liquidity            Liquidity
	Correlation          Correlation
	DataQuality          DataQuality
	ActualScore          float64
}

type Result struct {
	ThresholdVersion      string   `json:"thresholdVersion"`
	BaseThreshold         float64  `json:"baseThreshold"`
	TierAdjustment        float64  `json:"tierAdjustment"`
	RegimeAdjustment      float64  `json:"regimeAdjustment"`
	VolatilityAdjustment  float64  `json:"volatilityAdjustment"`
	SpoofAdjustment       float64  `json:"spoofAdjustment"`
	LiquidityAdjustment   float64  `json:"liquidityAdjustment"`
	CorrelationAdjustment float64  `json:"correlationAdjustment"`
	FinalThreshold        float64  `json:"finalThreshold"`
	ActualScore           float64  `json:"actualScore"`
	Passed                bool     `json:"passed"`
	Blocked               bool     `json:"blockedByThreshold"`
	ReasonCodes           []string `json:"thresholdReasonCodes"`
}
