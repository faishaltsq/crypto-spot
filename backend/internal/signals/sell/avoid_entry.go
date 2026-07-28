package sell

import (
	"log"

	"github.com/example/crypto-spot-signal/internal/domain"
)

// evaluateAvoidEntry fires when a BUY_SETUP/BUY_CONFIRMED_CANDIDATE exists
// for a pair (per buyCtx, supplied by the orchestrator) but SELL-side
// bearish evidence contradicts entering. This never blocks the BUY engine
// directly — signals/engine.go remains the sole authority over BUY
// lifecycle — it only surfaces an advisory AVOID_ENTRY signal alongside it.
func (e *Engine) evaluateAvoidEntry(f FeatureSnapshot, ruleScore float64, shouldRateLimit bool) (*domain.Signal, bool) {
	if ruleScore < e.cfg.MinRuleScore {
		return nil, false
	}
	if f.TradeFlow.AggressiveSellRatio < 0.55 && !f.Structure.SupportBroken {
		return nil, false // not enough bearish conviction to warn against a BUY candidate
	}

	if !e.allowIssue(f.Symbol, domain.AvoidEntrySignal, shouldRateLimit) {
		return nil, false
	}

	sig := baseSignal(f, domain.AvoidEntrySignal, "ACTIVE", ruleScore, zeroThreshold(ruleScore))
	log.Printf("[sell] ISSUED %s type=%s score=%.1f (avoid entry)", f.Symbol, domain.AvoidEntrySignal, ruleScore)
	return sig, true
}
