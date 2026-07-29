package signals

import (
	"context"
	"log"
	"math"
	"sync"
	"time"

	"github.com/example/crypto-spot-signal/internal/ai"
	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/quality"
	"github.com/example/crypto-spot-signal/internal/signals/threshold"
	"github.com/google/uuid"
)

// DefaultConfirmScore is the fallback BUY_CONFIRMED rule-score threshold
// used only when no EngineConfig is supplied (e.g. zero-value construction
// in tests). Production construction always passes an explicit ConfirmScore
// via New(EngineConfig{...}). All live gating reads e.cfg, never a const.
const DefaultConfirmScore = 80.0

// BurstWindow — minimum time between signals across all pairs (anti-burst).
const BurstWindow = 3 * time.Second

// burstGuard prevents signal bursts across ALL pairs.
type burstGuard struct {
	mu           sync.Mutex
	lastSignalAt time.Time
	lastMinuteAt time.Time
	countThisMin int
}

func (b *burstGuard) allow(maxPerMin int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	// Hard burst guard: no two signals within BurstWindow
	if !b.lastSignalAt.IsZero() && now.Sub(b.lastSignalAt) < BurstWindow {
		return false
	}
	// Per-minute rate limit
	if now.Sub(b.lastMinuteAt) >= time.Minute {
		b.lastMinuteAt = now
		b.countThisMin = 0
	}
	if b.countThisMin >= maxPerMin {
		return false
	}
	b.lastSignalAt = now
	b.countThisMin++
	return true
}

type Engine struct {
	ai         *ai.Client
	qualitySvc *quality.Service
	metrics    *quality.Metrics

	mu           sync.Mutex
	cfg          EngineConfig
	thresholdCfg threshold.Config
	last         map[string]time.Time // per-pair cooldown
	burst        burstGuard
}

// New constructs a BUY engine from a resolved EngineConfig. The threshold
// calculator's base is seeded from cfg.ConfirmScore so a zero-value config
// still yields a sane threshold. If cfg.ConfirmScore is zero (unset),
// DefaultConfirmScore is used as the threshold base only.
func New(cfg EngineConfig, aiClient *ai.Client, qualitySvc *quality.Service, metrics *quality.Metrics) *Engine {
	thresholdBase := cfg.ConfirmScore
	if thresholdBase <= 0 {
		thresholdBase = DefaultConfirmScore
	}
	thresholdCfg, err := threshold.LoadConfig(thresholdBase)
	if err != nil {
		log.Printf("threshold configuration invalid, using defaults: %v", err)
		thresholdCfg = threshold.DefaultConfig(thresholdBase)
	}
	return &Engine{
		ai:           aiClient,
		qualitySvc:   qualitySvc,
		metrics:      metrics,
		cfg:          cfg,
		thresholdCfg: thresholdCfg,
		last:         make(map[string]time.Time),
	}
}

// SetConfig atomically swaps the live EngineConfig used by Evaluate. This is
// the live-reload entry point invoked when Admin Settings change so runtime
// gating actually reflects stored settings (the fase-2 bug fix). It is
// analogous to SetThresholdConfig but for the engine's own gates.
func (e *Engine) SetConfig(cfg EngineConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	e.mu.Lock()
	e.cfg = cfg
	e.mu.Unlock()
	return nil
}

func (e *Engine) SetThresholdConfig(cfg threshold.Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	e.mu.Lock()
	e.thresholdCfg = cfg
	e.mu.Unlock()
	return nil
}

