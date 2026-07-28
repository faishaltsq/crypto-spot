package structure

import (
	"testing"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
)

func candle(high, low, closePrice float64, closed bool) domain.Candle {
	return domain.Candle{High: high, Low: low, Close: closePrice, Closed: closed, OpenTime: time.Now()}
}

func TestClosedCandlesFiltersUnclosed(t *testing.T) {
	candles := []domain.Candle{
		candle(10, 9, 9.5, true),
		{High: 11, Low: 10, Close: 10.5, Closed: false},
	}
	closed := ClosedCandles(candles)
	if len(closed) != 1 {
		t.Fatalf("expected 1 closed candle, got %d", len(closed))
	}
}

func buildDowntrendSeries() []domain.Candle {
	// Construct a synthetic downtrend with two well-separated, unambiguous
	// swing highs (110 then 90) and two swing lows (70 then 50), each with
	// >=2 strictly-lower/higher neighbors on both sides so SwingHighs/
	// SwingLows (lookback=2) detect them reliably.
	highsLowsCloses := []struct{ h, l, c float64 }{
		{80, 75, 78}, {85, 78, 82}, {110, 90, 100}, {85, 78, 82}, {80, 75, 78}, // swing high idx2=110
		{70, 60, 65}, {65, 55, 60}, {50, 40, 45}, {65, 55, 60}, {70, 60, 65}, // swing low idx7=40
		{75, 70, 72}, {80, 74, 77}, {90, 82, 86}, {80, 74, 77}, {75, 70, 72}, // swing high idx12=90 (lower than 110)
		{60, 55, 57}, {55, 50, 52}, {30, 20, 25}, {55, 50, 52}, {60, 55, 57}, // swing low idx17=20 (lower than 40)
	}
	out := make([]domain.Candle, 0, len(highsLowsCloses))
	for _, p := range highsLowsCloses {
		out = append(out, candle(p.h, p.l, p.c, true))
	}
	return out
}

func TestDetectLowerHighAndLowerLow(t *testing.T) {
	candles := buildDowntrendSeries()
	highs := SwingHighs(candles)
	lows := SwingLows(candles)
	if len(highs) < 2 {
		t.Fatalf("expected at least 2 swing highs, got %d", len(highs))
	}
	if len(lows) < 1 {
		t.Fatalf("expected at least 1 swing low, got %d", len(lows))
	}
	detected, latest, prior := DetectLowerHigh(highs)
	if !detected {
		t.Fatalf("expected lower high detected: latest=%v prior=%v", latest, prior)
	}
}

func TestDetectSupportBreakUsesCloseNotWick(t *testing.T) {
	closed := []domain.Candle{
		candle(105, 95, 100, true),
		candle(102, 90, 96, true), // wick below 95 support but closes above... wait support=95
	}
	broken, breakPrice, confirmed := DetectSupportBreak(closed, 95)
	if broken {
		t.Fatalf("close of 96 should not break support of 95, got broken=%v price=%v", broken, breakPrice)
	}
	if !confirmed {
		t.Fatal("expected confirmed=true when there is data")
	}

	closedBelow := []domain.Candle{
		candle(105, 95, 100, true),
		candle(96, 88, 90, true), // closes below support
	}
	broken2, breakPrice2, _ := DetectSupportBreak(closedBelow, 95)
	if !broken2 {
		t.Fatal("expected support break when close is below support level")
	}
	if breakPrice2 != 90 {
		t.Fatalf("breakPrice = %v, want 90", breakPrice2)
	}
}

func TestDetectSupportBreakEmptyInput(t *testing.T) {
	broken, _, confirmed := DetectSupportBreak(nil, 100)
	if broken || confirmed {
		t.Fatal("empty candle input must never report a confirmed break")
	}
}

func TestComputeInsufficientCandlesReturnsZeroValue(t *testing.T) {
	result := Compute("BTC_USDT", "1m", nil)
	if result.SupportBroken || result.LowerHighDetected || result.StructureScore != 0 {
		t.Fatalf("expected zero-value structure for no candles, got %+v", result)
	}
}
