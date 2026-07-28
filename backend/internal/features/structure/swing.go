package structure

import "github.com/example/crypto-spot-signal/internal/domain"

// ClosedCandles filters to only fully-closed candles, in chronological
// order. The SELL engine must never evaluate structure against an
// in-progress candle (spec requirement: "closed candle confirmation").
func ClosedCandles(candles []domain.Candle) []domain.Candle {
	out := make([]domain.Candle, 0, len(candles))
	for _, c := range candles {
		if c.Closed && c.Close > 0 {
			out = append(out, c)
		}
	}
	return out
}

// SwingPoint is a local high or low within a closed-candle series.
type SwingPoint struct {
	Index int
	Price float64
}

// swingLookback is how many candles on each side must be lower/higher for a
// candle to count as a local swing high/low. 2 is a standard, conservative
// value: it requires a real local extreme rather than single-candle noise.
const swingLookback = 2

// SwingHighs finds local swing highs in a closed-candle series.
func SwingHighs(candles []domain.Candle) []SwingPoint {
	var highs []SwingPoint
	for i := swingLookback; i < len(candles)-swingLookback; i++ {
		isHigh := true
		for j := i - swingLookback; j <= i+swingLookback; j++ {
			if j == i {
				continue
			}
			if candles[j].High > candles[i].High {
				isHigh = false
				break
			}
		}
		if isHigh {
			highs = append(highs, SwingPoint{Index: i, Price: candles[i].High})
		}
	}
	return highs
}

// SwingLows finds local swing lows in a closed-candle series.
func SwingLows(candles []domain.Candle) []SwingPoint {
	var lows []SwingPoint
	for i := swingLookback; i < len(candles)-swingLookback; i++ {
		isLow := true
		for j := i - swingLookback; j <= i+swingLookback; j++ {
			if j == i {
				continue
			}
			if candles[j].Low < candles[i].Low {
				isLow = false
				break
			}
		}
		if isLow {
			lows = append(lows, SwingPoint{Index: i, Price: candles[i].Low})
		}
	}
	return lows
}

// DetectLowerHigh compares the two most recent swing highs. Returns
// (detected, latest, prior).
func DetectLowerHigh(highs []SwingPoint) (bool, float64, float64) {
	if len(highs) < 2 {
		return false, 0, 0
	}
	prior := highs[len(highs)-2]
	latest := highs[len(highs)-1]
	return latest.Price < prior.Price, latest.Price, prior.Price
}

// DetectLowerLow compares the two most recent swing lows. Returns
// (detected, latest, prior).
func DetectLowerLow(lows []SwingPoint) (bool, float64, float64) {
	if len(lows) < 2 {
		return false, 0, 0
	}
	prior := lows[len(lows)-2]
	latest := lows[len(lows)-1]
	return latest.Price < prior.Price, latest.Price, prior.Price
}
