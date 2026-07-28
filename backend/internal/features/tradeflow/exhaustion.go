package tradeflow

// SellExhaustion scores momentum exhaustion for sell pressure: sell volume
// declining while price stays flat/recovers suggests sellers are running out
// (used for SELL invalidation "SELL_PRESSURE_WEAKENED" and take-profit
// "momentum exhaustion" evidence, mirrored for the buy side).
//
// recentSellVolume/priorSellVolume are consecutive equal-length windows.
func SellExhaustion(recentSellVolume, priorSellVolume, recentDeltaRatio, priorDeltaRatio float64) float64 {
	if priorSellVolume <= 0 {
		return 0
	}
	volumeDecline := clamp(1-(recentSellVolume/priorSellVolume), 0, 1)
	deltaImprovement := clamp((recentDeltaRatio-priorDeltaRatio)/2, 0, 1)
	return clamp(volumeDecline*0.6+deltaImprovement*0.4, 0, 1)
}

// BuyRecovery is the mirror of SellExhaustion, used for SELL invalidation
// reason BUY_PRESSURE_RECOVERED: aggressive buy volume picking back up after
// a decline.
func BuyRecovery(recentBuyVolume, priorBuyVolume, recentDeltaRatio, priorDeltaRatio float64) float64 {
	if priorBuyVolume <= 0 {
		if recentBuyVolume > 0 {
			return 1
		}
		return 0
	}
	volumeGrowth := clamp((recentBuyVolume/priorBuyVolume)-1, 0, 1)
	deltaImprovement := clamp((recentDeltaRatio-priorDeltaRatio)/2, 0, 1)
	return clamp(volumeGrowth*0.6+deltaImprovement*0.4, 0, 1)
}

// MomentumExhaustion is the take-profit specific exhaustion signal: buying
// momentum weakening while price is still elevated (positive volume delta
// shrinking). Distinct from SellExhaustion because take-profit exhaustion is
// about waning buy pressure at a high, not waning sell pressure at a low.
func MomentumExhaustion(recentBuyVolumeDeltaRatio, priorBuyVolumeDeltaRatio float64) float64 {
	if priorBuyVolumeDeltaRatio <= 0 {
		return 0
	}
	decline := clamp(1-(recentBuyVolumeDeltaRatio/priorBuyVolumeDeltaRatio), 0, 1)
	return decline
}
