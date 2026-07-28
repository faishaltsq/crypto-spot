package sell

import (
	"log"

	"github.com/example/crypto-spot-signal/internal/domain"
)

// TakeProfitScore is distinct from the bearish RuleScore: it measures
// overextension + momentum exhaustion at a high, which is evidence for
// "this rally is stretched and buying pressure is fading" rather than
// "this is a confirmed downtrend". A pair can score high on take-profit
// evidence while scoring low on bearish RuleScore (early distribution),
// which is exactly the distinction the spec asks for.
func TakeProfitScore(f FeatureSnapshot) float64 {
	overextension := overextensionScore(f)
	exhaustion := exhaustionScore(f)
	return clamp(overextension*0.55+exhaustion*0.45, 0, 100)
}

func overextensionScore(f FeatureSnapshot) float64 {
	// TrendAlignment > 0 means bullish-aligned; the further above the
	// confirm threshold, the more "stretched" the rally is considered.
	if f.TrendAlignment <= 0 {
		return 0
	}
	return clamp(f.TrendAlignment*100, 0, 100)
}

func exhaustionScore(f FeatureSnapshot) float64 {
	buyMomentumFading := clamp(1-f.TradeFlow.AggressiveBuyRatio, 0, 1) * 100
	return buyMomentumFading
}

// evaluateTakeProfit fires TAKE_PROFIT_SUGGESTED for an open BUY position
// when overextension + momentum exhaustion evidence crosses configured
// thresholds. Independent of the bearish protective-SELL thresholds.
func (e *Engine) evaluateTakeProfit(f FeatureSnapshot, buyCtx ActiveBuyContext, shouldRateLimit bool) (*domain.Signal, bool) {
	if !e.cfg.TakeProfitEnabled {
		return nil, false
	}
	score := TakeProfitScore(f)
	if score < e.cfg.TakeProfitSetupScore {
		return nil, false
	}
	if overextensionScore(f) < e.cfg.TakeProfitMinOverextension {
		return nil, false
	}
	if exhaustionScore(f) < e.cfg.TakeProfitMinExhaustion {
		return nil, false
	}
	if e.cfg.TakeProfitRequireCVDDivergence && f.TradeFlow.NegativeCVDSlope >= 0 {
		return nil, false
	}

	if !e.allowIssue(f.Symbol, domain.TakeProfitSuggested, shouldRateLimit) {
		return nil, false
	}

	sig := baseSignal(f, domain.TakeProfitSuggested, "ACTIVE", score, zeroThreshold(score))
	if score >= e.cfg.TakeProfitConfirmScore {
		sig.Status = "CONFIRMED"
	} else {
		sig.Status = "SETUP"
	}
	sig.Reasons = append(sig.Reasons, domain.ReasonOverextendedRally, domain.ReasonBuyMomentumExhaustion)
	if buyCtx.ActiveSignalID != "" {
		sig.Reasons = append(sig.Reasons, "RELATED_POSITION:"+buyCtx.ActiveSignalID)
	}
	log.Printf("[sell] ISSUED %s type=%s score=%.1f (take profit for position %s)", f.Symbol, domain.TakeProfitSuggested, score, buyCtx.ActiveSignalID)
	return sig, true
}
