package threshold

func Calculate(cfg Config, input Input) Result {
	result := Result{ThresholdVersion: cfg.Version, BaseThreshold: cfg.BaseThreshold, ActualScore: input.ActualScore}
	regime := input.Regime
	if regime == "" {
		regime = Undetermined
		result.ReasonCodes = append(result.ReasonCodes, ReasonMissingRegime)
	}
	result.TierAdjustment = cfg.Tier[input.Tier]
	result.RegimeAdjustment = cfg.Regime[regime]
	result.VolatilityAdjustment = volatilityAdjustment(cfg.Volatility, input.VolatilityPercentile)
	result.SpoofAdjustment = cfg.Spoof[input.SpoofRisk]
	result.LiquidityAdjustment = cfg.Liquidity[input.Liquidity]
	result.CorrelationAdjustment = cfg.Correlation[input.Correlation]
	result.FinalThreshold = result.BaseThreshold + result.TierAdjustment + result.RegimeAdjustment + result.VolatilityAdjustment + result.SpoofAdjustment + result.LiquidityAdjustment + result.CorrelationAdjustment
	if reason := tierReason(input.Tier); reason != "" {
		result.ReasonCodes = append(result.ReasonCodes, reason)
	}
	if reason := regimeReason(regime); reason != "" {
		result.ReasonCodes = append(result.ReasonCodes, reason)
	}
	if input.VolatilityPercentile < 0 {
		result.ReasonCodes = append(result.ReasonCodes, ReasonMissingVolatility)
	} else if result.VolatilityAdjustment > 0 {
		result.ReasonCodes = append(result.ReasonCodes, ReasonHighVolatilityPenalty)
	}
	if input.SpoofRisk == SpoofModerate {
		result.ReasonCodes = append(result.ReasonCodes, ReasonModerateSpoofPenalty)
	}
	if input.Liquidity == LiquidityModerate {
		result.ReasonCodes = append(result.ReasonCodes, ReasonModerateLiquidityPenalty)
	}
	if input.Correlation == CorrelationModerateBurst {
		result.ReasonCodes = append(result.ReasonCodes, ReasonModerateCorrelationPenalty)
	} else if input.Correlation == CorrelationHighCluster {
		result.ReasonCodes = append(result.ReasonCodes, ReasonHighCorrelationPenalty)
	}
	if cfg.BlockRegimes[regime] {
		result.Blocked = true
		result.ReasonCodes = append(result.ReasonCodes, ReasonStrongDowntrendBlock)
	}
	if cfg.BlockSpoof[input.SpoofRisk] {
		result.Blocked = true
		result.ReasonCodes = append(result.ReasonCodes, ReasonHighSpoofBlock)
	}
	if cfg.BlockLiquidity[input.Liquidity] {
		result.Blocked = true
		result.ReasonCodes = append(result.ReasonCodes, ReasonLowLiquidityBlock)
	}
	if cfg.BlockDataQuality[input.DataQuality] {
		result.Blocked = true
		result.ReasonCodes = append(result.ReasonCodes, ReasonDataQualityBlock)
	}
	result.Passed = !result.Blocked && result.ActualScore >= result.FinalThreshold
	if !result.Blocked && !result.Passed {
		result.ReasonCodes = append(result.ReasonCodes, ReasonBelowFinalThreshold)
	}
	return result
}
