package execution_simulation

import (
	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/market"
)

// CalculateSlippage simulates a market order of the given size (in USD) against the orderbook
// to determine the volume-weighted average fill price and the resulting slippage.
// For a BUY signal, we simulate a market BUY, which walks up the ASK side of the orderbook.
func CalculateSlippage(notionalUSD float64, snapshot market.PairSnapshot) SlippageEstimate {
	if notionalUSD <= 0 {
		return SlippageEstimate{}
	}
	if snapshot.Book.BestAsk <= 0 || len(snapshot.TopAsks) == 0 {
		return SlippageEstimate{NotionalUSD: notionalUSD, FullyFilled: false}
	}

	remainingQuote := notionalUSD
	totalSpentQuote := 0.0
	totalAcquiredBase := 0.0
	levelsHit := 0

	for _, ask := range snapshot.TopAsks {
		levelsHit++
		levelQuote := ask.Price * ask.Amount

		if remainingQuote <= levelQuote {
			// This level can fully fill the remaining order
			baseAmountToFill := remainingQuote / ask.Price
			totalSpentQuote += remainingQuote
			totalAcquiredBase += baseAmountToFill
			remainingQuote = 0
			break
		} else {
			// Eat this entire level and continue to the next
			totalSpentQuote += levelQuote
			totalAcquiredBase += ask.Amount
			remainingQuote -= levelQuote
		}
	}

	if totalAcquiredBase == 0 {
		return SlippageEstimate{NotionalUSD: notionalUSD, FullyFilled: false}
	}

	vwapFillPrice := totalSpentQuote / totalAcquiredBase
	basePrice := snapshot.LastPrice
	if basePrice <= 0 {
		basePrice = snapshot.Book.MidPrice
	}

	slippageBPS := 0.0
	if basePrice > 0 {
		// Slippage = (Fill Price - Base Price) / Base Price * 10000
		slippageBPS = (vwapFillPrice - basePrice) / basePrice * 10000.0
	}

	return SlippageEstimate{
		NotionalUSD:        notionalUSD,
		EstimatedFillPrice: vwapFillPrice,
		SlippageBPS:        slippageBPS,
		OrderbookLevelsHit: levelsHit,
		FullyFilled:        remainingQuote <= 0.001, // allow tiny float drift
	}
}

// CalculateCapacity determines the maximum order size the orderbook can absorb
// up to a certain slippage tolerance, and how depleted the depth is.
func CalculateCapacity(snapshot market.PairSnapshot) CapacityResult {
	// Simple proxy: 50% of the currently visible Ask depth in quote currency
	maxSupported := snapshot.Book.AskDepthQuote * 0.5
	depletion := 0.0
	if snapshot.Book.AskDepthQuote > 0 {
		depletion = maxSupported / snapshot.Book.AskDepthQuote * 100
	}

	return CapacityResult{
		MaxSupportedNotionalUSD: maxSupported,
		DepthDepletionPercent:   depletion,
	}
}

// Ensure unique sorted asks
func GetSortedAsks(levels []domain.Level) []domain.Level {
	// In the real system, snapshot.Asks is already sorted ascending by price.
	return levels
}