// Evaluate runs a feature snapshot through the full signal gate.
// Returns (signal, true) only if ALL hard conditions are met.
// Returns (nil, false) with log output if rejected.
func (e *Engine) Evaluate(ctx context.Context, feature domain.FeatureSnapshot) (*domain.Signal, bool) {
	rejectionReasons := make([]string, 0, 4)

	// ── GATE 1: Quality Gate service ─────────────────────────────────────────
	if e.qualitySvc != nil {
		if !e.qualitySvc.IsSignalAllowed(feature.Symbol) {
			reasons := e.qualitySvc.BlockedReason(feature.Symbol)
			for _, reason := range reasons {
				feature.BlockedReasons = append(feature.BlockedReasons, string(reason))
			}
		}
	}

	// ── GATE 2: Data quality must be valid ───────────────────────────────────
	// ── GATE 2: Candidate status ─────────────────────────────────────────────
	// BLOCKED candidates are persisted as audits so API/UI can explain rejection.
	if feature.Status != "BUY_SETUP" && feature.Status != "BUY_CONFIRMED_CANDIDATE" && feature.Status != "BLOCKED" && feature.Status != "WATCH" {
		if len(rejectionReasons) > 0 {
			return nil, false
		}
		return nil, false
	}

	// Snapshot live config + threshold under the lock so a concurrent
	// SetConfig/SetThresholdConfig can't tear reads within one evaluation.
	e.mu.Lock()
	cfg := e.cfg
	thresholdCfg := e.thresholdCfg
	e.mu.Unlock()

	thresholdResult := threshold.Calculate(thresholdCfg, thresholdInput(feature))
	shouldRateLimit := !thresholdResult.Blocked && thresholdResult.Passed && len(feature.BlockedReasons) == 0

	// ── GATE 5: Per-pair cooldown ─────────────────────────────────────────────
	if shouldRateLimit {
		e.mu.Lock()
		last := e.last[feature.Symbol]
		if !last.IsZero() && time.Since(last) < cfg.PairCooldown {
			e.mu.Unlock()
			log.Printf("[signal] COOLDOWN %s: next allowed at %s", feature.Symbol, last.Add(cfg.PairCooldown).Format(time.RFC3339))
			return nil, false
		}
		e.mu.Unlock()
	}

	// ── GATE 6: Anti-burst (global) ───────────────────────────────────────────
	if shouldRateLimit && !e.burst.allow(cfg.MaxNewPerMinute) {
		log.Printf("[signal] BURST_LIMIT %s: global signal rate limit hit", feature.Symbol)
		return nil, false
	}

	// ── GATE 7: AI review metadata ───────────────────────────────────────────
	// Reviewer output cannot reject, promote, or otherwise alter rule-owned lifecycle.
	review := e.ai.Review(ctx, feature)

	// ── DETERMINE SIGNAL TYPE ─────────────────────────────────────────────────
	// BUY_CONFIRMED requires rule-owned conditions only. AI is not a signal gate.
	signalType := "BUY_SETUP"
	confirmBlockedReasons := make([]string, 0)
	confirmBlockedReasons = append(confirmBlockedReasons, feature.BlockedReasons...)

	if !thresholdResult.Passed {
		confirmBlockedReasons = append(confirmBlockedReasons, domain.ReasonInsufficientRuleScore)
	}
	if feature.SpoofScore > cfg.MaxSpoofScore {
		confirmBlockedReasons = append(confirmBlockedReasons, domain.ReasonHighSpoofRisk)
	}
	if feature.TrendAlignment < cfg.MinTrendAlignment {
		confirmBlockedReasons = append(confirmBlockedReasons, domain.ReasonLowTrendAlignment)
	}

	if len(confirmBlockedReasons) == 0 && !thresholdResult.Blocked {
		signalType = "BUY_CONFIRMED"
	}

	// Status maps to signal type
	status := "SETUP"
	if signalType == "BUY_CONFIRMED" {
		status = "CONFIRMED"
	}
	if thresholdResult.Blocked || !thresholdResult.Passed || len(feature.BlockedReasons) > 0 {
		status = "BLOCKED"
	}

	if e.metrics != nil {
		e.metrics.RecordSignalDecision(true)
	}

	now := time.Now()
	entry := feature.Price
	volatility := estimateVolatility(feature)
	invalidation := entry * (1 - volatility)
	target1 := entry * (1 + volatility*1.2)
	target2 := entry * (1 + volatility*2.0)

	// Build structured evidence
	evidence := buildEvidence(feature, review)

	// Threshold detail — fully auditable
	thresholdDetail := domain.ThresholdDetail{
		ThresholdVersion:      thresholdResult.ThresholdVersion,
		BaseThreshold:         thresholdResult.BaseThreshold,
		TierAdjustment:        thresholdResult.TierAdjustment,
		RegimeAdjustment:      thresholdResult.RegimeAdjustment,
		VolatilityAdjustment:  thresholdResult.VolatilityAdjustment,
		SpoofAdjustment:       thresholdResult.SpoofAdjustment,
		LiquidityAdjustment:   thresholdResult.LiquidityAdjustment,
		CorrelationAdjustment: thresholdResult.CorrelationAdjustment,
		FinalThreshold:        thresholdResult.FinalThreshold,
		ActualScore:           thresholdResult.ActualScore,
		Passed:                thresholdResult.Passed,
		BlockedByThreshold:    thresholdResult.Blocked,
		ThresholdReasonCodes:  thresholdResult.ReasonCodes,
		TrendAlignmentPct:     feature.TrendAlignment,
		DataQualityScore:      feature.DataQualityScore,
		DataQualityStatus:     feature.DataQualityStatus,
		SpoofScore:            feature.SpoofScore,
		SpoofStatus:           feature.SpoofStatus,
	}

	// Get data quality score from quality service (most authoritative source)
	dataQualityScore := feature.DataQualityScore
	dataQualityStatus := feature.DataQualityStatus
	if e.qualitySvc != nil {
		if report, ok := e.qualitySvc.GetReport(feature.Symbol); ok {
			dataQualityScore = report.Score
			// Map quality report status to domain status
			switch report.Status {
			case "VALID":
				dataQualityStatus = domain.DataQualityValid
			case "DEGRADED":
				dataQualityStatus = domain.DataQualityDegraded
			case "STALE":
				dataQualityStatus = domain.DataQualityStale
			default:
				dataQualityStatus = domain.DataQualityBlocked
			}
		}
	}

	allBlocked := append(feature.BlockedReasons, confirmBlockedReasons...)
	if thresholdResult.Blocked || !thresholdResult.Passed {
		allBlocked = append(allBlocked, thresholdResult.ReasonCodes...)
	}

	recordKind := "CANDIDATE"
	isActionable := false
	notificationEligible := false
	blockedStage := ""

	if status == "BLOCKED" {
		recordKind = domain.RecordKindBlockedAudit
		blockedStage = "EVALUATION"
	} else if status == "SETUP" {
		recordKind = domain.RecordKindActionableSetup
		isActionable = true
		notificationEligible = true
	} else if status == "CONFIRMED" {
		recordKind = domain.RecordKindActionableConfirmed
		isActionable = true
		notificationEligible = true
	}

	if feature.Status == "WATCH" {
		recordKind = domain.RecordKindWatch
		status = "WATCH" // Override DB status for watch
	}

	signal := &domain.Signal{
		ID:                uuid.NewString(),
		Symbol:            feature.Symbol,
		Type:              signalType,
		Status:            status,
		PrimaryTimeframe:  choosePrimaryTimeframe(feature),
		RecordKind:        recordKind,
		DecisionStage:     "EVALUATED",
		IsActionable:      isActionable,
		NotificationEligible: notificationEligible,
		ActualScore:       thresholdResult.ActualScore,
		FinalThreshold:    thresholdResult.FinalThreshold,
		ScoreMargin:       thresholdResult.ActualScore - thresholdResult.FinalThreshold,
		BlockedStage:      blockedStage,
		EvaluatedAt:       now,
		EntryPrice:        entry,
		Invalidation:      invalidation,
		Target1:           target1,
		Target2:           target2,
		RuleScore:         feature.RuleScore,
		AI:                review,
		Reasons:           feature.Reasons,
		RiskFlags:         feature.RiskFlags,
		MissingFeatures:   feature.MissingFeatures,
		BlockedReasons:    allBlocked,
		Features:          feature,
		Version:           domain.CurrentSignalVersion(),
		Evidence:          evidence,
		Threshold:         thresholdDetail,
		DataQualityScore:  dataQualityScore,
		DataQualityStatus: dataQualityStatus,
		DataSource:        feature.DataSource,
		CreatedAt:         now,
		ExpiresAt:         now.Add(2 * time.Hour),
	}
	signal.Enrich()

	if status != "BLOCKED" {
		e.mu.Lock()
		e.last[feature.Symbol] = now
		e.mu.Unlock()
	}

	log.Printf("[signal] ISSUED %s type=%s score=%.1f dq=%.1f dqStatus=%s spoof=%.1f trendAlign=%.2f ai=%s",
		feature.Symbol, signalType, feature.RuleScore, dataQualityScore, dataQualityStatus,
		feature.SpoofScore, feature.TrendAlignment, review.Decision)

	return signal, true
}

