package spoofing

import (
	"testing"

	"github.com/example/crypto-spot-signal/internal/domain"
)

func level(price, amount float64) domain.Level {
	return domain.Level{Price: price, Amount: amount}
}

func TestDetectWallFindsOutlier(t *testing.T) {
	levels := []domain.Level{
		level(100, 1), level(99, 1), level(98, 50), level(97, 1), level(96, 1),
	}
	detected, price, quote := DetectWall(levels)
	if !detected {
		t.Fatal("expected wall to be detected")
	}
	if price != 98 {
		t.Fatalf("wall price = %v, want 98", price)
	}
	if quote != 98*50 {
		t.Fatalf("wall quote = %v, want %v", quote, 98*50)
	}
}

func TestDetectWallNoOutlierNoWall(t *testing.T) {
	levels := []domain.Level{
		level(100, 10), level(99, 10), level(98, 11), level(97, 10),
	}
	detected, _, _ := DetectWall(levels)
	if detected {
		t.Fatal("expected no wall for evenly distributed book")
	}
}

func TestDetectWallEmptyBook(t *testing.T) {
	detected, _, _ := DetectWall(nil)
	if detected {
		t.Fatal("empty book must never report a wall")
	}
}

func TestWallFailureRequiresSignificantDrop(t *testing.T) {
	failed, _ := WallFailure(10000, 9000) // only 10% consumed
	if failed {
		t.Fatal("small drop should not count as wall failure")
	}

	failed2, confidence := WallFailure(10000, 1000) // 90% consumed
	if !failed2 {
		t.Fatal("large drop should count as wall failure")
	}
	if confidence <= 0 {
		t.Fatalf("expected positive confidence, got %v", confidence)
	}
}

func TestWallFailureNoPreviousWall(t *testing.T) {
	failed, confidence := WallFailure(0, 500)
	if failed || confidence != 0 {
		t.Fatal("no previous wall means no failure to report")
	}
}

func TestDetectIcebergRequiresMultipleRefills(t *testing.T) {
	observations := []LevelObservation{
		{QuoteNotional: 1000, FilledQuote: 0},
		{QuoteNotional: 950, FilledQuote: 900},
		{QuoteNotional: 980, FilledQuote: 850},
		{QuoteNotional: 970, FilledQuote: 900},
		{QuoteNotional: 960, FilledQuote: 880},
	}
	detected, confidence := DetectIceberg(observations)
	if !detected {
		t.Fatal("expected iceberg pattern to be detected")
	}
	if confidence <= 0 {
		t.Fatalf("expected positive confidence, got %v", confidence)
	}
}

func TestDetectIcebergInsufficientObservations(t *testing.T) {
	observations := []LevelObservation{{QuoteNotional: 1000, FilledQuote: 500}}
	detected, _ := DetectIceberg(observations)
	if detected {
		t.Fatal("single observation must never trigger iceberg detection")
	}
}

func TestTrackerAnalyzeDetectsBidWallFailureAcrossCycles(t *testing.T) {
	tracker := NewTracker()
	// Level at price 98 with amount 500 => quote 49000, average of the other
	// three tiny levels is ~1, comfortably clearing WallThresholdMultiplier.
	bids1 := []domain.Level{
		level(100, 0.1), level(99, 0.1), level(98, 5000), level(97, 0.1), level(96, 0.1), level(95, 0.1),
	}
	first := tracker.Analyze("BTC_USDT", bids1, nil, domain.BookMetrics{})
	if !first.BidWallDetected {
		t.Fatalf("expected wall detected on first cycle, got %+v", first)
	}

	bids2 := []domain.Level{
		level(100, 0.1), level(99, 0.1), level(98, 1), level(97, 0.1), level(96, 0.1), level(95, 0.1),
	}
	result := tracker.Analyze("BTC_USDT", bids2, nil, domain.BookMetrics{})
	if !result.BidWallFailed {
		t.Fatalf("expected bid wall failure to be detected on second cycle, got %+v", result)
	}
}
