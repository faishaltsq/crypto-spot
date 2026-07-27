package threshold

const (
	ReasonTierBPenalty                = "TIER_B_PENALTY"
	ReasonTierCPenalty                = "TIER_C_PENALTY"
	ReasonWeakUptrendPenalty          = "WEAK_UPTREND_PENALTY"
	ReasonRangingPenalty              = "RANGING_PENALTY"
	ReasonHighVolatilityRegimePenalty = "HIGH_VOLATILITY_REGIME_PENALTY"
	ReasonMarketSellOffPenalty        = "MARKET_SELL_OFF_PENALTY"
	ReasonPumpConditionPenalty        = "PUMP_CONDITION_PENALTY"
	ReasonUndeterminedRegimePenalty   = "UNDETERMINED_REGIME_PENALTY"
	ReasonMissingRegime               = "MISSING_REGIME"
	ReasonMissingVolatility           = "MISSING_VOLATILITY"
	ReasonHighVolatilityPenalty       = "HIGH_VOLATILITY_PERCENTILE_PENALTY"
	ReasonModerateSpoofPenalty        = "MODERATE_SPOOF_PENALTY"
	ReasonModerateLiquidityPenalty    = "MODERATE_LIQUIDITY_PENALTY"
	ReasonModerateCorrelationPenalty  = "MODERATE_CORRELATION_PENALTY"
	ReasonHighCorrelationPenalty      = "HIGH_CLUSTER_CONCENTRATION_PENALTY"
	ReasonHighSpoofBlock              = "HIGH_SPOOF_RISK_BLOCK"
	ReasonLowLiquidityBlock           = "LOW_LIQUIDITY_BLOCK"
	ReasonDataQualityBlock            = "DATA_QUALITY_BLOCK"
	ReasonStrongDowntrendBlock        = "STRONG_DOWNTREND_BLOCK"
	ReasonBelowFinalThreshold         = "BELOW_FINAL_THRESHOLD"
)
