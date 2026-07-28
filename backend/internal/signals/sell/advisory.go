package sell

import "github.com/example/crypto-spot-signal/internal/signals/threshold"

// zeroThreshold builds a minimal threshold.Result for advisory SELL signal
// types (AVOID_ENTRY, EXIT_WARNING, TAKE_PROFIT_SUGGESTED) that are gated by
// their own dedicated evidence checks rather than the dynamic
// tier/regime/volatility threshold calculator used by protective SELL and
// the BUY engine. It still records the actual rule score for audit purposes.
func zeroThreshold(ruleScore float64) threshold.Result {
	return threshold.Result{
		ThresholdVersion: "advisory-v1",
		ActualScore:      ruleScore,
		Passed:           true,
	}
}
