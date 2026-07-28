package tradeflow

import (
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
)

// Aggregator computes multi-window SellFlowSnapshots for a symbol from its
// full trade history plus a price context per window.
type Aggregator struct {
	sampleCfg SampleConfig
}

func NewAggregator(sampleCfg SampleConfig) *Aggregator {
	return &Aggregator{sampleCfg: sampleCfg}
}

// ComputeWindow builds one SellFlowSnapshot for a given window duration.
// trades must already be the full retained trade history for the symbol
// (unfiltered); this function performs the time-window slicing itself so
// prior-window comparisons stay consistent.
func (a *Aggregator) ComputeWindow(symbol string, allTrades []domain.Trade, window time.Duration, price PriceContext) SellFlowSnapshot {
	now := time.Now()
	cutoff := now.Add(-window)
	priorCutoff := now.Add(-2 * window)

	var current, prior []domain.Trade
	for _, t := range allTrades {
		if !t.Timestamp.Before(cutoff) {
			current = append(current, t)
		} else if !t.Timestamp.Before(priorCutoff) {
			prior = append(prior, t)
		}
	}
	return Compute(symbol, window, current, prior, price, a.sampleCfg)
}

// ComputeAllWindows builds a MultiWindowSnapshot across every standard
// observation window.
func (a *Aggregator) ComputeAllWindows(symbol string, allTrades []domain.Trade, price PriceContext) MultiWindowSnapshot {
	out := MultiWindowSnapshot{Symbol: symbol, Windows: make(map[string]SellFlowSnapshot)}
	for label, window := range StandardWindows() {
		out.Windows[label] = a.ComputeWindow(symbol, allTrades, window, price)
	}
	return out
}
