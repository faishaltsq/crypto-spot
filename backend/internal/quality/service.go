package quality

import (
	"sync"
	"time"

	"github.com/example/crypto-spot-signal/internal/market"
)

// Service manages data quality evaluation for all active pairs.
// It caches the latest quality report for each pair and provides
// thread-safe access to quality status.
type Service struct {
	evaluator *Evaluator
	cfg       QualityConfig

	mu      sync.RWMutex
	reports map[string]QualityReport // keyed by symbol

	// Runtime metrics (set externally)
	metricsMu            sync.RWMutex
	queueUtilization     float64
	redisLatencyMs       float64
	dbWriteBacklogSize   int
	lastReconnectTimes   map[string]time.Time
	bookResyncInProgress map[string]bool
}

// NewService creates a new quality service.
func NewService(cfg QualityConfig) *Service {
	return &Service{
		evaluator:            NewEvaluator(cfg),
		cfg:                  cfg,
		reports:              make(map[string]QualityReport),
		lastReconnectTimes:   make(map[string]time.Time),
		bookResyncInProgress: make(map[string]bool),
	}
}

// EvaluatePair runs quality checks for a single pair and caches the result.
func (s *Service) EvaluatePair(snapshot market.PairSnapshot) QualityReport {
	input := BuildHealthInput(snapshot)

	// Inject runtime metrics
	s.metricsMu.RLock()
	input.QueueUtilization = s.queueUtilization
	input.RedisLatencyMs = s.redisLatencyMs
	input.DBWriteBacklogSize = s.dbWriteBacklogSize
	if t, ok := s.lastReconnectTimes[snapshot.Symbol]; ok {
		input.LastReconnectAt = t
	}
	if resync, ok := s.bookResyncInProgress[snapshot.Symbol]; ok {
		input.BookResyncInProgress = resync
	}
	s.metricsMu.RUnlock()

	report := s.evaluator.Evaluate(snapshot, input)

	// Update timestamps from previous report
	s.mu.Lock()
	prev, hasPrev := s.reports[snapshot.Symbol]
	if hasPrev {
		report.LastValidAt = prev.LastValidAt
		report.LastDegradedAt = prev.LastDegradedAt
		report.LastBlockedAt = prev.LastBlockedAt
	}
	now := report.EvaluatedAt
	switch report.Status {
	case StatusValid:
		report.LastValidAt = &now
	case StatusDegraded:
		report.LastDegradedAt = &now
	case StatusBlocked:
		report.LastBlockedAt = &now
	}
	s.reports[snapshot.Symbol] = report
	s.mu.Unlock()

	return report
}

// EvaluateAll runs quality checks for all given snapshots.
func (s *Service) EvaluateAll(snapshots []market.PairSnapshot) map[string]QualityReport {
	results := make(map[string]QualityReport, len(snapshots))
	for _, snap := range snapshots {
		results[snap.Symbol] = s.EvaluatePair(snap)
	}
	return results
}

// IsSignalAllowed returns whether signal generation is allowed for the given symbol.
func (s *Service) IsSignalAllowed(symbol string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	report, ok := s.reports[symbol]
	if !ok {
		return false // no quality data = not allowed
	}
	return report.SignalAllowed
}

// GetReport returns the latest quality report for a symbol.
func (s *Service) GetReport(symbol string) (QualityReport, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	report, ok := s.reports[symbol]
	return report, ok
}

// AllReports returns a copy of all cached quality reports.
func (s *Service) AllReports() []QualityReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]QualityReport, 0, len(s.reports))
	for _, r := range s.reports {
		out = append(out, r)
	}
	return out
}

// BlockedReason returns the reasons why signal is blocked for a symbol.
// Returns nil if signal is allowed.
func (s *Service) BlockedReason(symbol string) []ReasonCode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	report, ok := s.reports[symbol]
	if !ok {
		return []ReasonCode{ReasonInsufficientData}
	}
	if report.SignalAllowed {
		return nil
	}
	return report.Reasons
}

// Stats returns summary statistics across all pairs.
func (s *Service) Stats() QualityStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var stats QualityStats
	totalScore := 0.0
	for _, r := range s.reports {
		stats.Total++
		totalScore += r.Score
		switch r.Status {
		case StatusValid:
			stats.Valid++
		case StatusDegraded:
			stats.Degraded++
		case StatusStale:
			stats.Stale++
		case StatusBlocked:
			stats.Blocked++
		case StatusUnsynced:
			stats.Unsynced++
		case StatusIncomplete:
			stats.Incomplete++
		}
		if !r.SignalAllowed {
			stats.BlockingSignals++
		}
	}
	if stats.Total > 0 {
		stats.AvgScore = totalScore / float64(stats.Total)
	}
	return stats
}

// --- Runtime metric setters (called from main loop) ---

// SetQueueUtilization updates the current event queue utilization percentage.
func (s *Service) SetQueueUtilization(pct float64) {
	s.metricsMu.Lock()
	s.queueUtilization = pct
	s.metricsMu.Unlock()
}

// SetRedisLatency updates the current Redis latency measurement.
func (s *Service) SetRedisLatency(ms float64) {
	s.metricsMu.Lock()
	s.redisLatencyMs = ms
	s.metricsMu.Unlock()
}

// SetDBWriteBacklog updates the current database write backlog size.
func (s *Service) SetDBWriteBacklog(size int) {
	s.metricsMu.Lock()
	s.dbWriteBacklogSize = size
	s.metricsMu.Unlock()
}

// SetReconnectTime records when a pair's WebSocket connection was last reconnected.
func (s *Service) SetReconnectTime(symbol string, t time.Time) {
	s.metricsMu.Lock()
	s.lastReconnectTimes[symbol] = t
	s.metricsMu.Unlock()
}

// SetBookResync marks whether a pair's orderbook is currently resyncing.
func (s *Service) SetBookResync(symbol string, inProgress bool) {
	s.metricsMu.Lock()
	s.bookResyncInProgress[symbol] = inProgress
	s.metricsMu.Unlock()
}

// QualityStats holds aggregate quality metrics across all pairs.
type QualityStats struct {
	Total           int     `json:"totalPairs"`
	Valid           int     `json:"validPairs"`
	Degraded        int     `json:"degradedPairs"`
	Stale           int     `json:"stalePairs"`
	Blocked         int     `json:"blockedPairs"`
	Unsynced        int     `json:"unsyncedPairs"`
	Incomplete      int     `json:"incompletePairs"`
	BlockingSignals int     `json:"pairsBlockingSignals"`
	AvgScore        float64 `json:"avgScore"`
}
