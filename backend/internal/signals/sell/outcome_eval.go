package sell

import (
	"time"

	"github.com/example/crypto-spot-signal/internal/outcome"
	"github.com/example/crypto-spot-signal/internal/storage"
)

// EvaluateDirectional measures directional accuracy for a SELL-family
// candidate: did price actually decline (a downside move), not whether a
// hypothetical short position would have been profitable after fees. This
// is intentionally distinct from the BUY engine's execution-simulation-based
// net-return metric (execution_simulation package) — SELL signals in this
// SPOT-only system are informational/protective, never executed as trades.
func EvaluateDirectional(c storage.SellCandidate, obs outcome.PriceObservation, currentPrice float64, now time.Time) storage.SellSignalOutcome {
	directionalReturn := outcome.CalculateExcursionPct(c.EntryPrice, currentPrice)
	maxDownside := outcome.CalculateExcursionPct(c.EntryPrice, obs.Low)
	maxAdverseUpside := outcome.CalculateExcursionPct(c.EntryPrice, obs.High)

	// Directionally accurate = price actually declined from entry.
	directionalAccuracy := directionalReturn < 0

	// Support reclaim: price recovered back above entry after having
	// declined below it (obs.Low < entry but current price >= entry).
	supportReclaim := obs.Low < c.EntryPrice && currentPrice >= c.EntryPrice

	// Breakdown follow-through: price continued below the projected
	// downside target (Target1, which priceLevels.go sets below entry).
	breakdownFollowThrough := c.Target1 > 0 && obs.Low <= c.Target1

	// Invalidated: price reclaimed the invalidation level set at signal
	// creation (Invalidation is the nearest bearish structure level or
	// entry+volatility buffer — see signals/sell/protective_sell.go).
	invalidated := c.Invalidation > 0 && currentPrice >= c.Invalidation

	return storage.SellSignalOutcome{
		SignalID:               c.SignalID,
		Symbol:                 c.Symbol,
		EvaluatedAt:            now,
		DirectionalReturn:      directionalReturn,
		DirectionalAccuracy:    directionalAccuracy,
		MaxDownsideMove:        maxDownside,
		MaxAdverseUpsideMove:   maxAdverseUpside,
		SupportReclaim:         supportReclaim,
		BreakdownFollowThrough: breakdownFollowThrough,
		Invalidated:            invalidated,
	}
}
