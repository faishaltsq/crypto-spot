package structure

import "github.com/example/crypto-spot-signal/internal/domain"

// SupportLevel returns the most recent significant swing low below the
// current price, used as the reference level for breakdown detection.
func SupportLevel(lows []SwingPoint) float64 {
	if len(lows) == 0 {
		return 0
	}
	return lows[len(lows)-1].Price
}

// DetectSupportBreak checks whether the latest CLOSED candle's close price
// is below the support level. It intentionally checks Close, not Low/wick,
// so a brief intra-candle wick below support does not count as a confirmed
// breakdown — the spec calls this out explicitly ("closed candle
// confirmation").
func DetectSupportBreak(closedCandles []domain.Candle, supportLevel float64) (broken bool, breakPrice float64, confirmed bool) {
	if len(closedCandles) == 0 || supportLevel <= 0 {
		return false, 0, false
	}
	last := closedCandles[len(closedCandles)-1]
	if last.Close < supportLevel {
		return true, last.Close, true
	}
	return false, 0, true
}

// DetectFailedReclaim checks whether price attempted to reclaim a broken
// support level but the most recent closed candle failed to close back
// above it. attempted is true whenever any candle since the break traded
// (High) back above supportLevel; failed is true only when the latest
// closed candle nonetheless closed back below it.
func DetectFailedReclaim(closedCandles []domain.Candle, supportLevel float64, breakIndex int) (attempted, failed bool, reclaimHigh float64) {
	if supportLevel <= 0 || breakIndex < 0 || breakIndex >= len(closedCandles) {
		return false, false, 0
	}
	for i := breakIndex + 1; i < len(closedCandles); i++ {
		if closedCandles[i].High > reclaimHigh {
			reclaimHigh = closedCandles[i].High
		}
		if closedCandles[i].High >= supportLevel {
			attempted = true
		}
	}
	if !attempted {
		return false, false, 0
	}
	last := closedCandles[len(closedCandles)-1]
	failed = last.Close < supportLevel
	return attempted, failed, reclaimHigh
}

// BreakdownFollowThrough measures how far price has continued below the
// support level after the break, as a percentage of the support level
// itself. Used as bearish continuation evidence.
func BreakdownFollowThrough(closedCandles []domain.Candle, supportLevel float64) (followedThrough bool, pct float64) {
	if len(closedCandles) == 0 || supportLevel <= 0 {
		return false, 0
	}
	last := closedCandles[len(closedCandles)-1]
	if last.Close >= supportLevel {
		return false, 0
	}
	pct = (supportLevel - last.Close) / supportLevel * 100
	return pct > 0, pct
}
