package tradeflow

import "math"

// SellingAbsorption estimates how much aggressive sell volume the market
// absorbed without a proportional price decline. High absorption (bids
// absorbing sell flow) means the observed selling did not translate into a
// matching price drop — a bearish sell signal without this backing evidence
// is weaker, and it directly feeds `bid_wall_failure`/`ask_wall_failure`
// cross-checks against the spoofing package.
//
// priceChangePct is the percent price move over the window (negative for a
// decline). sellVolumeUSDT is the aggressive sell notional in the same
// window. expectedDeclinePctPerUSDT is a calibration constant representing
// how much price would be expected to move per unit of sell notional in a
// "normal" (non-absorbed) market for this pair; callers derive it from
// recent realized volatility divided by typical volume, so absorption is
// relative to the pair's own liquidity rather than a fixed constant.
func SellingAbsorption(priceChangePct, sellVolumeUSDT, expectedDeclinePctPerUSDT float64) float64 {
	if sellVolumeUSDT <= 0 || expectedDeclinePctPerUSDT <= 0 {
		return 0
	}
	expectedDecline := -sellVolumeUSDT * expectedDeclinePctPerUSDT
	if expectedDecline >= 0 {
		return 0
	}
	// absorption = how much smaller the actual decline was than expected.
	// 1.0 = fully absorbed (no decline despite heavy selling), 0 = no absorption.
	ratio := 1 - (priceChangePct / expectedDecline)
	return clamp(ratio, 0, 1)
}

// BuyingAbsorption is the mirror computation for aggressive buy volume
// against asks, used by the take-profit "ask replenishment" evidence.
func BuyingAbsorption(priceChangePct, buyVolumeUSDT, expectedRisePctPerUSDT float64) float64 {
	if buyVolumeUSDT <= 0 || expectedRisePctPerUSDT <= 0 {
		return 0
	}
	expectedRise := buyVolumeUSDT * expectedRisePctPerUSDT
	if expectedRise <= 0 {
		return 0
	}
	ratio := 1 - (priceChangePct / expectedRise)
	return clamp(ratio, 0, 1)
}

func clamp(v, min, max float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
