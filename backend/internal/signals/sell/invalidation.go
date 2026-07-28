package sell

import "github.com/example/crypto-spot-signal/internal/domain"

// InvalidationCheck evaluates whether an ACTIVE SELL/TAKE_PROFIT signal
// should be invalidated based on fresh evidence, per the spec's four
// invalidation conditions. It never uses price targets (that is the BUY
// engine's model) — SELL invalidation is entirely evidence-based: the
// bearish thesis itself stopped holding.
type InvalidationCheck struct {
	Invalidated bool
	Reason      string // one of domain.Invalidation* constants
}

// CheckInvalidation inspects the current FeatureSnapshot for a symbol with
// an active SELL-family signal and returns the first invalidation condition
// that applies, in priority order (support reclaim is checked first because
// it's the strongest, least ambiguous signal that the bearish thesis broke).
func CheckInvalidation(f FeatureSnapshot) InvalidationCheck {
	if f.Structure.SupportBroken && f.Structure.ReclaimAttempted && !f.Structure.ReclaimFailed {
		return InvalidationCheck{Invalidated: true, Reason: domain.InvalidationSupportReclaimed}
	}
	if f.TradeFlow.BuyRecovery >= 0.6 {
		return InvalidationCheck{Invalidated: true, Reason: domain.InvalidationBuyPressureRecovered}
	}
	if f.TradeFlow.SellExhaustion >= 0.6 {
		return InvalidationCheck{Invalidated: true, Reason: domain.InvalidationSellPressureWeakened}
	}
	if bullishDivergence(f) {
		return InvalidationCheck{Invalidated: true, Reason: domain.InvalidationBullishDivergence}
	}
	return InvalidationCheck{Invalidated: false}
}

// bullishDivergence detects price making a lower low while the CVD slope
// turns positive (or sell ratio collapses) — classic bullish divergence
// that contradicts continued bearish conviction.
func bullishDivergence(f FeatureSnapshot) bool {
	return f.Structure.LowerLowDetected && f.TradeFlow.NegativeCVDSlope == 0 && f.TradeFlow.AggressiveSellRatio < 0.45
}