func thresholdInput(feature domain.FeatureSnapshot) threshold.Input {
	tier := threshold.TierA
	if feature.Tier == 2 {
		tier = threshold.TierB
	} else if feature.Tier >= 3 {
		tier = threshold.TierC
	}
	spoof := threshold.SpoofLow
	if feature.SpoofStatus == domain.SpoofStatusHigh {
		spoof = threshold.SpoofHigh
	} else if feature.SpoofStatus == domain.SpoofStatusMedium {
		spoof = threshold.SpoofModerate
	}
	liquidity := threshold.LiquidityHealthy
	if feature.LiquidityScore < 40 {
		liquidity = threshold.LiquidityLow
	} else if feature.LiquidityScore < 70 {
		liquidity = threshold.LiquidityModerate
	}
	quality := threshold.DataQuality(feature.DataQualityStatus)
	if quality == "UNAVAILABLE" {
		quality = threshold.DataQualityIncomplete
	}
	correlation := threshold.Correlation(feature.CorrelationState)
	if correlation == "" {
		correlation = threshold.CorrelationIndependent
	}
	return threshold.Input{Tier: tier, Regime: threshold.Regime(feature.MarketRegime), VolatilityPercentile: feature.VolatilityPercentile, SpoofRisk: spoof, Liquidity: liquidity, Correlation: correlation, DataQuality: quality, ActualScore: feature.RuleScore}
}

