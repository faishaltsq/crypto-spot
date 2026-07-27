package features

import (
	"math"
	"sort"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/market"
)

type Config struct {
	MaxSpreadBPS  float64
	MinDepthQuote float64
}

type Engine struct {
	cfg Config
}

func New(cfg Config) *Engine {
	return &Engine{cfg: cfg}
}

func (e *Engine) Compute(snapshot market.PairSnapshot) domain.FeatureSnapshot {
	return e.ComputeWithSource(snapshot, domain.DataSourceGate)
}

// ComputeWithSource computes features and tags the result with the data source.
func (e *Engine) ComputeWithSource(snapshot market.PairSnapshot, src domain.DataSource) domain.FeatureSnapshot {
	now := time.Now()
	trades1m := market.TradeWindow(snapshot.Trades, time.Minute)
	price := snapshot.LastPrice
	if price == 0 {
		price = snapshot.Book.MidPrice
	}

	buyRatio := 0.5
	if trades1m.TotalQuote > 0 {
		buyRatio = trades1m.BuyQuote / trades1m.TotalQuote
	}

	relativeVolume := relativeVolume1m(snapshot.Candles["1m"], trades1m.TotalQuote)
	trendByTF := make(map[string]string)
	ema9ByTF := make(map[string]float64)
	ema20ByTF := make(map[string]float64)

	timeframes := make([]string, 0, len(snapshot.Candles))
	for timeframe := range snapshot.Candles {
		timeframes = append(timeframes, timeframe)
	}
	sort.Slice(timeframes, func(i, j int) bool {
		return timeframeWeight(timeframes[i]) < timeframeWeight(timeframes[j])
	})

	weightedTrend := 0.0
	totalTrendWeight := 0.0
	for _, timeframe := range timeframes {
		closes := closedCloses(snapshot.Candles[timeframe])
		ema9 := ema(closes, 9)
		ema20 := ema(closes, 20)
		ema9ByTF[timeframe] = ema9
		ema20ByTF[timeframe] = ema20

		trend := "neutral"
		if len(closes) >= 9 && ema9 > ema20 && closes[len(closes)-1] >= ema9 {
			trend = "bullish"
		} else if len(closes) >= 9 && ema9 < ema20 && closes[len(closes)-1] <= ema9 {
			trend = "bearish"
		}
		trendByTF[timeframe] = trend

		weight := trendWeight(timeframe)
		totalTrendWeight += weight
		switch trend {
		case "bullish":
			weightedTrend += weight
		case "bearish":
			weightedTrend -= weight
		}
	}

	trendAlignment := 0.0
	if totalTrendWeight > 0 {
		trendAlignment = weightedTrend / totalTrendWeight
	}

	liquidityScore := scoreLiquidity(snapshot.Book, e.cfg)
	volumeScore := clamp((relativeVolume-0.5)*30+(buyRatio-0.5)*50+50, 0, 100)
	orderFlowScore := clamp(50+trades1m.DeltaRatio*45+snapshot.Book.Imbalance*35, 0, 100)
	trendScore := clamp(50+trendAlignment*50, 0, 100)

	// ── DATA QUALITY CALCULATION ────────────────────────────────────────────
	// BUG FIX: replaced hardcoded 100.0 with real penalty-based calculation.
	// Every missing/stale/bad condition subtracts points from 100.
	dataQuality := 100.0
	missingFeatures := make([]string, 0)
	blockedReasons := make([]string, 0)

	// Critical: orderbook not synced (–45)
	if !snapshot.Book.Synced {
		dataQuality -= 45
		missingFeatures = append(missingFeatures, domain.ReasonMissingOrderbook)
		blockedReasons = append(blockedReasons, domain.ReasonMissingOrderbook)
	}

	// Stale market data: last update > 30s ago (–35)
	if snapshot.LastMarketUpdate.IsZero() || now.Sub(snapshot.LastMarketUpdate) > 30*time.Second {
		dataQuality -= 35
		missingFeatures = append(missingFeatures, domain.ReasonStaleMarketData)
		blockedReasons = append(blockedReasons, domain.ReasonStaleMarketData)
	}

	// Insufficient 1m candle history (–15)
	if len(snapshot.Candles["1m"]) < 20 {
		dataQuality -= 15
		missingFeatures = append(missingFeatures, domain.ReasonLimitedCandleHistory)
	}

	// Low recent trade activity (–10)
	if trades1m.Count < 3 {
		dataQuality -= 10
		missingFeatures = append(missingFeatures, domain.ReasonLowRecentTradeCount)
	}

	// Missing multi-timeframe data: need at least 3 timeframes (–15)
	// Important timeframes for trend analysis
	requiredTFs := []string{"5m", "15m", "1h"}
	missingTFCount := 0
	for _, tf := range requiredTFs {
		if len(snapshot.Candles[tf]) < 9 {
			missingTFCount++
		}
	}
	if missingTFCount >= 2 {
		dataQuality -= 15
		missingFeatures = append(missingFeatures, domain.ReasonMissingTimeframes)
		blockedReasons = append(blockedReasons, domain.ReasonMissingMultiTimeframe)
	}

	// No price available (–20)
	if price <= 0 {
		dataQuality -= 20
		missingFeatures = append(missingFeatures, "MISSING_PRICE")
		blockedReasons = append(blockedReasons, "MISSING_PRICE")
	}

	dataQuality = clamp(dataQuality, 0, 100)
	// ────────────────────────────────────────────────────────────────────────

	baseScore := trendScore*0.30 + volumeScore*0.22 + orderFlowScore*0.23 + liquidityScore*0.15 + dataQuality*0.10
	spoofPenalty := snapshot.Book.SpoofScore * 0.18
	overextensionPenalty := overextensionPenalty(price, ema20ByTF["15m"], ema20ByTF["1h"])
	ruleScore := clamp(baseScore-spoofPenalty-overextensionPenalty, 0, 100)

	reasons := make([]string, 0)
	if trendAlignment >= 0.35 {
		reasons = append(reasons, domain.ReasonMultiTFAlignment)
	}
	if relativeVolume >= 1.5 {
		reasons = append(reasons, domain.ReasonVolumeExpansion)
	}
	if trades1m.DeltaRatio >= 0.15 {
		reasons = append(reasons, domain.ReasonPositiveCVD)
	}
	if snapshot.Book.Imbalance >= 0.15 {
		reasons = append(reasons, domain.ReasonBidAbsorption)
	}
	if snapshot.Book.SpoofScore <= 30 {
		reasons = append(reasons, domain.ReasonLowSpoofRisk)
	}
	if liquidityScore >= 70 {
		reasons = append(reasons, domain.ReasonAdequateLiquidity)
	}

	riskFlags := make([]string, 0)
	if snapshot.Book.SpreadBPS > e.cfg.MaxSpreadBPS {
		riskFlags = append(riskFlags, domain.ReasonWideSpread)
	}
	if snapshot.Book.BidDepthQuote+snapshot.Book.AskDepthQuote < e.cfg.MinDepthQuote*2 {
		riskFlags = append(riskFlags, domain.ReasonLowLiquidity)
	}
	if snapshot.Book.SpoofScore > 60 {
		riskFlags = append(riskFlags, domain.ReasonHighSpoofRisk)
		blockedReasons = append(blockedReasons, domain.ReasonHighSpoofRisk)
	}
	if trendAlignment < 0 {
		riskFlags = append(riskFlags, domain.ReasonMultiTFConflict)
	}
	if trendAlignment < 0.20 && trendAlignment >= 0 {
		riskFlags = append(riskFlags, domain.ReasonLowTrendAlignment)
	}

	// Derive DataQualityStatus
	dqStatus := dataQualityStatus(dataQuality)

	// Status determination — only advance to candidate if data is ready
	status := "NO_SIGNAL"
	switch {
	case dataQuality < 60:
		status = "WAITING_DATA"
	case len(blockedReasons) > 0:
		status = "BLOCKED"
	case ruleScore >= 80:
		status = "BUY_CONFIRMED_CANDIDATE"
	case ruleScore >= 70:
		status = "BUY_SETUP"
	case ruleScore >= 60:
		status = "WATCH"
	}

	return domain.FeatureSnapshot{
		Symbol:             snapshot.Symbol,
		Tier:               snapshot.Tier,
		DataSource:         src,
		Price:              price,
		Change24hPercent:   snapshot.Change24hPercent,
		QuoteVolume24h:     snapshot.QuoteVolume24h,
		SpreadBPS:          snapshot.Book.SpreadBPS,
		BidDepthQuote:      snapshot.Book.BidDepthQuote,
		AskDepthQuote:      snapshot.Book.AskDepthQuote,
		OrderbookImbalance: snapshot.Book.Imbalance,
		SpoofScore:         snapshot.Book.SpoofScore,
		SpoofStatus:        domain.SpoofStatusFrom(snapshot.Book.SpoofScore),
		BuyRatio1m:         buyRatio,
		VolumeDeltaRatio1m: trades1m.DeltaRatio,
		RelativeVolume1m:   relativeVolume,
		TrendByTimeframe:   trendByTF,
		EMA9ByTimeframe:    ema9ByTF,
		EMA20ByTimeframe:   ema20ByTF,
		TrendAlignment:     trendAlignment,
		LiquidityScore:     liquidityScore,
		VolumeScore:        volumeScore,
		OrderFlowScore:     orderFlowScore,
		TrendScore:         trendScore,
		DataQualityScore:   dataQuality,
		DataQualityStatus:  dqStatus,
		RuleScore:          ruleScore,
		Status:             status,
		Reasons:            reasons,
		RiskFlags:          unique(riskFlags),
		MissingFeatures:    missingFeatures,
		BlockedReasons:     unique(blockedReasons),
		CalculatedAt:       now,
	}
}

