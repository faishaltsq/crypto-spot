package tradeflow

import "time"

// SellFlowSnapshot is the executed-trade evidence for one symbol over one
// observation window. All notional values are quote-currency (USDT), not
// raw base-token quantity, so pairs are comparable.
type SellFlowSnapshot struct {
	Symbol string        `json:"symbol"`
	Window time.Duration `json:"windowSeconds"`

	AggressiveSellVolume float64 `json:"aggressiveSellVolume"`
	AggressiveBuyVolume  float64 `json:"aggressiveBuyVolume"`
	AggressiveSellRatio  float64 `json:"aggressiveSellRatio"`
	AggressiveBuyRatio   float64 `json:"aggressiveBuyRatio"`

	SellVolumeDelta       float64 `json:"sellVolumeDelta"`
	VolumeDeltaRatio      float64 `json:"volumeDeltaRatio"`
	CumulativeVolumeDelta float64 `json:"cumulativeVolumeDelta"`
	NegativeCVDSlope      float64 `json:"negativeCvdSlope"`

	TradeFrequency   float64 `json:"tradeFrequency"`
	AverageTradeSize float64 `json:"averageTradeSize"`

	LargeSellTradeCount    int     `json:"largeSellTradeCount"`
	LargeSellTradeNotional float64 `json:"largeSellTradeNotional"`
	LargeBuyTradeCount     int     `json:"largeBuyTradeCount"`
	LargeBuyTradeNotional  float64 `json:"largeBuyTradeNotional"`

	DownsideVolumeExpansion float64 `json:"downsideVolumeExpansion"`
	SellPressurePersistence float64 `json:"sellPressurePersistence"`
	PriceResponseToSelling  float64 `json:"priceResponseToSelling"`
	PriceResponseToBuying   float64 `json:"priceResponseToBuying"`
	SellingAbsorption       float64 `json:"sellingAbsorption"`
	BuyingAbsorption        float64 `json:"buyingAbsorption"`
	SellExhaustion          float64 `json:"sellExhaustion"`
	BuyRecovery             float64 `json:"buyRecovery"`

	TradeCount    int       `json:"tradeCount"`
	TotalNotional float64   `json:"totalNotional"`
	SampleStatus  string    `json:"sampleStatus"` // VALID or INSUFFICIENT_SAMPLE
	CalculatedAt  time.Time `json:"calculatedAt"`
}

const (
	SampleValid        = "VALID"
	SampleInsufficient = "INSUFFICIENT_SAMPLE"
)

// MultiWindowSnapshot holds SellFlowSnapshot computed at every required
// observation window (10s, 1m, 5m, 15m, 1h) for one symbol.
type MultiWindowSnapshot struct {
	Symbol  string                      `json:"symbol"`
	Windows map[string]SellFlowSnapshot `json:"windows"`
}

// StandardWindows is the fixed set of trade-flow observation windows.
func StandardWindows() map[string]time.Duration {
	return map[string]time.Duration{
		"10s": 10 * time.Second,
		"1m":  time.Minute,
		"5m":  5 * time.Minute,
		"15m": 15 * time.Minute,
		"1h":  time.Hour,
	}
}
