package outcome

import "time"

// Horizon defines a specific time duration after a signal for measuring performance.
type Horizon string

const (
	Horizon1m  Horizon = "1m"
	Horizon5m  Horizon = "5m"
	Horizon15m Horizon = "15m"
	Horizon30m Horizon = "30m"
	Horizon1h  Horizon = "1h"
	Horizon4h  Horizon = "4h"
	Horizon8h  Horizon = "8h"
	Horizon24h Horizon = "24h"
)

// Duration returns the actual time duration for a horizon.
func (h Horizon) Duration() time.Duration {
	switch h {
	case Horizon1m:
		return time.Minute
	case Horizon5m:
		return 5 * time.Minute
	case Horizon15m:
		return 15 * time.Minute
	case Horizon30m:
		return 30 * time.Minute
	case Horizon1h:
		return time.Hour
	case Horizon4h:
		return 4 * time.Hour
	case Horizon8h:
		return 8 * time.Hour
	case Horizon24h:
		return 24 * time.Hour
	default:
		return 0
	}
}

// AllHorizons returns a list of all standard horizons.
func AllHorizons() []Horizon {
	return []Horizon{
		Horizon1m, Horizon5m, Horizon15m, Horizon30m,
		Horizon1h, Horizon4h, Horizon8h, Horizon24h,
	}
}

// HorizonReturn captures the performance of a signal at a specific time horizon.
type HorizonReturn struct {
	Horizon            Horizon   `json:"horizon"`
	Timestamp          time.Time `json:"timestamp"`
	Price              float64   `json:"price"`
	ReturnPct          float64   `json:"returnPct"`          // Raw return percentage
	NetReturnPct       *float64  `json:"netReturnPct,omitempty"` // Percent; absent when simulation is incomplete
	MaximumFavorable   float64   `json:"maximumFavorable"`   // Highest return up to this horizon (MFE)
	MaximumAdverse     float64   `json:"maximumAdverse"`     // Lowest return up to this horizon (MAE)
	TargetHit          bool      `json:"targetHit"`
	InvalidationHit    bool      `json:"invalidationHit"`
	OutcomeStatus      string    `json:"outcomeStatus"`
}

// Result represents the comprehensive evaluation of a signal's outcome over time.
type Result struct {
	SignalID             string                   `json:"signalId"`
	Symbol               string                   `json:"symbol"`
	EntryPrice           float64                  `json:"entryPrice"`
	CreatedAt            time.Time                `json:"createdAt"`
	EvaluatedAt          time.Time                `json:"evaluatedAt"`
	
	// Multi-horizon performance tracking
	Returns              map[Horizon]HorizonReturn `json:"returns"`
	
	// Global summary for this signal
	MaximumFavorablePct  float64                  `json:"maximumFavorablePct"` // Peak MFE across all time
	MaximumAdversePct    float64                  `json:"maximumAdversePct"`   // Peak MAE across all time
	
	// First occurrence tracking
	TargetHit            bool                     `json:"targetHit"`
	TargetHitAt          *time.Time               `json:"targetHitAt,omitempty"`
	InvalidationHit      bool                     `json:"invalidationHit"`
	InvalidationHitAt    *time.Time               `json:"invalidationHitAt,omitempty"`
	OutcomeStatus        string                   `json:"outcomeStatus"`
}

// Candidate is a signal waiting to be evaluated.
type Candidate struct {
	SignalID     string
	Symbol       string
	EntryPrice   float64
	Target1      float64
	Target2      float64
	Invalidation float64
	CreatedAt    time.Time
}
