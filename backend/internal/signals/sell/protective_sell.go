package sell

import (
	"log"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/signals/threshold"
)

// priceLevels derives the NOT NULL target_price_1/target_price_2/
// invalidation_price columns for a SELL-family signal. Unlike BUY, SELL's
// real invalidation logic is evidence-based (see invalidation.go) — these
// price levels exist only to satisfy the shared signals table schema and to
// give the UI a visual downside projection / structural stop reference.
// Invalidation price is the nearest bearish structure level that would have
// to reclaim for the thesis to break (prior swing high, or a small volatility
// buffer above entry when no swing high is known).
func priceLevels(f FeatureSnapshot) (target1, target2, invalidation float64) {
	entry := f.Price
	volatility := estimateVolatility(f)
	target1 = entry * (1 - volatility*1.2)
	target2 = entry * (1 - volatility*2.0)
	if f.Structure.PriorHighPrice > entry {
		invalidation = f.Structure.PriorHighPrice
	} else {
		invalidation = entry * (1 + volatility)
	}
	return
}

func estimateVolatility(f FeatureSnapshot) float64 {
	base := 0.012
	if f.SpoofScoreRaw > 40 {
		base += 0.004
	}
	if f.VolatilityPercentile > 80 {
		base += 0.004
	}
	if base > 0.035 {
		base = 0.035
	}
	return base
}

// bearishTrendConfirmed decides whether a pair is bearish-aligned enough to
// justify a PROTECTIVE_SELL. The primary gate is the multi-timeframe
// weightedTrend (TrendAlignment, negative when timeframes agree on a
// downtrend): TrendAlignment <= -minAlignment/100 passes immediately.
//
// The weightedTrend is dominated by high-timeframe EMA9/EMA20 crossovers (1d
// weight 4, 4h weight 3). Early in a selloff price can fall sharply on 1m/5m
// while the daily/4h EMAs have not crossed yet, so TrendAlignment stays above
// the threshold and a genuinely bearish move is missed. To catch that case we
// add a low-timeframe override: if the fast timeframes (5m/15m) are already
// bearish AND aggressive sell flow dominates (AggressiveSellRatio >= 0.60)
// with a negative CVD slope, we treat the pair as bearish-aligned even though
// the slow-timeframe-weighted alignment hasn't caught up. This mirrors how a
// trader reads a fresh breakdown before the higher-timeframe averages roll.
func bearishTrendConfirmed(f FeatureSnapshot, minAlignment float64) bool {
	if f.TrendAlignment <= -minAlignment/100 {
		return true
	}
	lowTFBearish := f.TrendByTimeframe["5m"] == "bearish" || f.TrendByTimeframe["15m"] == "bearish"
	strongSellFlow := f.TradeFlow.AggressiveSellRatio >= 0.60 && f.TradeFlow.NegativeCVDSlope < 0
	// Require the move to not be counter-trended by a bullish higher timeframe;
	// a slightly-positive alignment near zero is acceptable, a clearly bullish
	// alignment is not.
	notCounterTrended := f.TrendAlignment < 0.20
	return lowTFBearish && strongSellFlow && notCounterTrended
}

// evaluateProtectiveSell emits SELL_SETUP/SELL_CONFIRMED for a pair that has
// no BUY-side involvement at all — a pure informational bearish signal used
// by the terminal's Active Signals list independent of any user position.
func (e *Engine) evaluateProtectiveSell(f FeatureSnapshot, ruleScore float64, thresholdResult threshold.Result, shouldRateLimit bool) (*domain.Signal, bool) {
	if ruleScore < e.cfg.SetupScore {
		return nil, false
	}
	if f.SpoofScoreRaw > e.cfg.MaxSpoofScore {
		return nil, false
	}
	if !bearishTrendConfirmed(f, e.cfg.MinTimeframeAlignment) {
		// Not bearish-aligned enough across timeframes.
		return nil, false
	}

	signalType := domain.SellSignalSetup
	status := "SETUP"
	if ruleScore >= e.cfg.ConfirmScore && thresholdResult.Passed && !thresholdResult.Blocked {
		signalType = domain.SellSignalConfirmed
		status = "CONFIRMED"
	}
	if thresholdResult.Blocked {
		status = "BLOCKED"
	}

	if !e.allowIssue(f.Symbol, signalType, shouldRateLimit) {
		return nil, false
	}

	sig := baseSignal(f, signalType, status, ruleScore, thresholdResult)
	log.Printf("[sell] ISSUED %s type=%s score=%.1f", f.Symbol, signalType, ruleScore)
	return sig, true
}

func baseSignal(f FeatureSnapshot, signalType, status string, ruleScore float64, thresholdResult threshold.Result) *domain.Signal {
	now := nowUTC()
	evidence := BuildEvidence(f)
	target1, target2, invalidation := priceLevels(f)
	thresholdDetail := domain.ThresholdDetail{
		ThresholdVersion:     thresholdResult.ThresholdVersion,
		BaseThreshold:        thresholdResult.BaseThreshold,
		TierAdjustment:       thresholdResult.TierAdjustment,
		RegimeAdjustment:     thresholdResult.RegimeAdjustment,
		VolatilityAdjustment: thresholdResult.VolatilityAdjustment,
		SpoofAdjustment:      thresholdResult.SpoofAdjustment,
		LiquidityAdjustment:  thresholdResult.LiquidityAdjustment,
		CorrelationAdjustment: thresholdResult.CorrelationAdjustment,
		FinalThreshold:       thresholdResult.FinalThreshold,
		ActualScore:          thresholdResult.ActualScore,
		Passed:               thresholdResult.Passed,
		BlockedByThreshold:   thresholdResult.Blocked,
		ThresholdReasonCodes: thresholdResult.ReasonCodes,
		TrendAlignmentPct:    f.TrendAlignment,
		DataQualityScore:     f.DataQualityScore,
		DataQualityStatus:    f.DataQualityStatus,
		SpoofScore:           f.SpoofScoreRaw,
		SpoofStatus:          f.SpoofStatus,
	}
	sig := &domain.Signal{
		ID:                newSignalID(),
		Symbol:            f.Symbol,
		Type:              signalType,
		Status:            status,
		PrimaryTimeframe:  choosePrimaryTimeframe(f),
		EntryPrice:        f.Price,
		Target1:           target1,
		Target2:           target2,
		Invalidation:      invalidation,
		RuleScore:         ruleScore,
		Reasons:           evidence.SupportingEvidence,
		RiskFlags:         evidence.ContradictingEvidence,
		MissingFeatures:   f.MissingFeatures,
		Evidence:          evidence,
		Threshold:         thresholdDetail,
		Version:           domain.CurrentSignalVersion(),
		DataQualityScore:  f.DataQualityScore,
		DataQualityStatus: f.DataQualityStatus,
		DataSource:        domain.DataSourceGate,
		CreatedAt:         now,
		ExpiresAt:         now.Add(2 * time.Hour),
	}
	sig.Enrich()
	return sig
}

func choosePrimaryTimeframe(f FeatureSnapshot) string {
	if f.TrendByTimeframe["15m"] == "bearish" {
		return "15m"
	}
	if f.TrendByTimeframe["5m"] == "bearish" {
		return "5m"
	}
	return "1m"
}
