package quality

import (
	"testing"
	"time"

	"github.com/example/crypto-spot-signal/internal/market"
)

func TestQualityRules(t *testing.T) {
	cfg := QualityConfig{
		MinSignalScore:         75,
		BlockSignalScore:       50,
		StaleTradeSec:          10,
		StaleTickerSec:         10,
		StaleOrderbookSec:      5,
		StaleCandleSec:         90,
		ReconnectCooldownSec:   30,
		MaxPriceDeviationBPS:   100,
		MaxQueueUtilizationPct: 85,
		MaxSpreadBPS:           35,
	}

	evaluator := NewEvaluator(cfg)
	now := time.Now()

	t.Run("Valid Data", func(t *testing.T) {
		snapshot := market.PairSnapshot{
			Symbol:           "BTC_USDT",
			LastPrice:        50000,
			LastMarketUpdate: now.Add(-1 * time.Second),
			Book: market.BookMetrics{
				Synced:        true,
				BestBid:       49990,
				BestAsk:       50010,
				MidPrice:      50000,
				SpreadBPS:     4,
				BidDepthQuote: 100000,
				AskDepthQuote: 100000,
				UpdatedAt:     now.Add(-1 * time.Second),
			},
			Trades: []market.Trade{
				{Timestamp: now.Add(-2 * time.Second)},
				{Timestamp: now.Add(-1 * time.Second)},
				{Timestamp: now},
			},
			Candles: map[string][]market.Candle{
				"1m": make([]market.Candle, 20),
				"5m": make([]market.Candle, 20),
				"15m": make([]market.Candle, 20),
			},
		}
		// Set timestamps manually to avoid zero-time checks failing
		for i := range snapshot.Candles["1m"] {
			snapshot.Candles["1m"][i].OpenTime = now.Add(-time.Duration(20-i) * time.Minute)
		}

		input := BuildHealthInput(snapshot)
		report := evaluator.Evaluate(snapshot, input)

		if report.Status != StatusValid {
			t.Errorf("expected VALID, got %s. Score: %f. Reasons: %v", report.Status, report.Score, report.Reasons)
		}
		if !report.SignalAllowed {
			t.Error("expected signal to be allowed")
		}
	})

	t.Run("Unsynced Orderbook", func(t *testing.T) {
		snapshot := market.PairSnapshot{
			Symbol: "BTC_USDT",
			Book: market.BookMetrics{
				Synced: false,
			},
		}
		input := BuildHealthInput(snapshot)
		report := evaluator.Evaluate(snapshot, input)

		if report.Status != StatusBlocked {
			t.Errorf("expected BLOCKED, got %s", report.Status)
		}
		if report.SignalAllowed {
			t.Error("expected signal to not be allowed")
		}
		
		found := false
		for _, r := range report.Reasons {
			if r == ReasonOrderbookUnsynced {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected ReasonOrderbookUnsynced to be in reasons")
		}
	})
}
