package sell

import (
	"log"
	"sync"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/signals/threshold"
	"github.com/google/uuid"
)

// Engine is the SELL-side counterpart to signals.Engine. It shares no
// mutable state with the BUY engine (separate burst guard, separate
// per-pair cooldowns, separate active-signal counters) so BUY and SELL
// issuance never interfere with each other, even though they observe the
// same market data.
type Engine struct {
	cfg      EngineConfig
	burst    BurstGuard
	cooldown *Cooldown

	mu          sync.Mutex
	activeCount int
}

func New(cfg EngineConfig) *Engine {
	return &Engine{cfg: cfg, cooldown: NewCooldown()}
}

// Evaluate runs one FeatureSnapshot through the SELL gate and returns a
// Result for each signal type that fired. A single call can produce at most
// one signal (SELL_CONFIRMED/SELL_SETUP take priority over AVOID_ENTRY/
// EXIT_WARNING/TAKE_PROFIT_SUGGESTED for the same pair in the same cycle),
// matching the BUY engine's single-signal-per-evaluation contract.
func (e *Engine) Evaluate(f FeatureSnapshot, buyCtx ActiveBuyContext) (*domain.Signal, bool) {
	if !e.cfg.Enabled {
		return nil, false
	}

	rejectionReasons := e.hardGates(f)
	if len(rejectionReasons) > 0 {
		return nil, false
	}

	ruleScore := RuleScore(f)
	thresholdResult := threshold.Calculate(e.cfg.ThresholdConfig, thresholdInputFrom(f, ruleScore))

	shouldRateLimit := !thresholdResult.Blocked && thresholdResult.Passed

	// Decide which signal family applies for this pair given BUY context.
	// Priority order matches urgency: an open position facing a bearish
	// reversal (EXIT_WARNING) outranks a routine take-profit suggestion; a
	// forming BUY candidate facing bearish evidence (AVOID_ENTRY) outranks
	// the pair-level informational SELL signal.
	switch {
	case buyCtx.HasActivePosition:
		if sig, ok := e.evaluateExitWarning(f, ruleScore, buyCtx, shouldRateLimit); ok {
			return sig, true
		}
		if sig, ok := e.evaluateTakeProfit(f, buyCtx, shouldRateLimit); ok {
			return sig, true
		}
		// Position held but neither an exit warning nor a take-profit fired:
		// still surface the pair-level informational PROTECTIVE_SELL if the
		// bearish thesis is strong, so a holder isn't left blind to a broad
		// bearish move just because their specific position gates didn't trip.
		return e.evaluateProtectiveSell(f, ruleScore, thresholdResult, shouldRateLimit)
	case buyCtx.HasCandidateSignal:
		// A forming BUY candidate exists on this pair. If the pair is genuinely
		// bearish enough to form a real PROTECTIVE_SELL setup (score >=
		// SetupScore, bearish trend confirmed), that full SELL signal — with
		// its own entry/target/invalidation levels — takes priority over the
		// advisory AVOID_ENTRY. AVOID_ENTRY is only a fallback for the weaker
		// case where bearish evidence is enough to warn against a BUY entry but
		// not enough to stand on its own as a tradeable SELL setup.
		if sig, ok := e.evaluateProtectiveSell(f, ruleScore, thresholdResult, shouldRateLimit); ok {
			return sig, true
		}
		return e.evaluateAvoidEntry(f, ruleScore, shouldRateLimit)
	default:
		return e.evaluateProtectiveSell(f, ruleScore, thresholdResult, shouldRateLimit)
	}
}

// hardGates returns non-empty when any mandatory precondition is missing;
// these can never be overridden by score, matching signals/engine.go's gate
// ordering philosophy (data readiness before scoring).
func (e *Engine) hardGates(f FeatureSnapshot) []string {
	var reasons []string
	if e.cfg.RequireOrderbookSync && !f.OrderbookSynced {
		reasons = append(reasons, domain.ReasonBookNotSynced)
	}
	if f.DataQualityScore < e.cfg.MinDataQuality {
		reasons = append(reasons, domain.ReasonLowDataQuality)
	}
	if e.cfg.RequireClosedCandle && !f.Structure.ClosedCandleConfirmed {
		reasons = append(reasons, domain.ReasonCandleNotClosed)
	}
	if f.TradeFlow.SampleStatus != "VALID" {
		reasons = append(reasons, domain.ReasonInsufficientTradeSample)
	}
	return reasons
}

func thresholdInputFrom(f FeatureSnapshot, ruleScore float64) threshold.Input {
	tier := threshold.TierA
	if f.Tier == 2 {
		tier = threshold.TierB
	} else if f.Tier >= 3 {
		tier = threshold.TierC
	}
	spoof := threshold.SpoofLow
	if f.SpoofStatus == domain.SpoofStatusHigh {
		spoof = threshold.SpoofHigh
	} else if f.SpoofStatus == domain.SpoofStatusMedium {
		spoof = threshold.SpoofModerate
	}
	liquidity := threshold.LiquidityHealthy
	if f.LiquidityScore < 40 {
		liquidity = threshold.LiquidityLow
	} else if f.LiquidityScore < 70 {
		liquidity = threshold.LiquidityModerate
	}
	quality := threshold.DataQuality(f.DataQualityStatus)
	if quality == "UNAVAILABLE" {
		quality = threshold.DataQualityIncomplete
	}
	correlation := threshold.Correlation(f.CorrelationState)
	if correlation == "" {
		correlation = threshold.CorrelationIndependent
	}
	return threshold.Input{
		Tier: tier, Regime: threshold.Regime(f.MarketRegime), VolatilityPercentile: f.VolatilityPercentile,
		SpoofRisk: spoof, Liquidity: liquidity, Correlation: correlation, DataQuality: quality, ActualScore: ruleScore,
	}
}

func (e *Engine) allowIssue(symbol, signalType string, shouldRateLimit bool) bool {
	if !shouldRateLimit {
		return true // blocked/audit-only signals bypass rate limiting, same as BUY engine
	}
	if !e.cooldown.Allow(symbol, signalType, e.cfg.PairCooldown) {
		log.Printf("[sell] COOLDOWN %s type=%s", symbol, signalType)
		return false
	}
	if !e.burst.Allow(e.cfg.MaxNewPerMinute) {
		log.Printf("[sell] BURST_LIMIT %s type=%s", symbol, signalType)
		return false
	}
	return true
}

func newSignalID() string { return uuid.NewString() }

func nowUTC() time.Time { return time.Now().UTC() }
