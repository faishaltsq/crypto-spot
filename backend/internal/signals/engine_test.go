package signals

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/example/crypto-spot-signal/internal/ai"
	"github.com/example/crypto-spot-signal/internal/domain"
)

// testConfig returns an EngineConfig matching the engine's historical
// hardcoded gates (ConfirmScore 80, MaxSpoofScore 60, MinTrendAlignment
// 0.20) with rate limits disabled so evaluation logic can be tested in
// isolation from throttling.
func testConfig() EngineConfig {
	return EngineConfig{
		ConfirmScore:      80,
		MaxSpoofScore:     60,
		MinTrendAlignment: 0.20,
		MaxNewPerMinute:   1000,
		PairCooldown:      0,
	}
}

func TestEvaluateUsesFinalThreshold(t *testing.T) {
	engine := New(testConfig(), ai.New(false, "", time.Second, 80), nil, nil)
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
	engine := New(testConfig(), ai.New(false, "", time.Second, 80), nil, nil)
	signal, created := engine.Evaluate(context.Background(), thresholdFeature(83))
	if !created || signal.Type != "BUY_CONFIRMED" {
		t.Fatalf("created=%v signal=%+v", created, signal)
	}
	if signal.Threshold.ThresholdVersion != "threshold-v1" || signal.Threshold.TierAdjustment != 3 || !signal.Threshold.Passed {
		t.Fatalf("threshold=%+v", signal.Threshold)
	}
}

func TestThresholdBreakdownSerializesForPersistence(t *testing.T) {
	engine := New(testConfig(), ai.New(false, "", time.Second, 80), nil, nil)
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

// TestConfigDrivesSpoofGate proves the MaxSpoofScore gate reads live config
// rather than a hardcoded const: an identical feature is CONFIRMED under a
// permissive spoof ceiling and demoted to SETUP under a strict one.
func TestConfigDrivesSpoofGate(t *testing.T) {
	feature := thresholdFeature(83)
	feature.SpoofScore = 55 // between the two ceilings below

	permissive := New(testConfig(), ai.New(false, "", time.Second, 80), nil, nil)
	sig, ok := permissive.Evaluate(context.Background(), feature)
	if !ok || sig.Type != "BUY_CONFIRMED" {
		t.Fatalf("permissive spoof: expected BUY_CONFIRMED, got ok=%v type=%s", ok, sig.Type)
	}

	strictCfg := testConfig()
	strictCfg.MaxSpoofScore = 50 // 55 now exceeds ceiling -> blocks CONFIRMED
	strict := New(strictCfg, ai.New(false, "", time.Second, 80), nil, nil)
	sig, ok = strict.Evaluate(context.Background(), feature)
	if !ok || sig.Type == "BUY_CONFIRMED" {
		t.Fatalf("strict spoof: expected demotion from CONFIRMED, got ok=%v type=%s", ok, sig.Type)
	}
}

// TestConfigDrivesTrendGate proves the MinTrendAlignment gate is config-driven.
func TestConfigDrivesTrendGate(t *testing.T) {
	feature := thresholdFeature(83)
	feature.TrendAlignment = 0.25

	permissive := New(testConfig(), ai.New(false, "", time.Second, 80), nil, nil)
	sig, ok := permissive.Evaluate(context.Background(), feature)
	if !ok || sig.Type != "BUY_CONFIRMED" {
		t.Fatalf("permissive trend: expected BUY_CONFIRMED, got ok=%v type=%s", ok, sig.Type)
	}

	strictCfg := testConfig()
	strictCfg.MinTrendAlignment = 0.40 // 0.25 now below floor -> blocks CONFIRMED
	strict := New(strictCfg, ai.New(false, "", time.Second, 80), nil, nil)
	sig, ok = strict.Evaluate(context.Background(), feature)
	if !ok || sig.Type == "BUY_CONFIRMED" {
		t.Fatalf("strict trend: expected demotion from CONFIRMED, got ok=%v type=%s", ok, sig.Type)
	}
}

// TestSetConfigLiveReload proves SetConfig swaps the live gate at runtime,
// which is the core fase-2 bug fix: Admin Settings changes must alter engine
// behavior without a restart.
func TestSetConfigLiveReload(t *testing.T) {
	feature := thresholdFeature(83)
	feature.SpoofScore = 55

	engine := New(testConfig(), ai.New(false, "", time.Second, 80), nil, nil)
	if sig, ok := engine.Evaluate(context.Background(), feature); !ok || sig.Type != "BUY_CONFIRMED" {
		t.Fatalf("before reload: expected BUY_CONFIRMED, got ok=%v type=%s", ok, sig.Type)
	}

	tighter := testConfig()
	tighter.MaxSpoofScore = 50
	if err := engine.SetConfig(tighter); err != nil {
		t.Fatalf("SetConfig: %v", err)
	}
	// Reset the global anti-burst guard: the hardcoded BurstWindow would
	// otherwise reject the second evaluation purely on timing, which is
	// unrelated to the config-reload behavior under test.
	engine.burst = burstGuard{}
	if sig, ok := engine.Evaluate(context.Background(), feature); !ok || sig.Type == "BUY_CONFIRMED" {
		t.Fatalf("after reload: expected demotion, got ok=%v type=%s", ok, sig.Type)
	}
}

// TestSetConfigRejectsInvalid proves SetConfig validates before swapping so a
// bad Admin Settings payload cannot corrupt the live gate.
func TestSetConfigRejectsInvalid(t *testing.T) {
	engine := New(testConfig(), ai.New(false, "", time.Second, 80), nil, nil)
	bad := testConfig()
	bad.MaxSpoofScore = 150 // out of 0..100 range
	if err := engine.SetConfig(bad); err == nil {
		t.Fatal("expected SetConfig to reject out-of-range MaxSpoofScore")
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
