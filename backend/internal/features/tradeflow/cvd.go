package tradeflow

import (
	"sort"

	"github.com/example/crypto-spot-signal/internal/domain"
)

// CVDPoint is one cumulative-volume-delta observation used for slope
// estimation (negative_cvd_slope).
type CVDPoint struct {
	TimestampUnix int64
	CumulativeCVD float64
}

// CumulativeVolumeDelta walks trades in chronological order and returns the
// running buy-minus-sell quote notional (CVD) plus the per-point series used
// to derive the slope.
func CumulativeVolumeDelta(trades []domain.Trade) (float64, []CVDPoint) {
	ordered := make([]domain.Trade, len(trades))
	copy(ordered, trades)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Timestamp.Before(ordered[j].Timestamp)
	})

	var cvd float64
	points := make([]CVDPoint, 0, len(ordered))
	for _, t := range ordered {
		switch t.Side {
		case "buy":
			cvd += t.Quote
		case "sell":
			cvd -= t.Quote
		}
		points = append(points, CVDPoint{TimestampUnix: t.Timestamp.Unix(), CumulativeCVD: cvd})
	}
	return cvd, points
}

// NegativeCVDSlope performs a simple least-squares slope fit over the CVD
// series and returns the slope only when it is negative; a positive or flat
// slope returns 0 (there is no "negative slope" to report).
func NegativeCVDSlope(points []CVDPoint) float64 {
	n := len(points)
	if n < 2 {
		return 0
	}
	var sumX, sumY, sumXY, sumXX float64
	base := points[0].TimestampUnix
	for _, p := range points {
		x := float64(p.TimestampUnix - base)
		y := p.CumulativeCVD
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}
	nF := float64(n)
	denominator := nF*sumXX - sumX*sumX
	if denominator == 0 {
		return 0
	}
	slope := (nF*sumXY - sumX*sumY) / denominator
	if slope >= 0 {
		return 0
	}
	return slope
}
