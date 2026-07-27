package ai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
)

func candidate() domain.FeatureSnapshot {
	return domain.FeatureSnapshot{
		Symbol: "BTC_USDT", Status: "BUY_CONFIRMED_CANDIDATE", RuleScore: 84,
		DataQualityScore: 90, SpoofScore: 20, TrendAlignment: 0.4,
		TrendByTimeframe: map[string]string{"15m": "bullish"},
		Reasons:          []string{"VOLUME_EXPANSION"},
	}
}

func TestReviewSendsOnlyFeatureSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		if string(body) == "" || containsForbiddenField(string(body)) {
			t.Fatal("raw feature fields crossed AI boundary")
		}
		_, _ = w.Write([]byte(`{"decision":"WAIT","confidence":0.5,"summary":"review","supporting_reason_codes":[],"contradicting_reason_codes":[],"risk_flags":[],"provider":"deepseek","model":"test","latency_ms":1,"fallback":false,"prompt_version":"ai-review-v1","schema_version":"ai-review-schema-v1"}`))
	}))
	defer server.Close()

	review := New(true, server.URL, time.Second, 70).Review(context.Background(), candidate())
	if review.Decision != "WAIT" || review.Fallback {
		t.Fatalf("unexpected review: %#v", review)
	}
}

func TestReviewFallsBackForUnknownDecision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"decision":"PROMOTE","confidence":1,"summary":"bad"}`))
	}))
	defer server.Close()

	review := New(true, server.URL, time.Second, 70).Review(context.Background(), candidate())
	if !review.Fallback || review.ProviderErrorCode != "AI_RESPONSE_INVALID" {
		t.Fatalf("expected validated fallback: %#v", review)
	}
}

func TestBlockedFeatureNeverCallsProvider(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	feature := candidate()
	feature.Status = "BLOCKED"
	feature.BlockedReasons = []string{"HIGH_SPOOF_RISK"}
	review := New(true, server.URL, time.Second, 70).Review(context.Background(), feature)
	if called || review.Decision != "UNAVAILABLE" {
		t.Fatalf("blocked provider safety failed: called=%t review=%#v", called, review)
	}
}

func containsForbiddenField(body string) bool {
	return contains(body, "entryPrice") || contains(body, "blockedReasons") || contains(body, "api_key")
}

func contains(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
