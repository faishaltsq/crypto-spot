package domain

import (
	"strings"
	"time"
)

// RecordKind definitions to separate actionable signals from audit/candidates
const (
	RecordKindCandidate           = "CANDIDATE"
	RecordKindWatch               = "WATCH"
	RecordKindActionableSetup     = "ACTIONABLE_SETUP"
	RecordKindActionableConfirmed = "ACTIONABLE_CONFIRMED"
	RecordKindBlockedAudit        = "BLOCKED_AUDIT"
	RecordKindSuppressedAudit     = "SUPPRESSED_AUDIT"
	RecordKindTerminal            = "TERMINAL"
)

// Terminal statuses are lifecycle end-states: once a signal reaches one of
// these, it can never be considered active again, regardless of signal
// type. This is backend-owned business logic — the frontend must never
// derive "is active" from raw status strings on its own.
var terminalStatuses = map[string]bool{
	"CLOSED":      true,
	"INVALIDATED": true,
	"EXPIRED":     true,
	"BLOCKED":     true,
	"SUPPRESSED":  true,
}

// SignalIsActive is the single backend-owned rule for whether a signal
// (BUY or SELL family) should be surfaced as "active" to any client. A
// signal is active if its status has not reached a terminal state. This
// intentionally does NOT require status == "ACTIVE" — BUY signals live in
// SETUP/CONFIRMED while active, and SELL-family signals live in
// SETUP/CONFIRMED/ACTIVE while active. Only terminal statuses make a
// signal inactive.
func SignalIsActive(status string) bool {
	return !terminalStatuses[strings.ToUpper(strings.TrimSpace(status))]
}

// SignalIsActiveAt is SignalIsActive plus an expiry check. Some SELL-family
// signal types (AVOID_ENTRY, EXIT_WARNING) are advisory-only and are never
// enqueued into an outcome evaluator loop, so their DB status can remain
// non-terminal indefinitely. Treating an expired signal as active would
// make the Terminal show stale advisories forever, so any signal past its
// ExpiresAt is always considered inactive regardless of stored status.
func SignalIsActiveAt(status string, expiresAt, now time.Time) bool {
	if !SignalIsActive(status) {
		return false
	}
	if !expiresAt.IsZero() && now.After(expiresAt) {
		return false
	}
	return true
}

// SignalDirection derives BUY/SELL from a signal's type string. Every
// signal type in this system unambiguously belongs to one direction.
func SignalDirection(signalType string) string {
	t := strings.ToUpper(signalType)
	switch {
	case strings.HasPrefix(t, "BUY"):
		return "BUY"
	case strings.HasPrefix(t, "SELL"),
		t == TakeProfitSuggested,
		t == AvoidEntrySignal,
		t == ExitWarningSignal:
		return "SELL"
	default:
		return "BUY"
	}
}

// SignalStrategy maps a raw signal type to the coarse strategy family used
// by the frontend's unified signal model.
func SignalStrategy(signalType string) string {
	switch strings.ToUpper(signalType) {
	case "BUY_SETUP", "BUY_CONFIRMED":
		return "ENTRY_BUY"
	case SellSignalSetup, SellSignalConfirmed:
		return "PROTECTIVE_SELL"
	case TakeProfitSuggested:
		return "TAKE_PROFIT"
	case ExitWarningSignal:
		return "EXIT_WARNING"
	case AvoidEntrySignal:
		return "AVOID_ENTRY"
	default:
		return "PROTECTIVE_SELL"
	}
}

// SignalLifecycleGroup buckets a raw status into the coarse lifecycle
// groups the frontend renders (tabs, sort priority, visual hierarchy).
func SignalLifecycleGroup(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "SETUP":
		return "SETUP"
	case "CONFIRMED":
		return "CONFIRMED"
	case "ACTIVE":
		return "ACTIVE"
	case "BLOCKED":
		return "BLOCKED"
	case "INVALIDATED":
		return "INVALIDATED"
	case "EXPIRED":
		return "EXPIRED"
	case "CLOSED":
		return "CLOSED"
	case "SUPPRESSED":
		return "BLOCKED"
	case "WATCH":
		return "WATCH"
	default:
		return "WATCH"
	}
}
