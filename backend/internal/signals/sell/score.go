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
	//
	// When NO wall event is observed at all (neither failed nor detected), the
	// wall component carries zero information — a hardcoded neutral 50 would
	// otherwise silently cap this 15% slice at half its range and drag the
	// total below SetupScore even when trend+flow+structure are strongly
	// bearish. In that case we drop the wall component and renormalize the
	// remaining weights so absent orderbook-wall data neither helps nor hurts.
	const (
		wTrend     = 0.30
		wFlow      = 0.30
		wStructure = 0.25
		wWall      = 0.15
	)
	var base float64
	if f.Walls.BidWallFailed {
		wallScore := 50 + f.Walls.BidWallFailureConfidence*50
		base = trendScore*wTrend + flowScore*wFlow + structureScore*wStructure + wallScore*wWall
	} else if f.Walls.BidWallDetected {
		wallScore := 30.0 // a holding bid wall is mild counter-evidence
		base = trendScore*wTrend + flowScore*wFlow + structureScore*wStructure + wallScore*wWall
	} else {
		// No wall signal: renormalize trend/flow/structure to fill 100%.
		total := wTrend + wFlow + wStructure
		base = (trendScore*wTrend + flowScore*wFlow + structureScore*wStructure) / total
	}

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
