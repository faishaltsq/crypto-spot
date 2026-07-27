package universe

import (
	"strings"
)

var excludedSuffixes = []string{
	"3L", "3S", "5L", "5S", "BULL", "BEAR", "HALF", "HEDGE",
}

func IsValidSpotPair(base, quote string, minQuoteVolume float64, currentQuoteVolume float64, stablecoins []string, spreadBps float64, maxSpreadBps float64) (bool, string) {
	if quote != "USDT" {
		return false, "Invalid quote currency"
	}
	
	stableMap := make(map[string]bool)
	for _, s := range stablecoins {
		stableMap[s] = true
	}
	
	if stableMap[base] {
		return false, "Stablecoin to stablecoin pair"
	}
	for _, suffix := range excludedSuffixes {
		if strings.HasSuffix(base, suffix) {
			return false, "Leveraged/ETF token"
		}
	}
	if currentQuoteVolume < minQuoteVolume {
		return false, "Low quote volume"
	}
	if spreadBps > maxSpreadBps || spreadBps >= 10000 {
		return false, "Spread too wide or empty order book"
	}
	return true, ""
}
