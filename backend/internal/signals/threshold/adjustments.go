package threshold

func volatilityAdjustment(bands []VolatilityAdjustment, percentile float64) float64 {
	if percentile < 0 {
		return 0
	}
	adjustment := 0.0
	for _, band := range bands {
		if percentile >= band.MinPercentile {
			adjustment = band.Adjustment
		}
	}
	return adjustment
}

func tierReason(t Tier) string {
	if t == TierB {
		return ReasonTierBPenalty
	}
	if t == TierC {
		return ReasonTierCPenalty
	}
	return ""
}

func regimeReason(r Regime) string {
	switch r {
	case WeakUptrend:
		return ReasonWeakUptrendPenalty
	case Ranging:
		return ReasonRangingPenalty
	case HighVolatility:
		return ReasonHighVolatilityRegimePenalty
	case MarketSellOff:
		return ReasonMarketSellOffPenalty
	case PumpCondition:
		return ReasonPumpConditionPenalty
	case Undetermined:
		return ReasonUndeterminedRegimePenalty
	default:
		return ""
	}
}
