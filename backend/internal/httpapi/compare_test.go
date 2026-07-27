package httpapi

import (
	"math"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
)

func TestParseCompareRequest(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/compare?pairs=BTC_USDT,ETH_USDT&timeframe=15m&lookback=24h", nil)
	value, err := parseCompareRequest(request)
	if err != nil || len(value.Pairs) != 2 || value.Timeframe != "15m" || value.Lookback != "24h" {
		t.Fatalf("valid request rejected: %#v, %v", value, err)
	}
}

func TestParseCompareRequestAcceptsSnapshotFilters(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/compare?pairs=BTC_USDT,ETH_USDT&timeframe=15m&lookback=24h&marketTier=1&minimumDataQuality=80&activeSignalOnly=true", nil)
	value, err := parseCompareRequest(request)
	if err != nil || value.MarketTier != 1 || value.MinimumQuality == nil || *value.MinimumQuality != 80 || !value.ActiveSignalOnly {
		t.Fatalf("valid filters rejected: %#v, %v", value, err)
	}
}

func TestCompareOutcomeHorizonMapsOneDayToPersistedTwentyFourHours(t *testing.T) {
	if got := compareOutcomeHorizon("1d"); got != "24h" {
		t.Fatalf("1d mapped to %q", got)
	}
	if got := compareOutcomeHorizon("15m"); got != "15m" {
		t.Fatalf("15m mapped to %q", got)
	}
}

func TestParseCompareRequestRejectsBoundsDuplicatesAndInvalidPairs(t *testing.T) {
	for _, query := range []string{
		"pairs=BTC_USDT&timeframe=15m&lookback=24h",
		"pairs=BTC_USDT,ETH_USDT,SOL_USDT,XRP_USDT,DOGE_USDT&timeframe=15m&lookback=24h",
		"pairs=BTC_USDT,BTC_USDT&timeframe=15m&lookback=24h",
		"pairs=BTC_USDT,INVALID&timeframe=15m&lookback=24h",
		"pairs=BTC_USDT,ETH_USDT&timeframe=2m&lookback=24h",
		"pairs=BTC_USDT,ETH_USDT&timeframe=15m&lookback=2h",
	} {
		if _, err := parseCompareRequest(httptest.NewRequest("GET", "/api/v1/compare?"+query, nil)); err == nil {
			t.Fatalf("invalid query accepted: %s", query)
		}
	}
}

func TestNormalizedPerformanceUsesFirstCloseAndSkipsInvalidClose(t *testing.T) {
	points := normalizedPerformance([]domain.Candle{{OpenTime: time.Unix(1, 0), Close: 0}, {OpenTime: time.Unix(2, 0), Close: 100}, {OpenTime: time.Unix(3, 0), Close: 110}})
	if len(points) != 2 || points[0].Value != 0 || math.Abs(points[1].Value-10) > 0.000001 {
		t.Fatalf("unexpected normalized points: %#v", points)
	}
}

func TestCandlesSinceHonorsLookbackBoundary(t *testing.T) {
	base := time.Unix(100, 0)
	candles := []domain.Candle{{OpenTime: base}, {OpenTime: base.Add(time.Minute)}, {OpenTime: base.Add(2 * time.Minute)}}
	got := candlesSince(candles, base.Add(time.Minute))
	if len(got) != 2 || !got[0].OpenTime.Equal(base.Add(time.Minute)) {
		t.Fatalf("unexpected candles: %#v", got)
	}
}

func TestCompareCacheRetainsSnapshotTimestampUntilExpiry(t *testing.T) {
	cache := newCompareCache(time.Second)
	now := time.Now()
	value := CompareResponse{SnapshotAt: now.Format(time.RFC3339Nano)}
	cache.put("key", value, now)
	got, ok := cache.get("key", now.Add(500*time.Millisecond))
	if !ok || got.SnapshotAt != value.SnapshotAt {
		t.Fatal("cache did not return same snapshot")
	}
	if _, ok := cache.get("key", now.Add(time.Second)); ok {
		t.Fatal("expired cache returned a value")
	}
}

func TestCompareCacheEvictsExpiredEntries(t *testing.T) {
	cache := newCompareCache(time.Second)
	now := time.Now()
	cache.put("expired", CompareResponse{}, now.Add(-2*time.Second))
	cache.put("current", CompareResponse{}, now)
	cache.get("current", now)
	if _, exists := cache.entries["expired"]; exists {
		t.Fatal("expired cache entry was retained")
	}
}
