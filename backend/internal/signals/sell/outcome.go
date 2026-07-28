package sell

import (
	"context"
	"log"
	"time"

	"github.com/example/crypto-spot-signal/internal/market"
	"github.com/example/crypto-spot-signal/internal/outcome"
	"github.com/example/crypto-spot-signal/internal/storage"
)

// OutcomeStorage is the narrow persistence dependency the SELL outcome
// evaluator needs, mirroring outcome.Storage's shape so both evaluators
// follow the same interface-segregation pattern.
type OutcomeStorage interface {
	ActiveSellCandidates(ctx context.Context) ([]storage.SellCandidate, error)
	SaveSellSignalOutcome(ctx context.Context, o storage.SellSignalOutcome) error
	CloseSellSignal(ctx context.Context, signalID, status, invalidationReason string, closedAt time.Time) error
}

// OutcomeEvaluator measures whether SELL-family signals were directionally
// correct: did price actually decline (protective sell) after the signal,
// rather than measuring short-position profit. It reuses outcome.PriceTracker
// for High/Low excursion tracking since that logic is direction-agnostic.
type OutcomeEvaluator struct {
	storage      OutcomeStorage
	marketStore  *market.Store
	tracker      *outcome.PriceTracker
	pollInterval time.Duration
}

func NewOutcomeEvaluator(storage OutcomeStorage, marketStore *market.Store) *OutcomeEvaluator {
	return &OutcomeEvaluator{
		storage:      storage,
		marketStore:  marketStore,
		tracker:      outcome.NewPriceTracker(),
		pollInterval: time.Minute,
	}
}

// Run starts the background directional-accuracy evaluation loop. It is a
// completely separate long-running goroutine from outcome.Service.Run (BUY)
// — the two never share tracker state, even though both watch the same
// live market.Store.
func (e *OutcomeEvaluator) Run(ctx context.Context) {
	ticker := time.NewTicker(e.pollInterval)
	defer ticker.Stop()

	e.sync(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.sync(ctx)
			e.evaluate(ctx)
		}
	}
}

func (e *OutcomeEvaluator) sync(ctx context.Context) {
	candidates, err := e.storage.ActiveSellCandidates(ctx)
	if err != nil {
		log.Printf("[sell-outcome] failed to load active candidates: %v", err)
		return
	}
	for _, c := range candidates {
		e.tracker.AddCandidate(outcome.Candidate{
			SignalID: c.SignalID, Symbol: c.Symbol, EntryPrice: c.EntryPrice,
			Target1: c.Target1, Invalidation: c.Invalidation, CreatedAt: c.CreatedAt,
		})
	}
}

func (e *OutcomeEvaluator) evaluate(ctx context.Context) {
	candidates, err := e.storage.ActiveSellCandidates(ctx)
	if err != nil {
		return
	}
	now := time.Now()

	for _, c := range candidates {
		snapshot, ok := e.marketStore.Snapshot(c.Symbol, 0)
		if !ok {
			continue
		}
		e.tracker.UpdateFromTrades(c.SignalID, outcome.ExtractRelevantTrades(snapshot.Trades, c.CreatedAt))
		if candles, ok := snapshot.Candles["1m"]; ok {
			e.tracker.UpdateFromCandles(c.SignalID, outcome.ExtractRelevantCandles(candles, c.CreatedAt))
		}

		obs, ok := e.tracker.GetObservation(c.SignalID)
		if !ok {
			continue
		}
		currentPrice := snapshot.LastPrice
		if currentPrice <= 0 {
			currentPrice = snapshot.Book.MidPrice
		}
		if currentPrice <= 0 {
			continue
		}

		result := EvaluateDirectional(c, obs, currentPrice, now)
		if err := e.storage.SaveSellSignalOutcome(ctx, result); err != nil {
			log.Printf("[sell-outcome] failed to save outcome for %s: %v", c.SignalID, err)
		}

		if result.Invalidated {
			status := "INVALIDATED"
			if err := e.storage.CloseSellSignal(ctx, c.SignalID, status, "SUPPORT_RECLAIMED", now); err != nil {
				log.Printf("[sell-outcome] failed to close signal %s: %v", c.SignalID, err)
			}
			e.tracker.RemoveCandidate(c.SignalID)
			continue
		}

		age := now.Sub(c.CreatedAt)
		if age >= 24*time.Hour {
			if err := e.storage.CloseSellSignal(ctx, c.SignalID, "CLOSED", "EXPIRED", now); err != nil {
				log.Printf("[sell-outcome] failed to close expired signal %s: %v", c.SignalID, err)
			}
			e.tracker.RemoveCandidate(c.SignalID)
		}
	}
}
