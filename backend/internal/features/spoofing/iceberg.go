package spoofing

// LevelObservation is one point-in-time snapshot of a single price level's
// remaining quote notional, used to detect iceberg (hidden/replenishing)
// orders across consecutive scan cycles.
type LevelObservation struct {
	QuoteNotional float64
	FilledQuote   float64 // quote notional traded at this price since last observation
}

// IcebergThresholdRefills is how many consecutive replenishments at the same
// price level are required before flagging it as a suspected iceberg,
// rather than reacting to a single refill which could be coincidental.
const IcebergThresholdRefills = 3

// DetectIceberg inspects a sequence of observations at one price level and
// flags it as a suspected iceberg order when the level is repeatedly
// consumed by trading and then replenished back to a similar size, rather
// than depleting like a normal resting order.
func DetectIceberg(observations []LevelObservation) (bool, float64) {
	if len(observations) < IcebergThresholdRefills+1 {
		return false, 0
	}
	refills := 0
	for i := 1; i < len(observations); i++ {
		prev := observations[i-1]
		curr := observations[i]
		if prev.FilledQuote <= 0 {
			continue
		}
		// A refill: notional was consumed by trading (FilledQuote>0) yet the
		// remaining quote at that level is close to what it was before.
		if curr.QuoteNotional >= prev.QuoteNotional*0.85 {
			refills++
		}
	}
	if refills < IcebergThresholdRefills {
		return false, 0
	}
	confidence := clamp(float64(refills)/float64(len(observations)-1), 0, 1)
	return true, confidence
}