func buildEvidence(feature domain.FeatureSnapshot, review domain.AIReview) domain.SignalEvidence {
	var supporting, contradicting []string

	// Supporting evidence
	if feature.TrendAlignment >= 0.35 {
		supporting = append(supporting, domain.ReasonMultiTFAlignment)
	}
	if feature.RelativeVolume1m >= 1.5 {
		supporting = append(supporting, domain.ReasonVolumeExpansion)
	}
	if feature.VolumeDeltaRatio1m >= 0.15 {
		supporting = append(supporting, domain.ReasonPositiveCVD)
	}
	if feature.OrderbookImbalance >= 0.15 {
		supporting = append(supporting, domain.ReasonBidAbsorption)
	}
	if feature.LiquidityScore >= 70 {
		supporting = append(supporting, domain.ReasonAdequateLiquidity)
	}
	if feature.SpoofScore <= 30 {
		supporting = append(supporting, domain.ReasonLowSpoofRisk)
	}

	// Contradicting evidence
	if feature.SpoofScore > 60 {
		contradicting = append(contradicting, domain.ReasonHighSpoofRisk)
	}
	if feature.TrendAlignment < 0 {
		contradicting = append(contradicting, domain.ReasonMultiTFConflict)
	}
	if feature.TrendAlignment >= 0 && feature.TrendAlignment < 0.20 {
		contradicting = append(contradicting, domain.ReasonLowTrendAlignment)
	}
	if feature.LiquidityScore < 40 {
		contradicting = append(contradicting, domain.ReasonLowLiquidity)
	}
	if feature.SpreadBPS > 35 {
		contradicting = append(contradicting, domain.ReasonWideSpread)
	}
	if feature.VolumeDeltaRatio1m < -0.15 {
		contradicting = append(contradicting, domain.ReasonNegativeCVD)
	}
	if len(feature.MissingFeatures) > 0 {
		contradicting = append(contradicting, domain.ReasonDataNotReady)
	}

	// Passed and failed rules (from feature reasons and risk flags)
	passed := make([]string, len(feature.Reasons))
	copy(passed, feature.Reasons)

	failed := make([]string, len(feature.RiskFlags))
	copy(failed, feature.RiskFlags)

	// Combine reason codes
	reasonCodes := append(supporting, contradicting...)
	if review.Decision == "CONFIRM" {
		reasonCodes = append(reasonCodes, "AI_CONFIRMED")
	}

	return domain.SignalEvidence{
		SupportingEvidence:    supporting,
		ContradictingEvidence: contradicting,
		PassedRules:           passed,
		FailedRules:           failed,
		ReasonCodes:           reasonCodes,
	}
}

func estimateVolatility(feature domain.FeatureSnapshot) float64 {
	base := 0.012
	if feature.SpreadBPS > 15 {
		base += 0.004
	}
	if feature.RelativeVolume1m > 2 {
		base += 0.004
	}
	if feature.SpoofScore > 40 {
		base += 0.004
	}
	return math.Min(base, 0.035)
}

func choosePrimaryTimeframe(feature domain.FeatureSnapshot) string {
	if feature.TrendByTimeframe["15m"] == "bullish" {
		return "15m"
	}
	if feature.TrendByTimeframe["5m"] == "bullish" {
		return "5m"
	}
	return "1m"
}
