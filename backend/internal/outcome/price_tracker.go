package outcome

import (
	"math"
	"sync"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
)

// PriceObservation tracks the highest and lowest prices seen since a signal was created.
// This is used for precise Maximum Favorable Excursion (MFE) and Maximum Adverse Excursion (MAE) calculations.
type PriceObservation struct {
	High float64
	Low  float64
}

// PriceTracker monitors real-time market data to track excursions for active signals.
type PriceTracker struct {
	mu           sync.RWMutex
	observations map[string]PriceObservation // Key: Signal ID
}

// NewPriceTracker creates a new price tracker.
func NewPriceTracker() *PriceTracker {
	return &PriceTracker{
		observations: make(map[string]PriceObservation),
	}
}

// AddCandidate registers a new signal to track.
func (pt *PriceTracker) AddCandidate(candidate Candidate) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	// Initialize with entry price
	pt.observations[candidate.SignalID] = PriceObservation{
		High: candidate.EntryPrice,
		Low:  candidate.EntryPrice,
	}
}

// RemoveCandidate stops tracking a signal.
func (pt *PriceTracker) RemoveCandidate(signalID string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	delete(pt.observations, signalID)
}

// UpdateFromTrades updates observations based on recent trades.
// In a real system, this should ideally be called directly from the WebSocket feed
// for tick-by-tick accuracy, but can also be updated periodically from market store.
func (pt *PriceTracker) UpdateFromTrades(signalID string, trades []domain.Trade) {
	if len(trades) == 0 {
		return
	}

	pt.mu.Lock()
	defer pt.mu.Unlock()

	obs, exists := pt.observations[signalID]
	if !exists {
		return
	}

	for _, t := range trades {
		if t.Price > obs.High {
			obs.High = t.Price
		}
		if t.Price < obs.Low {
			obs.Low = t.Price
		}
	}
	pt.observations[signalID] = obs
}

// UpdateFromCandles updates observations based on candle high/lows.
// This is a coarser fallback if tick-level tracking missed some data.
func (pt *PriceTracker) UpdateFromCandles(signalID string, candles []domain.Candle) {
	if len(candles) == 0 {
		return
	}

	pt.mu.Lock()
	defer pt.mu.Unlock()

	obs, exists := pt.observations[signalID]
	if !exists {
		return
	}

	for _, c := range candles {
		if c.High > obs.High {
			obs.High = c.High
		}
		if c.Low < obs.Low {
			obs.Low = c.Low
		}
	}
	pt.observations[signalID] = obs
}

// GetObservation returns the current MFE/MAE prices for a signal.
func (pt *PriceTracker) GetObservation(signalID string) (PriceObservation, bool) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	obs, ok := pt.observations[signalID]
	return obs, ok
}

// CalculateExcursionPct calculates the percentage excursion given a base price.
func CalculateExcursionPct(base, compare float64) float64 {
	if base <= 0 {
		return 0
	}
	return (compare - base) / base * 100
}

// ExtractRelevantTrades filters a trade list for trades that occurred after a specific time.
func ExtractRelevantTrades(trades []domain.Trade, since time.Time) []domain.Trade {
	var result []domain.Trade
	for _, t := range trades {
		if t.Timestamp.After(since) {
			result = append(result, t)
		}
	}
	return result
}

// ExtractRelevantCandles filters a candle list for candles that cover time after a specific time.
func ExtractRelevantCandles(candles []domain.Candle, since time.Time) []domain.Candle {
	var result []domain.Candle
	for _, c := range candles {
		// If candle opened after, or was active during 'since'
		if c.OpenTime.After(since) || math.Abs(c.OpenTime.Sub(since).Seconds()) < 60 { // assuming 1m candles
			result = append(result, c)
		}
	}
	return result
}
