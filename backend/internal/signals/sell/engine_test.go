package sell

import (
	"testing"
	"time"

	"github.com/example/crypto-spot-signal/internal/config"
	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/features/spoofing"
	"github.com/example/crypto-spot-signal/internal/features/structure"
	"github.com/example/crypto-spot-signal/internal/features/tradeflow"
)

func bearishSnapshot(symbol string) FeatureSnapshot {
	return FeatureSnapshot{
		Symbol:            symbol,
		Tier:              1,
		Price:             100,
		TrendByTimeframe:  map[string]string{"15m": "bearish", "5m": "bearish"},
		TrendAlignment:    -0.6,
		LiquidityScore:    80,
		DataQualityScore:  95,
		DataQualityStatus: domain.DataQualityValid,
		SpoofScoreRaw:     10,
		SpoofStatus:       domain.SpoofStatusLow,
		MarketRegime:      "RANGING",
		OrderbookSynced:   true,
		TradeFlow: tradeflow.SellFlowSnapshot{
			AggressiveSellRatio:  0.75,
			AggressiveBuyRatio:   0.25,
			NegativeCVDSlope:     -5,
			LargeSellTradeCount:  4,
			SampleStatus:         tradeflow.SampleValid,
		},
		Structure: structure.BearishStructure{
			SupportBroken:          true,
			ClosedCandleConfirmed:  true,
			LowerHighDetected:      true,
			LowerLowDetected:       true,
			ReclaimFailed:          true,
			BreakdownFollowThrough: true,
			StructureScore:         90,
		},
		Walls: spoofing.WallAnalysis{
			BidWallFailed:            true,
			BidWallFailureConfidence: 0.8,
		},
		CalculatedAt: time.Now(),
	}
}

func testConfig() EngineConfig {
	return FromAppConfig(config.SellConfig{
		Enabled:                     true,
		RequireClosedCandle:         true,
		RequireExecutedTradeConfirm: true,
		RequireOrderbookSync:        true,
		RequireFailedReclaim:        true,
		SetupScore:                  60,
		ConfirmScore:                75,
		MinRuleScore:                50,
		MinModelProbability:         0.5,
		MinDataQuality:              70,
		MaxSpoofScore:               60,
		MinTradeflowScore:           50,
		MinTimeframeAlignment:       30,
		PairCooldown:                30 * time.Minute,
		MaxActiveGlobal:             15,
		MaxActivePerPair:            1,
		MaxActivePerCluster:         3,
		MaxNewPerMinute:             5,
		TakeProfitEnabled:           true,
		TakeProfitSetupScore:        60,
		TakeProfitConfirmScore:      75,
		TakeProfitMinOverextension:  50,
		TakeProfitMinExhaustion:     50,
	})
}

func TestEngineEvaluateProtectiveSellFiresOnStrongBearishEvidence(t *testing.T) {
	engine := New(testConfig())
	f := bearishSnapshot("BTC_USDT")
	f.SellRuleScore = RuleScore(f)

	sig, created := engine.Evaluate(f, ActiveBuyContext{})
	if !created {
		t.Fatalf("expected a SELL signal to be created, ruleScore=%.1f", f.SellRuleScore)
	}
	if sig.Type != domain.SellSignalSetup && sig.Type != domain.SellSignalConfirmed {
		t.Fatalf("unexpected signal type: %s", sig.Type)
	}
}

func TestEngineEvaluateBlocksOnUnsyncedOrderbook(t *testing.T) {
	engine := New(testConfig())
	f := bearishSnapshot("BTC_USDT")
	f.OrderbookSynced = false

	_, created := engine.Evaluate(f, ActiveBuyContext{})
	if created {
		t.Fatal("expected no signal when orderbook is unsynced")
	}
}

func TestEngineEvaluateBlocksOnInsufficientTradeSample(t *testing.T) {
	engine := New(testConfig())
	f := bearishSnapshot("BTC_USDT")
	f.TradeFlow.SampleStatus = tradeflow.SampleInsufficient

	_, created := engine.Evaluate(f, ActiveBuyContext{})
	if created {
		t.Fatal("expected no signal when trade sample is insufficient")
	}
}

