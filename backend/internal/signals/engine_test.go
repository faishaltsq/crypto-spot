package signals

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/example/crypto-spot-signal/internal/ai"
	"github.com/example/crypto-spot-signal/internal/domain"
)

func TestEvaluateUsesFinalThreshold(t *testing.T) {
	engine := New(80, 0, ai.New(false, "", time.Second, 80), nil, nil)
	feature := thresholdFeature(82)
	signal, created := engine.Evaluate(context.Background(), feature)
	if !created {
		t.Fatal("expected auditable blocked signal")
	}
	if signal.Type == "BUY_CONFIRMED" || signal.Status != "BLOCKED" {
		t.Fatalf("type=%s status=%s", signal.Type, signal.Status)
	}
	if signal.Threshold.FinalThreshold != 83 || signal.Threshold.Passed {
		t.Fatalf("threshold=%+v", signal.Threshold)
	}
}

func TestEvaluatePersistsThresholdBreakdown(t *testing.T) {
	engine := New(80, 0, ai.New(false, "", time.Second, 80), nil, nil)
	signal, created := engine.Evaluate(context.Background(), thresholdFeature(83))
	if !created || signal.Type != "BUY_CONFIRMED" {
		t.Fatalf("created=%v signal=%+v", created, signal)
	}
	if signal.Threshold.ThresholdVersion != "threshold-v1" || signal.Threshold.TierAdjustment != 3 || !signal.Threshold.Passed {
		t.Fatalf("threshold=%+v", signal.Threshold)
	}
}

func TestThresholdBreakdownSerializesForPersistence(t *testing.T) {
	engine := New(80, 0, ai.New(false, "", time.Second, 80), nil, nil)
	signal, created := engine.Evaluate(context.Background(), thresholdFeature(83))
	if !created {
		t.Fatal("expected signal")
	}
	payload, err := json.Marshal(signal.Threshold)
	if err != nil {
		t.Fatal(err)
	}
	var stored struct {
		Version string  `json:"thresholdVersion"`
		Final   float64 `json:"finalThreshold"`
		Passed  bool    `json:"passed"`
	}
	if err := json.Unmarshal(payload, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Version != "threshold-v1" || stored.Final != 83 || !stored.Passed {
		t.Fatalf("stored threshold=%+v", stored)
	}
}

func thresholdFeature(score float64) domain.FeatureSnapshot {
	return domain.FeatureSnapshot{
		Symbol: "BTC_USDT", Tier: 2, Price: 100, RuleScore: score,
		Status: "BUY_CONFIRMED_CANDIDATE", TrendAlignment: 0.5,
		SpoofStatus: domain.SpoofStatusLow, LiquidityScore: 80,
		DataQualityStatus: domain.DataQualityValid, DataQualityScore: 100,
		MarketRegime: "STRONG_UPTREND", VolatilityPercentile: 10,
		CorrelationState: "INDEPENDENT", TrendByTimeframe: map[string]string{},
	}
}
