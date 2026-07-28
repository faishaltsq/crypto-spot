package sell

import "github.com/example/crypto-spot-signal/internal/domain"

// BuildEvidence assembles structured supporting/contradicting evidence for a
// SELL signal, mirroring signals/engine.go's buildEvidence but built
// entirely from bearish/SELL-specific reason codes (domain/types.go).
func BuildEvidence(f FeatureSnapshot) domain.SignalEvidence {
	var supporting, contradicting []string

	if f.TradeFlow.AggressiveSellRatio >= 0.60 {
		supporting = append(supporting, domain.ReasonAggressiveSellPressure)
	}
	if f.TradeFlow.NegativeCVDSlope < 0 {
		supporting = append(supporting, domain.ReasonNegativeCVDSlope)
	}
	if f.TradeFlow.LargeSellTradeCount >= 3 {
		supporting = append(supporting, domain.ReasonLargeSellTrades)
	}
	if f.Walls.BidWallFailed {
		supporting = append(supporting, domain.ReasonBidWallFailure)
	}
	if f.Structure.SupportBroken && f.Structure.ClosedCandleConfirmed {
		supporting = append(supporting, domain.ReasonSupportBreakdown)
	}
	if f.Structure.LowerHighDetected && f.Structure.LowerLowDetected {
		supporting = append(supporting, domain.ReasonLowerHighLowerLow)
	}
	if f.Structure.ReclaimFailed {
		supporting = append(supporting, domain.ReasonFailedReclaim)
	}
	if f.Structure.BreakdownFollowThrough {
		supporting = append(supporting, domain.ReasonBreakdownFollowThrough)
	}
	if f.TrendAlignment <= -0.35 {
		supporting = append(supporting, domain.ReasonBearishMultiTFAlign)
	}

	if f.TradeFlow.SellExhaustion >= 0.5 {
		contradicting = append(contradicting, domain.ReasonSellPressureWeakened)
	}
	if f.TradeFlow.BuyRecovery >= 0.5 {
		contradicting = append(contradicting, domain.ReasonBuyPressureRecovered)
	}
	if f.Structure.SupportBroken && !f.Structure.ReclaimFailed && f.Structure.ReclaimAttempted {
		contradicting = append(contradicting, domain.ReasonSupportReclaimed)
	}
	if f.Walls.IcebergSuspected {
		contradicting = append(contradicting, domain.ReasonSuspectedIcebergWall)
	}
	if len(f.MissingFeatures) > 0 {
		contradicting = append(contradicting, domain.ReasonDataNotReady)
	}

	reasonCodes := append(append([]string{}, supporting...), contradicting...)

	return domain.SignalEvidence{
		SupportingEvidence:    supporting,
		ContradictingEvidence: contradicting,
		ReasonCodes:           reasonCodes,
	}
}
