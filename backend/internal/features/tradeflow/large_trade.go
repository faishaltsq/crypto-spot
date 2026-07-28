package tradeflow

import (
	"sort"

	"github.com/example/crypto-spot-signal/internal/domain"
)

// LargeTradeThresholdUSDT flags a single trade as "large" once its quote
// notional crosses this value. This is a fixed, documented heuristic rather
// than a per-pair statistical threshold, matching the spec's request for
// deterministic large-trade detection.
const LargeTradeThresholdUSDT = 5000.0

// LargeTrades counts and sums large aggressive buy/sell trades by quote
// notional, never by raw base-token amount, so results are comparable across
// pairs with wildly different token prices.
func LargeTrades(trades []domain.Trade) (buyCount, sellCount int, buyNotional, sellNotional float64) {
	for _, t := range trades {
		if t.Quote < LargeTradeThresholdUSDT {
			continue
		}
		switch t.Side {
		case "buy":
			buyCount++
			buyNotional += t.Quote
		case "sell":
			sellCount++
			sellNotional += t.Quote
		}
	}
	return
}

// AverageTradeSize returns mean quote notional per trade, 0 when there are
// no trades.
func AverageTradeSize(trades []domain.Trade) float64 {
	if len(trades) == 0 {
		return 0
	}
	var total float64
	for _, t := range trades {
		total += t.Quote
	}
	return total / float64(len(trades))
}

// TradeFrequency returns trades per second across the observed window.
func TradeFrequency(trades []domain.Trade, windowSeconds float64) float64 {
	if windowSeconds <= 0 || len(trades) == 0 {
		return 0
	}
	return float64(len(trades)) / windowSeconds
}

// OldestNewest returns the earliest and latest trade timestamps, used to
// determine actual observation span rather than assuming the full requested
// window was covered.
func OldestNewest(trades []domain.Trade) (oldestUnix, newestUnix int64) {
	if len(trades) == 0 {
		return 0, 0
	}
	ordered := make([]domain.Trade, len(trades))
	copy(ordered, trades)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Timestamp.Before(ordered[j].Timestamp)
	})
	return ordered[0].Timestamp.Unix(), ordered[len(ordered)-1].Timestamp.Unix()
}
