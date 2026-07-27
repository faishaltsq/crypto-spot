package features

import (
	"testing"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/market"
)

func TestComputeProducesBoundedScore(t *testing.T) {
	now := time.Now()
	candles := make([]domain.Candle, 0, 30)
	for index := 0; index < 30; index++ {
		price := 100.0 + float64(index)*0.2
		candles = append(candles, domain.Candle{
			Symbol:      "BTC_USDT",
			Timeframe:   "1m",
			OpenTime:    now.Add(time.Duration(index-30) * time.Minute),
			Open:        price - 0.1,
			High:        price + 0.2,
			Low:         price - 0.2,
			Close:       price,
			QuoteVolume: 1000,
			Closed:      true,
		})
	}
	trades := []domain.Trade{
		{Symbol: "BTC_USDT", Side: "buy", Price: 106, Quote: 1200, Timestamp: now.Add(-20 * time.Second)},
		{Symbol: "BTC_USDT", Side: "buy", Price: 106, Quote: 800, Timestamp: now.Add(-10 * time.Second)},
		{Symbol: "BTC_USDT", Side: "sell", Price: 105.9, Quote: 300, Timestamp: now.Add(-5 * time.Second)},
	}
	snapshot := market.PairSnapshot{
		Symbol:           "BTC_USDT",
		LastPrice:        106,
		LastMarketUpdate: now,
		Trades:           trades,
		Candles:          map[string][]domain.Candle{"1m": candles, "15m": candles, "1h": candles},
		Book: domain.BookMetrics{
			Synced:        true,
			MidPrice:      106,
			SpreadBPS:     2,
			BidDepthQuote: 80000,
			AskDepthQuote: 50000,
			Imbalance:     0.23,
			SpoofScore:    10,
		},
	}

	engine := New(Config{MaxSpreadBPS: 35, MinDepthQuote: 25000})
	feature := engine.Compute(snapshot)
	if feature.RuleScore < 0 || feature.RuleScore > 100 {
		t.Fatalf("score outside expected range: %f", feature.RuleScore)
	}
	if feature.DataQualityScore < 60 {
		t.Fatalf("expected usable data quality, got %f", feature.DataQualityScore)
	}
	if feature.TrendByTimeframe["1h"] != "bullish" {
		t.Fatalf("expected bullish trend, got %s", feature.TrendByTimeframe["1h"])
	}
}
