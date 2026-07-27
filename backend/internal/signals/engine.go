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
	"github.com/google/uuid"
)

// Signal gate thresholds.
// These are the source of truth for what constitutes a confirmed signal.
const (
	// MinRuleScoreForSetup — minimum rule score to enter BUY_SETUP
	MinRuleScoreForSetup = 70.0

	// MinRuleScoreForConfirmed — minimum rule score to produce BUY_CONFIRMED
	MinRuleScoreForConfirmed = 80.0

	// MinDataQualityForSignal — below this data quality, no signal can be issued
	MinDataQualityForSignal = 75.0

	// MaxSpoofScoreForConfirmed — spoof score above this blocks BUY_CONFIRMED
	MaxSpoofScoreForConfirmed = 60.0

	// MinTrendAlignmentForConfirmed — below this, can produce SETUP but not CONFIRMED
	MinTrendAlignmentForConfirmed = 0.20

	// BurstWindow — minimum time between signals across all pairs (anti-burst)
	BurstWindow = 3 * time.Second

	// MaxActiveSignalsGlobal — hard cap on total active signals
	MaxActiveSignalsGlobal = 15
)

// burstGuard prevents signal bursts across ALL pairs.
type burstGuard struct {
	mu            sync.Mutex
	lastSignalAt  time.Time
	lastMinuteAt  time.Time
	countThisMin  int
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
	minScore       float64
	confirmScore   float64
	cooldown       time.Duration
	maxPerMin      int
	ai             *ai.Client
	qualitySvc     *quality.Service
	metrics        *quality.Metrics
	mu             sync.Mutex
	last           map[string]time.Time // per-pair cooldown
	burst          burstGuard
	activeCount    int
}