// dataQualityStatus maps a numeric data quality score to a status label.
func dataQualityStatus(score float64) domain.DataQualityStatus {
	switch {
	case score >= 90:
		return domain.DataQualityValid
	case score >= 75:
		return domain.DataQualityDegraded
	case score >= 50:
		return domain.DataQualityStale
	default:
		return domain.DataQualityBlocked
	}
}


func scoreLiquidity(book domain.BookMetrics, cfg Config) float64 {
	if !book.Synced || book.MidPrice <= 0 {
		return 0
	}
	spreadScore := 100.0
	if cfg.MaxSpreadBPS > 0 {
		spreadScore = clamp(100-(book.SpreadBPS/cfg.MaxSpreadBPS)*100, 0, 100)
	}
	depth := book.BidDepthQuote + book.AskDepthQuote
	depthScore := 100.0
	if cfg.MinDepthQuote > 0 {
		depthScore = clamp((depth/(cfg.MinDepthQuote*2))*100, 0, 100)
	}
	return spreadScore*0.55 + depthScore*0.45
}

func relativeVolume1m(candles []domain.Candle, currentQuote float64) float64 {
	volumes := make([]float64, 0, 20)
	for i := len(candles) - 1; i >= 0 && len(volumes) < 20; i-- {
		if candles[i].Closed && candles[i].QuoteVolume > 0 {
			volumes = append(volumes, candles[i].QuoteVolume)
		}
	}
	if len(volumes) < 5 {
		if currentQuote > 0 {
			return 1
		}
		return 0
	}
	total := 0.0
	for _, volume := range volumes {
		total += volume
	}
	average := total / float64(len(volumes))
	if average <= 0 {
		return 0
	}
	return currentQuote / average
}

