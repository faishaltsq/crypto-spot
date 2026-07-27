package outcome

import (
	"context"
	"log"
	"time"

	"github.com/example/crypto-spot-signal/internal/market"
)

// Storage interface for reading candidates and writing results.
type Storage interface {
	ActiveCandidates(ctx context.Context) ([]Candidate, error)
	SaveOutcome(ctx context.Context, result Result) error
}

// Service manages the evaluation of signal outcomes over time.
type Service struct {
	storage      Storage
	marketStore  *market.Store
	tracker      *PriceTracker
	pollInterval time.Duration
}

// NewService creates a new outcome evaluation service.
func NewService(storage Storage, marketStore *market.Store) *Service {
	return &Service{
		storage:      storage,
		marketStore:  marketStore,
		tracker:      NewPriceTracker(),
		pollInterval: 1 * time.Minute,
	}
}

// Run starts the background evaluation loop.
func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	// Initial load of candidates
	s.syncCandidates(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncCandidates(ctx)
			s.evaluateActive(ctx)
		}
	}
}

// syncCandidates ensures the PriceTracker is aware of all currently active signals.
func (s *Service) syncCandidates(ctx context.Context) {
	candidates, err := s.storage.ActiveCandidates(ctx)
	if err != nil {
		log.Printf("[outcome] failed to load active candidates: %v", err)
		return
	}

	// We simply re-add them; AddCandidate handles initialization safely.
	// For a production system, we'd want to remove expired candidates from the tracker to save memory.
	// We'll let EvaluateActive handle removal when a candidate passes its 24h horizon.
	for _, c := range candidates {
		s.tracker.AddCandidate(c)
	}
}

// evaluateActive evaluates all active signals and persists the results.
func (s *Service) evaluateActive(ctx context.Context) {
	candidates, err := s.storage.ActiveCandidates(ctx)
	if err != nil {
		return
	}
	
	now := time.Now()

	for _, c := range candidates {
		// Update price tracker from recent market data
		snapshot, ok := s.marketStore.Snapshot(c.Symbol, 0)
		if ok {
			s.tracker.UpdateFromTrades(c.SignalID, ExtractRelevantTrades(snapshot.Trades, c.CreatedAt))
			if candles, ok := snapshot.Candles["1m"]; ok {
				s.tracker.UpdateFromCandles(c.SignalID, ExtractRelevantCandles(candles, c.CreatedAt))
			}
		}

		obs, ok := s.tracker.GetObservation(c.SignalID)
		if !ok {
			continue // Should not happen if syncCandidates worked
		}

		currentPrice := snapshot.LastPrice
		if currentPrice <= 0 && ok {
			currentPrice = snapshot.Book.MidPrice
		}

		age := now.Sub(c.CreatedAt)
		returns := make(map[Horizon]HorizonReturn)
		
		// Evaluate for each horizon that has passed
		allHorizonsFinished := true
		for _, h := range AllHorizons() {
			if age >= h.Duration() {
				// The horizon has passed, calculate outcome
				// In a real system, we'd query historical data exactly at Horizon duration
				// For this paper system, since we evaluate continuously, we might just record
				// the latest price at the exact moment, or use the observation.
				returns[h] = EvaluateHorizon(c, h, currentPrice, obs, now)
			} else {
				allHorizonsFinished = false
			}
		}

		if len(returns) > 0 {
			result := EvaluateTotal(c, returns)
			if err := s.storage.SaveOutcome(ctx, result); err != nil {
				log.Printf("[outcome] failed to save result for %s: %v", c.SignalID, err)
			}
		}

		if allHorizonsFinished {
			// All 24h have passed, we don't need to track this signal's prices anymore
			s.tracker.RemoveCandidate(c.SignalID)
		}
	}
}
