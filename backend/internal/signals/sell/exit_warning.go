package sell

import (
	"log"

	"github.com/example/crypto-spot-signal/internal/domain"
)

// evaluateExitWarning fires when the caller reports an OPEN BUY position
// (buyCtx.HasActivePosition) and bearish evidence suggests risk is rising.
// This is advisory only: the SELL engine never closes, modifies, or writes
// to a BUY signal's lifecycle state. It is purely an independent warning
// signal the UI/notification layer can surface next to the open position.
func (e *Engine) evaluateExitWarning(f FeatureSnapshot, ruleScore float64, buyCtx ActiveBuyContext, shouldRateLimit bool) (*domain.Signal, bool) {
	if ruleScore < e.cfg.MinRuleScore {
		return nil, false
	}
	if f.TradeFlow.AggressiveSellRatio < 0.55 && !f.Structure.LowerHighDetected {
		return nil, false
	}

	if !e.allowIssue(f.Symbol, domain.ExitWarningSignal, shouldRateLimit) {
		return nil, false
	}

	sig := baseSignal(f, domain.ExitWarningSignal, "ACTIVE", ruleScore, zeroThreshold(ruleScore))
	if buyCtx.ActiveSignalID != "" {
		sig.Reasons = append(sig.Reasons, "RELATED_POSITION:"+buyCtx.ActiveSignalID)
	}
	log.Printf("[sell] ISSUED %s type=%s score=%.1f (exit warning for position %s)", f.Symbol, domain.ExitWarningSignal, ruleScore, buyCtx.ActiveSignalID)
	return sig, true
}