func New(minScore float64, cooldown time.Duration, aiClient *ai.Client, qualitySvc *quality.Service, metrics *quality.Metrics) *Engine {
	return &Engine{
		minScore:     minScore,
		confirmScore: MinRuleScoreForConfirmed,
		cooldown:     cooldown,
		maxPerMin:    5, // hard cap: max 5 signals per minute across all pairs
		ai:           aiClient,
		qualitySvc:   qualitySvc,
		metrics:      metrics,
		last:         make(map[string]time.Time),
	}
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
			if e.metrics != nil {
				e.metrics.RecordSignalDecision(false)
			}
			log.Printf("[signal] BLOCKED %s: quality gate denied (reasons: %v)", feature.Symbol, reasons)
			return nil, false
		}
	}

	// ── GATE 2: Data quality must be valid ───────────────────────────────────
	if feature.DataQualityScore < MinDataQualityForSignal {
		rejectionReasons = append(rejectionReasons, domain.ReasonLowDataQuality)
	}
	// DataQualityStatus must be VALID or DEGRADED (not STALE, BLOCKED, UNAVAILABLE)
	if !feature.IsDataReady() {
		rejectionReasons = append(rejectionReasons, domain.ReasonLowDataQuality)
		log.Printf("[signal] REJECTED %s: data quality status=%s score=%.1f missing=%v",
			feature.Symbol, feature.DataQualityStatus, feature.DataQualityScore, feature.MissingFeatures)
		return nil, false
	}

	// ── GATE 3: Blocked by explicit blocked reasons ───────────────────────────
	if len(feature.BlockedReasons) > 0 {
		log.Printf("[signal] REJECTED %s: blocked by feature engine: %v", feature.Symbol, feature.BlockedReasons)
		return nil, false
	}

	// ── GATE 4: Minimum rule score for any signal ─────────────────────────────
	if feature.RuleScore < e.minScore {
		rejectionReasons = append(rejectionReasons, domain.ReasonInsufficientRuleScore)
	}

	// ── GATE 5: Valid candidate status ───────────────────────────────────────
	if feature.Status != "BUY_SETUP" && feature.Status != "BUY_CONFIRMED_CANDIDATE" {
		if len(rejectionReasons) > 0 {
			return nil, false
		}
		return nil, false
	}

	// Reject if any gate failed so far
	if len(rejectionReasons) > 0 {
		return nil, false
	}

	// ── GATE 6: Per-pair cooldown ─────────────────────────────────────────────
	e.mu.Lock()
	last := e.last[feature.Symbol]
	if !last.IsZero() && time.Since(last) < e.cooldown {
		e.mu.Unlock()
		log.Printf("[signal] COOLDOWN %s: next allowed at %s", feature.Symbol, last.Add(e.cooldown).Format(time.RFC3339))
		return nil, false
	}
	e.mu.Unlock()

	// ── GATE 7: Anti-burst (global) ───────────────────────────────────────────
	if !e.burst.allow(e.maxPerMin) {
		log.Printf("[signal] BURST_LIMIT %s: global signal rate limit hit", feature.Symbol)
		return nil, false
	}

	// ── GATE 8: AI review ────────────────────────────────────────────────────
	review := e.ai.Review(ctx, feature)
	if review.Decision == "REJECT" {
		if e.metrics != nil {
			e.metrics.RecordSignalDecision(false)
		}
		log.Printf("[signal] AI_REJECTED %s: AI returned REJECT (confidence=%.2f)", feature.Symbol, review.Confidence)
		return nil, false
	}

	// ── DETERMINE SIGNAL TYPE ─────────────────────────────────────────────────
	// BUY_CONFIRMED requires ALL of:
	//   1. rule_score >= MinRuleScoreForConfirmed (80)
	//   2. AI decision == CONFIRM
	//   3. spoof_score <= MaxSpoofScoreForConfirmed (60)
	//   4. trend_alignment >= MinTrendAlignmentForConfirmed (0.20)
	//   5. orderbook synced (already ensured by data quality above)
	signalType := "BUY_SETUP"
	confirmBlockedReasons := make([]string, 0)

	if feature.RuleScore < e.confirmScore {
		confirmBlockedReasons = append(confirmBlockedReasons, domain.ReasonInsufficientRuleScore)
	}
	if review.Decision != "CONFIRM" {
		confirmBlockedReasons = append(confirmBlockedReasons, domain.ReasonMissingAIReview)
	}
	if feature.SpoofScore > MaxSpoofScoreForConfirmed {
		confirmBlockedReasons = append(confirmBlockedReasons, domain.ReasonHighSpoofRisk)
	}
	if feature.TrendAlignment < MinTrendAlignmentForConfirmed {
		confirmBlockedReasons = append(confirmBlockedReasons, domain.ReasonLowTrendAlignment)
	}

	if len(confirmBlockedReasons) == 0 {
		signalType = "BUY_CONFIRMED"
	}

	// Status maps to signal type
	status := "SETUP"
	if signalType == "BUY_CONFIRMED" {
		status = "CONFIRMED"
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
	threshold := domain.ThresholdDetail{
		BaseThreshold:     e.minScore,
		ConfirmThreshold:  e.confirmScore,
		RegimePenalty:     0,
		SpoofPenalty:      0,
		FinalThreshold:    e.minScore,
		SignalScore:        feature.RuleScore,
		TrendAlignmentPct:  feature.TrendAlignment,
		DataQualityScore:  feature.DataQualityScore,
		DataQualityStatus: feature.DataQualityStatus,
		SpoofScore:        feature.SpoofScore,
		SpoofStatus:       feature.SpoofStatus,
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

	signal := &domain.Signal{
		ID:                uuid.NewString(),
		Symbol:            feature.Symbol,
		Type:              signalType,
		Status:            status,
		PrimaryTimeframe:  choosePrimaryTimeframe(feature),
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
		Threshold:         threshold,
		DataQualityScore:  dataQualityScore,
		DataQualityStatus: dataQualityStatus,
		DataSource:        feature.DataSource,
		CreatedAt:         now,
		ExpiresAt:         now.Add(2 * time.Hour),
	}

	e.mu.Lock()
	e.last[feature.Symbol] = now
	e.mu.Unlock()

	log.Printf("[signal] ISSUED %s type=%s score=%.1f dq=%.1f dqStatus=%s spoof=%.1f trendAlign=%.2f ai=%s",
		feature.Symbol, signalType, feature.RuleScore, dataQualityScore, dataQualityStatus,
		feature.SpoofScore, feature.TrendAlignment, review.Decision)

	return signal, true
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
