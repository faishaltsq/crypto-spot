package execution_simulation

// CalculateFee estimates the fee costs for a round-trip trade (entry and exit).
// We assume worst-case (taker fee for both) or configurable. 
// For conservative paper trading simulation, assuming taker fees is safest.
func CalculateFee(notionalUSD float64, config FeeConfig) FeeBreakdown {
	// If config is 0, provide realistic defaults (e.g., Binance/Gate.io basic tiers: 0.1% = 10 BPS)
	takerBPS := config.TakerBPS
	if takerBPS <= 0 {
		takerBPS = 10.0 // 10 bps = 0.1%
	}

	entryFeeUSD := notionalUSD * (takerBPS / 10000.0)
	exitFeeUSD := notionalUSD * (takerBPS / 10000.0) // assumes exit at same notional for simplicity

	return FeeBreakdown{
		EntryFeeUSD: entryFeeUSD,
		EntryFeeBPS: takerBPS,
		ExitFeeUSD:  exitFeeUSD,
		ExitFeeBPS:  takerBPS,
		TotalFeeUSD: entryFeeUSD + exitFeeUSD,
		TotalFeeBPS: takerBPS * 2,
	}
}
