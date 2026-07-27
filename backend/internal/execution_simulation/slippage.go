package execution_simulation

import "github.com/example/crypto-spot-signal/internal/domain"

type fill struct { quote, base float64; levels int }

func walkQuote(levels []domain.Level, quote float64) fill {
	var result fill
	for _, level := range levels {
		if level.Price <= 0 || level.Amount <= 0 || result.quote >= quote { continue }
		available := level.Price * level.Amount
		take := min(quote-result.quote, available)
		result.quote += take
		result.base += take / level.Price
		result.levels++
	}
	return result
}

func walkBase(levels []domain.Level, base float64) fill {
	var result fill
	for _, level := range levels {
		if level.Price <= 0 || level.Amount <= 0 || result.base >= base { continue }
		take := min(base-result.base, level.Amount)
		result.base += take
		result.quote += take * level.Price
		result.levels++
	}
	return result
}

func min(a, b float64) float64 { if a < b { return a }; return b }