func closedCloses(candles []domain.Candle) []float64 {
	closes := make([]float64, 0, len(candles))
	for _, candle := range candles {
		if candle.Closed && candle.Close > 0 {
			closes = append(closes, candle.Close)
		}
	}
	return closes
}

func ema(values []float64, period int) float64 {
	if len(values) == 0 {
		return 0
	}
	if len(values) < period {
		total := 0.0
		for _, value := range values {
			total += value
		}
		return total / float64(len(values))
	}
	alpha := 2.0 / float64(period+1)
	result := values[0]
	for _, value := range values[1:] {
		result = alpha*value + (1-alpha)*result
	}
	return result
}

func overextensionPenalty(price, ema15m, ema1h float64) float64 {
	penalty := 0.0
	for _, reference := range []float64{ema15m, ema1h} {
		if price <= 0 || reference <= 0 {
			continue
		}
		distance := math.Abs(price-reference) / reference * 100
		if distance > 6 {
			penalty += math.Min((distance-6)*2.5, 15)
		}
	}
	return penalty
}

func trendWeight(timeframe string) float64 {
	switch timeframe {
	case "1d":
		return 4
	case "4h", "8h":
		return 3
	case "1h":
		return 2.5
	case "15m", "30m":
		return 1.5
	default:
		return 1
	}
}

func timeframeWeight(timeframe string) int {
	switch timeframe {
	case "10s":
		return 1
	case "1m":
		return 2
	case "5m":
		return 3
	case "15m":
		return 4
	case "30m":
		return 5
	case "1h":
		return 6
	case "4h":
		return 7
	case "8h":
		return 8
	case "1d":
		return 9
	case "7d":
		return 10
	default:
		return 99
	}
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func unique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
