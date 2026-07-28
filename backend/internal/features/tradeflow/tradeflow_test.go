package tradeflow

import (
	"testing"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
)

func trade(side string, quote float64, secondsAgo int) domain.Trade {
	return domain.Trade{
		Side:      side,
		Quote:     quote,
		Timestamp: time.Now().Add(-time.Duration(secondsAgo) * time.Second),
	}
}

func TestComputeDelta(t *testing.T) {
	trades := []domain.Trade{
		trade("buy", 100, 1),
		trade("sell", 300, 2),
		trade("sell", 200, 3),
	}
	buyVol, sellVol, delta := ComputeDelta(trades)
	if buyVol != 100 {
		t.Fatalf("buyVol = %v, want 100", buyVol)
	}
	if sellVol != 500 {
		t.Fatalf("sellVol = %v, want 500", sellVol)
	}
	if delta != -400 {
		t.Fatalf("delta = %v, want -400", delta)
	}
}

func TestComputeDeltaIgnoresUnknownSide(t *testing.T) {
	trades := []domain.Trade{
		trade("buy", 100, 1),
		{Side: "", Quote: 999, Timestamp: time.Now()},
	}
	buyVol, sellVol, _ := ComputeDelta(trades)
	if buyVol != 100 || sellVol != 0 {
		t.Fatalf("unknown side leaked into totals: buy=%v sell=%v", buyVol, sellVol)
	}
}

func TestAggressiveRatios(t *testing.T) {
	buyRatio, sellRatio := AggressiveRatios(300, 700)
	if buyRatio != 0.3 {
		t.Fatalf("buyRatio = %v, want 0.3", buyRatio)
	}
	if sellRatio != 0.7 {
		t.Fatalf("sellRatio = %v, want 0.7", sellRatio)
	}
}

func TestAggressiveRatiosZeroVolume(t *testing.T) {
	buyRatio, sellRatio := AggressiveRatios(0, 0)
	if buyRatio != 0 || sellRatio != 0 {
		t.Fatalf("expected zero ratios for zero volume, got buy=%v sell=%v", buyRatio, sellRatio)
	}
}

func TestValidateInsufficientSample(t *testing.T) {
	cfg := SampleConfig{MinTradeCount: 20, MinTradeNotionalUSDT: 10000, MinObservationSeconds: 60}
	ok, status := Validate(5, 500, 60, cfg)
	if ok {
		t.Fatal("expected insufficient sample to fail validation")
	}
	if status != SampleInsufficient {
		t.Fatalf("status = %v, want %v", status, SampleInsufficient)
	}
}

func TestValidateSufficientSample(t *testing.T) {
	cfg := SampleConfig{MinTradeCount: 20, MinTradeNotionalUSDT: 10000, MinObservationSeconds: 60}
	ok, status := Validate(25, 15000, 60, cfg)
	if !ok || status != SampleValid {
		t.Fatalf("expected valid sample, got ok=%v status=%v", ok, status)
	}
}

func TestNegativeCVDSlopeOnlyReportsDecline(t *testing.T) {
	points := []CVDPoint{
		{TimestampUnix: 0, CumulativeCVD: 100},
		{TimestampUnix: 10, CumulativeCVD: 200},
	}
	if slope := NegativeCVDSlope(points); slope != 0 {
		t.Fatalf("rising CVD should report 0 slope, got %v", slope)
	}

	decliningPoints := []CVDPoint{
		{TimestampUnix: 0, CumulativeCVD: 200},
		{TimestampUnix: 10, CumulativeCVD: 100},
	}
	if slope := NegativeCVDSlope(decliningPoints); slope >= 0 {
		t.Fatalf("declining CVD should report negative slope, got %v", slope)
	}
}

func TestLargeTradesThreshold(t *testing.T) {
	trades := []domain.Trade{
		trade("sell", 6000, 1),
		trade("sell", 100, 2),
		trade("buy", 8000, 3),
	}
	buyCount, sellCount, buyNotional, sellNotional := LargeTrades(trades)
	if sellCount != 1 || sellNotional != 6000 {
		t.Fatalf("sell large trades = %d/%v, want 1/6000", sellCount, sellNotional)
	}
	if buyCount != 1 || buyNotional != 8000 {
		t.Fatalf("buy large trades = %d/%v, want 1/8000", buyCount, buyNotional)
	}
}

func TestComputeSampleStatusPropagates(t *testing.T) {
	cfg := SampleConfig{MinTradeCount: 100, MinTradeNotionalUSDT: 1000000, MinObservationSeconds: 60}
	trades := []domain.Trade{trade("sell", 100, 1)}
	snapshot := Compute("BTC_USDT", time.Minute, trades, nil, PriceContext{}, cfg)
	if snapshot.SampleStatus != SampleInsufficient {
		t.Fatalf("expected insufficient sample status, got %v", snapshot.SampleStatus)
	}
}
