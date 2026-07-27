package outcome

import (
	"math"
	"time"
)

// EvaluateHorizon calculates the performance of a signal at a specific horizon.
func EvaluateHorizon(
	candidate Candidate,
	horizon Horizon,
	currentPrice float64,
	obs PriceObservation,
	currentTime time.Time,
) HorizonReturn {
	targetHit := obs.High >= candidate.Target1
	invalidationHit := obs.Low <= candidate.Invalidation

	// Raw return based on current price
	returnPct := CalculateExcursionPct(candidate.EntryPrice, currentPrice)
	
	// Max excursions based on the observation tracker
	mfe := CalculateExcursionPct(candidate.EntryPrice, obs.High)
	mae := CalculateExcursionPct(candidate.EntryPrice, obs.Low)

	// In a real simulation, if Invalidation is hit, we might cap the return at the MAE or invalidation point.
	// For raw outcome reporting, we just record the facts.
	
	// Note: NetReturnPct is calculated later by combining this with Execution Simulation results.

	return HorizonReturn{
		Horizon:          horizon,
		Timestamp:        currentTime,
		Price:            currentPrice,
		ReturnPct:        returnPct,
		MaximumFavorable: math.Max(mfe, 0), // MFE should be positive
		MaximumAdverse:   math.Min(mae, 0), // MAE should be negative
		TargetHit:        targetHit,
		InvalidationHit:  invalidationHit,
		OutcomeStatus:    "EVALUATED",
	}
}

// EvaluateTotal calculates the global outcome for a signal, aggregating all its horizons.
func EvaluateTotal(candidate Candidate, returns map[Horizon]HorizonReturn) Result {
	res := Result{
		SignalID:        candidate.SignalID,
		Symbol:          candidate.Symbol,
		EntryPrice:      candidate.EntryPrice,
		CreatedAt:       candidate.CreatedAt,
		EvaluatedAt:     time.Now(),
		Returns:         returns,
		TargetHit:       false,
		InvalidationHit: false,
		OutcomeStatus:   "EVALUATED",
	}

	maxMFE := 0.0
	minMAE := 0.0

	// Find global MFE/MAE and first target/invalidation hits
	for _, ret := range returns {
		if ret.MaximumFavorable > maxMFE {
			maxMFE = ret.MaximumFavorable
		}
		if ret.MaximumAdverse < minMAE {
			minMAE = ret.MaximumAdverse
		}

		if ret.TargetHit && !res.TargetHit {
			res.TargetHit = true
			res.TargetHitAt = &ret.Timestamp
		}
		if ret.InvalidationHit && !res.InvalidationHit {
			res.InvalidationHit = true
			res.InvalidationHitAt = &ret.Timestamp
		}
	}

	res.MaximumFavorablePct = maxMFE
	res.MaximumAdversePct = minMAE

	return res
}
