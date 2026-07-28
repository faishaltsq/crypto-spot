package spoofing

import "github.com/example/crypto-spot-signal/internal/domain"

// WallThresholdMultiplier flags a price level as a "wall" once its quote
// notional exceeds this multiple of the average level size in the same
// book side. This adapts to each pair's liquidity instead of using one
// fixed USDT figure across all pairs.
const WallThresholdMultiplier = 4.0

// DetectWall finds the largest level on one side of the book and reports it
// as a wall if it clears WallThresholdMultiplier times the average level
// size. Returns (detected, price, quoteNotional).
func DetectWall(levels []domain.Level) (bool, float64, float64) {
	if len(levels) == 0 {
		return false, 0, 0
	}
	var total float64
	var maxQuote float64
	var maxPrice float64
	for _, lvl := range levels {
		quote := lvl.Price * lvl.Amount
		total += quote
		if quote > maxQuote {
			maxQuote = quote
			maxPrice = lvl.Price
		}
	}
	average := total / float64(len(levels))
	if average <= 0 {
		return false, 0, 0
	}
	if maxQuote >= average*WallThresholdMultiplier {
		return true, maxPrice, maxQuote
	}
	return false, 0, 0
}

// WallFailure compares a wall's remembered notional against its current
// notional (or absence) in a fresh snapshot to determine if it was
// consumed/pulled rather than holding. previousQuote/currentQuote must be
// sampled from the same price level across two points in time by the
// caller (the SELL engine tracks wall price+quote per symbol across scan
// cycles). Returns (failed, confidence 0-1).
func WallFailure(previousQuote, currentQuote float64) (bool, float64) {
	if previousQuote <= 0 {
		return false, 0
	}
	if currentQuote >= previousQuote*0.5 {
		return false, 0
	}
	// The more completely the wall vanished, the higher confidence it failed
	// (was consumed by real flow or pulled as spoofing) rather than partially
	// filled by normal trading.
	consumedRatio := 1 - (currentQuote / previousQuote)
	confidence := clamp((consumedRatio-0.5)*2, 0, 1)
	return true, confidence
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
