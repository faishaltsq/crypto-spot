package httpapi

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/market"
	"github.com/example/crypto-spot-signal/internal/storage"
)

const compareMinSamples = 20
const compareCacheMaxEntries = 64

var compareTimeframes = map[string]bool{"1m": true, "5m": true, "15m": true, "30m": true, "1h": true, "4h": true, "8h": true, "1d": true}
var compareLookbacks = map[string]time.Duration{"1h": time.Hour, "4h": 4 * time.Hour, "24h": 24 * time.Hour, "7d": 7 * 24 * time.Hour, "30d": 30 * 24 * time.Hour}

type compareRequest struct {
	Pairs            []string
	Timeframe        string
	Lookback         string
	MarketTier       int
	MinimumQuality   *float64
	ActiveSignalOnly bool
}
type compareCacheEntry struct {
	response  CompareResponse
	expiresAt time.Time
}
type compareCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]compareCacheEntry
}

func newCompareCache(ttl time.Duration) *compareCache {
	return &compareCache{ttl: ttl, entries: map[string]compareCacheEntry{}}
}
func (c *compareCache) get(key string, now time.Time) (CompareResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for cacheKey, entry := range c.entries {
		if !now.Before(entry.expiresAt) {
			delete(c.entries, cacheKey)
		}
	}
	entry, ok := c.entries[key]
	return entry.response, ok && now.Before(entry.expiresAt)
}
func (c *compareCache) put(key string, value CompareResponse, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for cacheKey, entry := range c.entries {
		if !now.Before(entry.expiresAt) {
			delete(c.entries, cacheKey)
		}
	}
	if len(c.entries) >= compareCacheMaxEntries {
		for cacheKey := range c.entries {
			delete(c.entries, cacheKey)
			break
		}
	}
	c.entries[key] = compareCacheEntry{response: value, expiresAt: now.Add(c.ttl)}
}