func TestEngineEvaluateBlocksOnLowDataQuality(t *testing.T) {
	engine := New(testConfig())
	f := bearishSnapshot("BTC_USDT")
	f.DataQualityScore = 10

	_, created := engine.Evaluate(f, ActiveBuyContext{})
	if created {
		t.Fatal("expected no signal when data quality is below minimum")
	}
}

func TestEngineEvaluateAvoidEntryWhenBuyCandidateExists(t *testing.T) {
	engine := New(testConfig())
	f := bearishSnapshot("ETH_USDT")

	sig, created := engine.Evaluate(f, ActiveBuyContext{HasCandidateSignal: true})
	if !created {
		t.Fatal("expected AVOID_ENTRY signal when a BUY candidate exists alongside bearish evidence")
	}
	if sig.Type != domain.AvoidEntrySignal {
		t.Fatalf("expected AVOID_ENTRY, got %s", sig.Type)
	}
}

func TestEngineEvaluateExitWarningWhenPositionActive(t *testing.T) {
	engine := New(testConfig())
	f := bearishSnapshot("ETH_USDT")

	sig, created := engine.Evaluate(f, ActiveBuyContext{HasActivePosition: true, ActiveSignalID: "abc-123"})
	if !created {
		t.Fatal("expected EXIT_WARNING signal when an active position faces bearish evidence")
	}
	if sig.Type != domain.ExitWarningSignal {
		t.Fatalf("expected EXIT_WARNING, got %s", sig.Type)
	}
}

func TestEngineDisabledNeverFires(t *testing.T) {
	cfg := testConfig()
	cfg.Enabled = false
	engine := New(cfg)
	f := bearishSnapshot("BTC_USDT")

	_, created := engine.Evaluate(f, ActiveBuyContext{})
	if created {
		t.Fatal("disabled engine must never emit a signal")
	}
}

func TestCooldownBlocksRepeatWithinWindow(t *testing.T) {
	cd := NewCooldown()
	if !cd.Allow("BTC_USDT", domain.SellSignalConfirmed, time.Hour) {
		t.Fatal("first call should be allowed")
	}
	if cd.Allow("BTC_USDT", domain.SellSignalConfirmed, time.Hour) {
		t.Fatal("second call within cooldown window should be blocked")
	}
	// A different signal type for the same pair has its own independent
	// cooldown bucket (spec: EXIT_WARNING must not be blocked by a prior
	// TAKE_PROFIT_SUGGESTED cooldown for the same pair).
	if !cd.Allow("BTC_USDT", domain.ExitWarningSignal, time.Hour) {
		t.Fatal("different signal type for same pair must have independent cooldown")
	}
}

func TestBurstGuardBlocksWithinWindow(t *testing.T) {
	var guard BurstGuard
	if !guard.Allow(5) {
		t.Fatal("first call should be allowed")
	}
	if guard.Allow(5) {
		t.Fatal("immediate second call within BurstWindow should be blocked")
	}
}

func TestCheckInvalidationSupportReclaimed(t *testing.T) {
	f := bearishSnapshot("BTC_USDT")
	f.Structure.ReclaimAttempted = true
	f.Structure.ReclaimFailed = false

	result := CheckInvalidation(f)
	if !result.Invalidated || result.Reason != domain.InvalidationSupportReclaimed {
		t.Fatalf("expected SUPPORT_RECLAIMED invalidation, got %+v", result)
	}
}

func TestCheckInvalidationNoneWhenThesisHolds(t *testing.T) {
	f := bearishSnapshot("BTC_USDT")
	f.TradeFlow.BuyRecovery = 0
	f.TradeFlow.SellExhaustion = 0

	result := CheckInvalidation(f)
	if result.Invalidated {
		t.Fatalf("expected no invalidation while bearish thesis holds, got %+v", result)
	}
}
