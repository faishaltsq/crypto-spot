package domain

import (
	"testing"
	"time"
)

func TestSignalIsActive(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{"SETUP", true},
		{"CONFIRMED", true},
		{"ACTIVE", true},
		{"WATCH", true},
		{"CLOSED", false},
		{"INVALIDATED", false},
		{"EXPIRED", false},
		{"BLOCKED", false},
		{"SUPPRESSED", false},
		{"closed", false}, // case-insensitive
	}
	for _, c := range cases {
		if got := SignalIsActive(c.status); got != c.want {
			t.Errorf("SignalIsActive(%q) = %v, want %v", c.status, got, c.want)
		}
	}
}

func TestSignalIsActiveAt_ExpiryOverride(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	if !SignalIsActiveAt("CONFIRMED", future, now) {
		t.Error("non-terminal status with future expiry should be active")
	}
	if SignalIsActiveAt("CONFIRMED", past, now) {
		t.Error("non-terminal status past expiry should be inactive")
	}
	if SignalIsActiveAt("CLOSED", future, now) {
		t.Error("terminal status should be inactive even with future expiry")
	}
	if !SignalIsActiveAt("ACTIVE", time.Time{}, now) {
		t.Error("zero-value expiresAt should not force inactive")
	}
}

func TestSignalDirection(t *testing.T) {
	cases := map[string]string{
		"BUY_SETUP":     "BUY",
		"BUY_CONFIRMED": "BUY",
		SellSignalSetup:     "SELL",
		SellSignalConfirmed: "SELL",
		TakeProfitSuggested: "SELL",
		AvoidEntrySignal:    "SELL",
		ExitWarningSignal:   "SELL",
	}
	for signalType, want := range cases {
		if got := SignalDirection(signalType); got != want {
			t.Errorf("SignalDirection(%q) = %q, want %q", signalType, got, want)
		}
	}
}

func TestSignalStrategy(t *testing.T) {
	cases := map[string]string{
		"BUY_SETUP":         "ENTRY_BUY",
		"BUY_CONFIRMED":     "ENTRY_BUY",
		SellSignalSetup:     "PROTECTIVE_SELL",
		SellSignalConfirmed: "PROTECTIVE_SELL",
		TakeProfitSuggested: "TAKE_PROFIT",
		AvoidEntrySignal:    "AVOID_ENTRY",
		ExitWarningSignal:   "EXIT_WARNING",
	}
	for signalType, want := range cases {
		if got := SignalStrategy(signalType); got != want {
			t.Errorf("SignalStrategy(%q) = %q, want %q", signalType, got, want)
		}
	}
}

func TestSignal_Enrich(t *testing.T) {
	s := Signal{
		Type:      "BUY_CONFIRMED",
		Status:    "CONFIRMED",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	s.Enrich()
	if !s.IsActive {
		t.Error("expected IsActive true for CONFIRMED BUY signal with future expiry")
	}
	if s.Direction != "BUY" {
		t.Errorf("expected Direction BUY, got %q", s.Direction)
	}
	if s.Strategy != "ENTRY_BUY" {
		t.Errorf("expected Strategy ENTRY_BUY, got %q", s.Strategy)
	}
	if s.LifecycleGroup != "CONFIRMED" {
		t.Errorf("expected LifecycleGroup CONFIRMED, got %q", s.LifecycleGroup)
	}

	closed := Signal{Type: "SELL_CONFIRMED", Status: "CLOSED", ExpiresAt: time.Now().Add(time.Hour)}
	closed.Enrich()
	if closed.IsActive {
		t.Error("closed signal must never be active")
	}

	expired := Signal{Type: AvoidEntrySignal, Status: "ACTIVE", ExpiresAt: time.Now().Add(-time.Minute)}
	expired.Enrich()
	if expired.IsActive {
		t.Error("expired signal must not be active even with non-terminal status")
	}
}
