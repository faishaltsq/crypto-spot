package tradeflow

import "github.com/example/crypto-spot-signal/internal/domain"

// ComputeDelta sums aggressive buy/sell quote notional within a trade slice.
// It trusts only the exchange-provided taker `Side` field (validated upstream
// in exchange/gate against Gate.io's documented spot trade taker-side
// semantics) and never infers aggression from price-vs-bid/ask comparisons.
func ComputeDelta(trades []domain.Trade) (buyVol, sellVol, delta float64) {
	for _, t := range trades {
		switch t.Side {
		case "buy":
			buyVol += t.Quote
		case "sell":
			sellVol += t.Quote
		default:
			// Unknown/missing taker side: excluded from both sides rather than
			// guessed, so an ambiguous trade never silently becomes aggressive
			// sell or aggressive buy evidence.
		}
	}
	delta = buyVol - sellVol
	return
}

// DeltaRatio returns (buyVol-sellVol)/total, or 0 when there is no volume.
func DeltaRatio(buyVol, sellVol float64) float64 {
	total := buyVol + sellVol
	if total <= 0 {
		return 0
	}
	return (buyVol - sellVol) / total
}

// AggressiveRatios returns the aggressive sell and buy ratio of total quote
// notional traded, matching the spec's aggressive_sell_ratio/aggressive_buy_ratio.
func AggressiveRatios(buyVol, sellVol float64) (buyRatio, sellRatio float64) {
	total := buyVol + sellVol
	if total <= 0 {
		return 0, 0
	}
	return buyVol / total, sellVol / total
}