type CompareResponse struct {
	SnapshotAt      string               `json:"snapshotAt"`
	Timeframe       string               `json:"timeframe"`
	Lookback        string               `json:"lookback"`
	Pairs           []ComparePair        `json:"pairs"`
	Unavailable     []CompareUnavailable `json:"unavailable"`
	CacheTTLSeconds int                  `json:"cacheTtlSeconds"`
	Filters         CompareFilters       `json:"filters"`
}
type CompareFilters struct {
	NormalizePerformance   bool     `json:"normalizePerformance"`
	MarketTier             int      `json:"marketTier,omitempty"`
	MinimumDataQuality     *float64 `json:"minimumDataQuality,omitempty"`
	ActiveSignalOnly       bool     `json:"activeSignalOnly"`
	WatchlistOnlyAvailable bool     `json:"watchlistOnlyAvailable"`
}
type CompareUnavailable struct {
	Symbol  string `json:"symbol"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
type ComparePair struct {
	Symbol                string            `json:"symbol"`
	Rank                  int               `json:"rank"`
	Tier                  int               `json:"tier"`
	Price                 float64           `json:"price"`
	Change24hPercent      float64           `json:"change24hPercent"`
	QuoteVolume24h        float64           `json:"quoteVolume24h"`
	RelativeVolume        *float64          `json:"relativeVolume"`
	SpreadBPS             *float64          `json:"spreadBps"`
	BidDepthQuote         *float64          `json:"bidDepthQuote"`
	AskDepthQuote         *float64          `json:"askDepthQuote"`
	LiquidityScore        *float64          `json:"liquidityScore"`
	Slippage100           *float64          `json:"estimatedSlippage100"`
	Slippage500           *float64          `json:"estimatedSlippage500"`
	CVD                   *float64          `json:"cvd"`
	OrderbookImbalance    *float64          `json:"orderbookImbalance"`
	SpoofScore            *float64          `json:"spoofScore"`
	Trend                 *string           `json:"trend"`
	Momentum              *float64          `json:"momentum"`
	ATRPercent            *float64          `json:"atrPercent"`
	TrendAlignment        *float64          `json:"multiTimeframeAlignment"`
	SignalScore           *float64          `json:"signalScore"`
	DynamicThreshold      *float64          `json:"dynamicThreshold"`
	DataQualityScore      *float64          `json:"dataQualityScore"`
	DataQualityStatus     string            `json:"dataQualityStatus"`
	ActiveSignal          *bool             `json:"activeSignal"`
	Historical            CompareHistorical `json:"historical"`
	PricePerformance      []ComparePoint    `json:"pricePerformance"`
	SupportingEvidence    []string          `json:"supportingEvidence"`
	ContradictingEvidence []string          `json:"contradictingEvidence"`
	Freshness             CompareFreshness  `json:"freshness"`
	PartialMetrics        []string          `json:"partialMetrics"`
}
type ComparePoint struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
}
type CompareFreshness struct {
	LastMarketUpdate string `json:"lastMarketUpdate"`
	IsStale          bool   `json:"isStale"`
	BookSynced       bool   `json:"bookSynced"`
}
type CompareHistorical struct {
	NetReturn          *float64 `json:"netReturn"`
	WinRate            *float64 `json:"winRate"`
	SampleCount        int      `json:"sampleCount"`
	NetExpectancy      *float64 `json:"netExpectancy"`
	MFE                *float64 `json:"mfe"`
	MAE                *float64 `json:"mae"`
	InsufficientSample bool     `json:"insufficientSample"`
}

func parseCompareRequest(r *http.Request) (compareRequest, error) {
	pairsRaw := strings.TrimSpace(r.URL.Query().Get("pairs"))
	if pairsRaw == "" {
		return compareRequest{}, fmt.Errorf("pairs is required")
	}
	seen := map[string]bool{}
	pairs := make([]string, 0, 4)
	for _, raw := range strings.Split(pairsRaw, ",") {
		symbol := strings.ToUpper(strings.TrimSpace(raw))
		if symbol == "" {
			continue
		}
		if seen[symbol] {
			return compareRequest{}, fmt.Errorf("duplicate pair: %s", symbol)
		}
		if !strings.HasSuffix(symbol, "_USDT") || len(symbol) <= len("_USDT") {
			return compareRequest{}, fmt.Errorf("invalid pair: %s", symbol)
		}
		seen[symbol] = true
		pairs = append(pairs, symbol)
	}
	if len(pairs) < 2 {
		return compareRequest{}, fmt.Errorf("at least 2 pairs are required")
	}
	if len(pairs) > 4 {
		return compareRequest{}, fmt.Errorf("maximum 4 pairs are allowed")
	}
	timeframe := r.URL.Query().Get("timeframe")
	if !compareTimeframes[timeframe] {
		return compareRequest{}, fmt.Errorf("invalid timeframe")
	}
	lookback := r.URL.Query().Get("lookback")
	if _, ok := compareLookbacks[lookback]; !ok {
		return compareRequest{}, fmt.Errorf("invalid lookback")
	}
	value := compareRequest{Pairs: pairs, Timeframe: timeframe, Lookback: lookback}
	if raw := r.URL.Query().Get("marketTier"); raw != "" {
		tier, err := strconv.Atoi(raw)
		if err != nil || tier < 1 || tier > 3 {
			return compareRequest{}, fmt.Errorf("invalid market tier")
		}
		value.MarketTier = tier
	}
	if raw := r.URL.Query().Get("minimumDataQuality"); raw != "" {
		quality, err := strconv.ParseFloat(raw, 64)
		if err != nil || quality < 0 || quality > 100 {
			return compareRequest{}, fmt.Errorf("invalid minimum data quality")
		}
		value.MinimumQuality = &quality
	}
	if raw := r.URL.Query().Get("activeSignalOnly"); raw != "" {
		activeOnly, err := strconv.ParseBool(raw)
		if err != nil {
			return compareRequest{}, fmt.Errorf("invalid active signal filter")
		}
		value.ActiveSignalOnly = activeOnly
	}
	return value, nil
}

func (s *Server) compareSnapshot(w http.ResponseWriter, r *http.Request) {
	request, err := parseCompareRequest(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}
	key := fmt.Sprintf("%s|%s|%s|%d|%v|%t", strings.Join(request.Pairs, ","), request.Timeframe, request.Lookback, request.MarketTier, request.MinimumQuality, request.ActiveSignalOnly)
	now := time.Now().UTC()
	if response, ok := s.compareCache.get(key, now); ok {
		writeJSON(w, http.StatusOK, response)
		return
	}
	response := s.buildCompareSnapshot(r, request, now)
	s.compareCache.put(key, response, now)
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) buildCompareSnapshot(r *http.Request, request compareRequest, now time.Time) CompareResponse {
	active := map[string]struct{ rank, tier int }{}
	for _, pair := range s.univSvc.ActivePairs() {
		active[pair.Symbol] = struct{ rank, tier int }{pair.Rank, pair.Tier}
	}
	response := CompareResponse{SnapshotAt: now.Format(time.RFC3339Nano), Timeframe: request.Timeframe, Lookback: request.Lookback, Pairs: []ComparePair{}, Unavailable: []CompareUnavailable{}, CacheTTLSeconds: 3, Filters: CompareFilters{NormalizePerformance: true, MarketTier: request.MarketTier, MinimumDataQuality: request.MinimumQuality, ActiveSignalOnly: request.ActiveSignalOnly, WatchlistOnlyAvailable: false}}
	for _, symbol := range request.Pairs {
		meta, isActive := active[symbol]
		if !isActive {
			response.Unavailable = append(response.Unavailable, CompareUnavailable{Symbol: symbol, Code: "INACTIVE_PAIR", Message: "Pair is not active in market universe"})
			continue
		}
		snapshot, ok := s.market.Snapshot(symbol, s.cfg.OrderbookDepthPercent)
		if !ok {
			response.Unavailable = append(response.Unavailable, CompareUnavailable{Symbol: symbol, Code: "UNAVAILABLE_PAIR", Message: "No live market snapshot"})
			continue
		}
		feature, featureOK := s.state.Feature(symbol)
		pair := buildComparePair(snapshot, feature, featureOK, meta.rank, meta.tier, request.Timeframe, now.Add(-compareLookbacks[request.Lookback]), now)
		if s.qualitySvc != nil {
			if report, ok := s.qualitySvc.GetReport(symbol); ok {
				pair.DataQualityScore = floatPtr(report.Score)
				pair.DataQualityStatus = string(report.Status)
			}
		}
		if s.repo != nil {
			history, err := s.repo.CompareHistory(r.Context(), symbol, compareOutcomeHorizon(request.Timeframe), now.Add(-compareLookbacks[request.Lookback]))
			if err == nil {
				pair.Historical = toCompareHistorical(history)
			} else {
				pair.PartialMetrics = append(pair.PartialMetrics, "historical")
			}
			signal, found, err := s.repo.CompareSignalState(r.Context(), symbol)
			if err == nil && found {
				pair.ActiveSignal, pair.DynamicThreshold = &signal.Active, signal.DynamicThreshold
			} else if err != nil {
				pair.PartialMetrics = append(pair.PartialMetrics, "signal")
			}
		}
		if request.MarketTier != 0 && pair.Tier != request.MarketTier {
			continue
		}
		if request.MinimumQuality != nil && (pair.DataQualityScore == nil || *pair.DataQualityScore < *request.MinimumQuality) {
			continue
		}
		if request.ActiveSignalOnly && (pair.ActiveSignal == nil || !*pair.ActiveSignal) {
			continue
		}
		response.Pairs = append(response.Pairs, pair)
	}
	return response
}

func compareOutcomeHorizon(timeframe string) string {
	if timeframe == "1d" {
		return "24h"
	}
	return timeframe
}

func buildComparePair(snapshot market.PairSnapshot, feature domain.FeatureSnapshot, featureOK bool, rank, tier int, timeframe string, since, now time.Time) ComparePair {
	pair := ComparePair{Symbol: snapshot.Symbol, Rank: rank, Tier: tier, Price: snapshot.LastPrice, Change24hPercent: snapshot.Change24hPercent, QuoteVolume24h: snapshot.QuoteVolume24h, DataQualityStatus: "UNAVAILABLE", Historical: CompareHistorical{InsufficientSample: true}}
	pair.SpreadBPS, pair.BidDepthQuote, pair.AskDepthQuote, pair.OrderbookImbalance, pair.SpoofScore = floatPtr(snapshot.Book.SpreadBPS), floatPtr(snapshot.Book.BidDepthQuote), floatPtr(snapshot.Book.AskDepthQuote), floatPtr(snapshot.Book.Imbalance), floatPtr(snapshot.Book.SpoofScore)
	pair.Slippage100, pair.Slippage500 = estimateSlippage(snapshot.Book, 100), estimateSlippage(snapshot.Book, 500)
	pair.Freshness = CompareFreshness{LastMarketUpdate: snapshot.LastMarketUpdate.Format(time.RFC3339Nano), IsStale: snapshot.LastMarketUpdate.IsZero() || now.Sub(snapshot.LastMarketUpdate) > 2*time.Minute, BookSynced: snapshot.Book.Synced}
	pair.PricePerformance = normalizedPerformance(candlesSince(snapshot.Candles[timeframe], since))
	if !featureOK {
		pair.PartialMetrics = append(pair.PartialMetrics, "feature")
		return pair
	}
	cvd := cumulativeVolumeDelta(snapshot.Trades)
	momentum := candleMomentum(snapshot.Candles[timeframe])
	pair.RelativeVolume, pair.LiquidityScore, pair.CVD, pair.Momentum, pair.TrendAlignment, pair.SignalScore = floatPtr(feature.RelativeVolume1m), floatPtr(feature.LiquidityScore), floatPtr(cvd), momentum, floatPtr(feature.TrendAlignment), floatPtr(feature.RuleScore)
	pair.ATRPercent = averageTrueRangePercent(snapshot.Candles[timeframe])
	if trend, ok := feature.TrendByTimeframe[timeframe]; ok {
		pair.Trend = &trend
	} else {
		pair.PartialMetrics = append(pair.PartialMetrics, "trend")
	}
	pair.DataQualityScore, pair.DataQualityStatus = floatPtr(feature.DataQualityScore), string(feature.DataQualityStatus)
	pair.SupportingEvidence, pair.ContradictingEvidence = feature.Reasons, append(append([]string{}, feature.RiskFlags...), feature.BlockedReasons...)
	return pair
}

func candlesSince(candles []domain.Candle, since time.Time) []domain.Candle {
	index := 0
	for index < len(candles) && candles[index].OpenTime.Before(since) {
		index++
	}
	return candles[index:]
}

func normalizedPerformance(candles []domain.Candle) []ComparePoint {
	if len(candles) == 0 {
		return []ComparePoint{}
	}
	base := 0.0
	points := make([]ComparePoint, 0, len(candles))
	for _, candle := range candles {
		if base == 0 && candle.Close > 0 {
			base = candle.Close
		}
		if base > 0 && candle.Close > 0 {
			points = append(points, ComparePoint{Time: candle.OpenTime.Format(time.RFC3339), Value: (candle.Close/base - 1) * 100})
		}
	}
	return points
}
func cumulativeVolumeDelta(trades []domain.Trade) float64 {
	var value float64
	for _, trade := range trades {
		if trade.Side == "buy" {
			value += trade.Quote
		} else {
			value -= trade.Quote
		}
	}
	return value
}
func candleMomentum(candles []domain.Candle) *float64 {
	if len(candles) < 2 || candles[len(candles)-2].Close <= 0 {
		return nil
	}
	return floatPtr((candles[len(candles)-1].Close/candles[len(candles)-2].Close - 1) * 100)
}
func averageTrueRangePercent(candles []domain.Candle) *float64 {
	if len(candles) < 2 || candles[len(candles)-1].Close <= 0 {
		return nil
	}
	start := len(candles) - 15
	if start < 1 {
		start = 1
	}
	var total float64
	for index := start; index < len(candles); index++ {
		previous, current := candles[index-1].Close, candles[index]
		total += math.Max(current.High-current.Low, math.Max(math.Abs(current.High-previous), math.Abs(current.Low-previous)))
	}
	return floatPtr(total / float64(len(candles)-start) / candles[len(candles)-1].Close * 100)
}
func estimateSlippage(book domain.BookMetrics, notional float64) *float64 {
	depth := math.Min(book.BidDepthQuote, book.AskDepthQuote)
	if depth <= 0 {
		return nil
	}
	return floatPtr(math.Min(10000, notional/depth*10000))
}
func floatPtr(value float64) *float64 { return &value }

func toCompareHistorical(value storage.CompareHistory) CompareHistorical {
	return CompareHistorical{NetReturn: value.NetReturn, WinRate: value.WinRate, SampleCount: value.SampleCount, NetExpectancy: value.NetExpectancy, MFE: value.MFE, MAE: value.MAE, InsufficientSample: value.SampleCount < compareMinSamples}
}
