package universe

import (
	"math"
	"sort"
)

type PairCandidate struct {
	Symbol         string
	Base           string
	Quote          string
	QuoteVolume24h float64
	BestBid        float64
	BestAsk        float64
}

// CalculateSpreadBps calculates spread in basis points.
func CalculateSpreadBps(bid, ask float64) float64 {
	if bid <= 0 || ask <= 0 || bid >= ask {
		return 10000 // Invalid spread
	}
	mid := (bid + ask) / 2.0
	return ((ask - bid) / mid) * 10000.0
}

// RankCandidates ranks candidates based on Quote Volume (60%), Spread (30%), and Trade Activity proxy (10%).
func RankCandidates(candidates []PairCandidate, maxSpreadBps float64) []RankedPair {
	if len(candidates) == 0 {
		return nil
	}

	var maxVol float64
	var validCandidates []PairCandidate
	for _, c := range candidates {
		spread := CalculateSpreadBps(c.BestBid, c.BestAsk)
		if spread > maxSpreadBps || spread >= 10000 {
			continue
		}
		if c.QuoteVolume24h > maxVol {
			maxVol = c.QuoteVolume24h
		}
		validCandidates = append(validCandidates, c)
	}

	var ranked []RankedPair
	for _, c := range validCandidates {
		spread := CalculateSpreadBps(c.BestBid, c.BestAsk)
		
		volScore := 0.0
		if maxVol > 0 {
			volScore = (c.QuoteVolume24h / maxVol) * 100.0
		}
		
		spreadScore := 0.0
		if spread < maxSpreadBps {
			spreadScore = ((maxSpreadBps - spread) / maxSpreadBps) * 100.0
		}

		// Proximate trade activity based on volume score (as a base)
		activityScore := volScore * 0.8 

		finalScore := (volScore * 0.60) + (spreadScore * 0.30) + (activityScore * 0.10)

		ranked = append(ranked, RankedPair{
			Symbol:         c.Symbol,
			RankScore:      finalScore,
			QuoteVolume24h: c.QuoteVolume24h,
			SpreadBps:      spread,
			SelectionReason: map[string]interface{}{
				"volScore":      math.Round(volScore*100)/100,
				"spreadScore":   math.Round(spreadScore*100)/100,
				"activityScore": math.Round(activityScore*100)/100,
			},
		})
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].RankScore > ranked[j].RankScore
	})

	for i := range ranked {
		ranked[i].Rank = i + 1
	}

	return ranked
}
