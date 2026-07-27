package quality

import (
	"log"
	"sync"
	"sync/atomic"
)

// Metrics tracks quality gate statistics using atomic counters.
// This avoids adding external dependencies for now (Prometheus can be added later).
type Metrics struct {
	mu sync.Mutex

	EvaluationCount atomic.Int64
	ValidCount      atomic.Int64
	DegradedCount   atomic.Int64
	BlockedCount    atomic.Int64

	SignalsAllowed atomic.Int64
	SignalsBlocked atomic.Int64
}

// NewMetrics creates a new metrics tracker.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// RecordEvaluation records the result of a quality evaluation.
func (m *Metrics) RecordEvaluation(report QualityReport) {
	m.EvaluationCount.Add(1)
	switch report.Status {
	case StatusValid:
		m.ValidCount.Add(1)
	case StatusDegraded:
		m.DegradedCount.Add(1)
	case StatusBlocked:
		m.BlockedCount.Add(1)
	}
}

// RecordSignalDecision records whether a signal was allowed or blocked.
func (m *Metrics) RecordSignalDecision(allowed bool) {
	if allowed {
		m.SignalsAllowed.Add(1)
	} else {
		m.SignalsBlocked.Add(1)
	}
}

// LogSummary writes a summary of quality metrics to the log.
func (m *Metrics) LogSummary() {
	log.Printf("[quality] evaluations=%d valid=%d degraded=%d blocked=%d signals_allowed=%d signals_blocked=%d",
		m.EvaluationCount.Load(),
		m.ValidCount.Load(),
		m.DegradedCount.Load(),
		m.BlockedCount.Load(),
		m.SignalsAllowed.Load(),
		m.SignalsBlocked.Load(),
	)
}

// Snapshot returns the current metrics as a serializable struct.
func (m *Metrics) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		Evaluations:    m.EvaluationCount.Load(),
		Valid:          m.ValidCount.Load(),
		Degraded:       m.DegradedCount.Load(),
		Blocked:        m.BlockedCount.Load(),
		SignalsAllowed: m.SignalsAllowed.Load(),
		SignalsBlocked: m.SignalsBlocked.Load(),
	}
}

// MetricsSnapshot is a point-in-time snapshot of quality metrics.
type MetricsSnapshot struct {
	Evaluations    int64 `json:"evaluations"`
	Valid          int64 `json:"valid"`
	Degraded       int64 `json:"degraded"`
	Blocked        int64 `json:"blocked"`
	SignalsAllowed int64 `json:"signalsAllowed"`
	SignalsBlocked int64 `json:"signalsBlocked"`
}
