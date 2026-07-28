package sell

import "math"

// RuleScore computes the SELL-side deterministic rule score (0-100) from a
// FeatureSnapshot, mirroring the weighting style of features/engine.go's
// BUY ruleScore but built entirely from bearish evidence: trade flow
// pressure, bearish structure, and orderbook wall failure, penalized by
// spoofing and data quality.
func RuleScore(f FeatureSnapshot) float64 {
	// Bearish trend component: TrendAlignment is negative when timeframes
	// agree on a downtrend (see features/engine.go weightedTrend). We invert
	// and scale it the same way the BUY engine scales its positive trendScore.
	trendScore := clamp(50-f.TrendAlignment*50, 0, 100)

	// Trade flow component: aggressive sell ratio and negative CVD slope.
	flowScore := clamp(
		f.TradeFlow.AggressiveSellRatio*60+
			negativeSlopeScore(f.TradeFlow.NegativeCVDSlope)*20+
			clamp(f.TradeFlow.SellExhaustion, 0, 1)*-10+ // exhaustion REDUCES sell score
			clamp(float64(f.TradeFlow.LargeSellTradeCount)/5, 0, 1)*30,
		0, 100)

	// Bearish structure component: already 0-100.
	structureScore := f.Structure.StructureScore

	// Wall-failure component: bid wall failing under sell pressure is strong
	// confirming evidence; a healthy, unfailed bid wall pulls the score down.
	wallScore := 50.0
	if f.Walls.BidWallFailed {
		wallScore = 50 + f.Walls.BidWallFailureConfidence*50
	} else if f.Walls.BidWallDetected {
		wallScore = 30 // a holding bid wall is mild counter-evidence
	}

	base := trendScore*0.30 + flowScore*0.30 + structureScore*0.25 + wallScore*0.15

	spoofPenalty := f.SpoofScoreRaw * 0.18
	dataQualityFactor := clamp(f.DataQualityScore/100, 0, 1)

	score := clamp(base-spoofPenalty, 0, 100) * dataQualityFactor
	return score
}

// negativeSlopeScore maps a negative CVD slope onto 0-1 confidence via a
// bounded log scale, avoiding unbounded blowup for pairs with huge notional.
func negativeSlopeScore(slope float64) float64 {
	if slope >= 0 {
		return 0
	}
	magnitude := math.Log1p(-slope)
	return clamp(magnitude/10, 0, 1)
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
