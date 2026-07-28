package structure

import (
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
)

// Compute builds a full BearishStructure for one symbol/timeframe from
// closed candles. Callers must pass at least (2*swingLookback+1)*2 candles
// for swing detection to find two comparable swing points; fewer candles
// yield a zero-value, non-detected structure rather than a fabricated one.
func Compute(symbol, timeframe string, candles []domain.Candle) BearishStructure {
	closed := ClosedCandles(candles)
	result := BearishStructure{
		Symbol:                symbol,
		Timeframe:             timeframe,
		ClosedCandleConfirmed: len(closed) > 0,
		CalculatedAt:          time.Now(),
	}
	if len(closed) == 0 {
		return result
	}

	highs := SwingHighs(closed)
	lows := SwingLows(closed)

	result.LowerHighDetected, result.LowerHighPrice, result.PriorHighPrice = DetectLowerHigh(highs)
	result.LowerLowDetected, result.LowerLowPrice, result.PriorLowPrice = DetectLowerLow(lows)

	support := SupportLevel(lows)
	result.SupportLevel = support

	broken, breakPrice, confirmed := DetectSupportBreak(closed, support)
	result.SupportBroken = broken
	result.SupportBrokenPrice = breakPrice
	result.ClosedCandleConfirmed = confirmed

	if broken {
		breakIndex := len(closed) - 1
		attempted, failed, reclaimHigh := DetectFailedReclaim(closed, support, breakIndex-1)
		result.ReclaimAttempted = attempted
		result.ReclaimFailed = failed
		result.ReclaimHighPrice = reclaimHigh

		followed, pct := BreakdownFollowThrough(closed, support)
		result.BreakdownFollowThrough = followed
		result.FollowThroughPct = pct
	}

	result.StructureScore = score(result)
	return result
}

// score combines the individual bearish structure signals into a single
// 0-100 evidence strength score. Weighting favors confirmed support breaks
// (the strongest, least ambiguous evidence) over swing pattern shape alone.
func score(s BearishStructure) float64 {
	total := 0.0
	if s.LowerHighDetected {
		total += 20
	}
	if s.LowerLowDetected {
		total += 20
	}
	if s.SupportBroken && s.ClosedCandleConfirmed {
		total += 35
	}
	if s.ReclaimFailed {
		total += 15
	}
	if s.BreakdownFollowThrough {
		total += 10
	}
	if total > 100 {
		total = 100
	}
	return total
}
