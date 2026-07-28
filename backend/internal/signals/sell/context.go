package sell

import "time"

// ActiveBuyContext describes whatever BUY-side signal state is currently
// known for a symbol, supplied by the caller (main.go's scanner loop) so the
// SELL engine can decide between AVOID_ENTRY (no position, BUY candidate
// forming) and EXIT_WARNING (already holding, from a prior BUY_CONFIRMED).
// The SELL engine never queries storage directly — it only reacts to what
// the orchestrator tells it, keeping this package storage-agnostic and
// testable without a database.
type ActiveBuyContext struct {
	HasCandidateSignal bool      // a BUY_SETUP/BUY_CONFIRMED_CANDIDATE exists right now
	HasActivePosition  bool      // a BUY_CONFIRMED signal is still open (not CLOSED/EXPIRED/INVALIDATED)
	ActiveSignalID     string
	ActiveEntryPrice   float64
	ActiveSince        time.Time
}
