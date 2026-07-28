package tradeflow

import (
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
)

// PriceContext supplies the price-change inputs needed for absorption
// scoring; it comes from the candle/orderbook layer, not from trades.
type PriceContext struct {
	PriceChangePct            float64 // over the window, negative = decline
	ExpectedDeclinePctPerUSDT float64 // pair-specific volatility/liquidity calibration
	ExpectedRisePctPerUSDT    float64
}

// Compute builds a full SellFlowSnapshot for one symbol/window from raw
// executed spot trades. trades must already be filtered to the window by
// the caller (see market.TradeWindow-style slicing). priorWindowTrades is
// the immediately preceding equal-length window, used for exhaustion/
// recovery deltas; it may be nil if unavailable, in which case those fields
// are left at zero rather than fabricated.
func Compute(symbol string, window time.Duration, trades, priorWindowTrades []domain.Trade, price PriceContext, sampleCfg SampleConfig) SellFlowSnapshot {
	now := time.Now()
	snapshot := SellFlowSnapshot{
		Symbol:       symbol,
		Window:       window,
		TradeCount:   len(trades),
		CalculatedAt: now,
	}

	buyVol, sellVol, delta := ComputeDelta(trades)
	snapshot.AggressiveBuyVolume = buyVol
	snapshot.AggressiveSellVolume = sellVol
	snapshot.SellVolumeDelta = delta
	snapshot.TotalNotional = buyVol + sellVol
	snapshot.VolumeDeltaRatio = DeltaRatio(buyVol, sellVol)
	snapshot.AggressiveBuyRatio, snapshot.AggressiveSellRatio = AggressiveRatios(buyVol, sellVol)

	cvd, points := CumulativeVolumeDelta(trades)
	snapshot.CumulativeVolumeDelta = cvd
	snapshot.NegativeCVDSlope = NegativeCVDSlope(points)

	snapshot.AverageTradeSize = AverageTradeSize(trades)
	snapshot.TradeFrequency = TradeFrequency(trades, window.Seconds())

	buyCount, sellCount, buyNotional, sellNotional := LargeTrades(trades)
	snapshot.LargeBuyTradeCount = buyCount
	snapshot.LargeSellTradeCount = sellCount
	snapshot.LargeBuyTradeNotional = buyNotional
	snapshot.LargeSellTradeNotional = sellNotional

	snapshot.SellingAbsorption = SellingAbsorption(price.PriceChangePct, sellVol, price.ExpectedDeclinePctPerUSDT)
	snapshot.BuyingAbsorption = BuyingAbsorption(price.PriceChangePct, buyVol, price.ExpectedRisePctPerUSDT)

	if priorWindowTrades != nil {
		priorBuyVol, priorSellVol, _ := ComputeDelta(priorWindowTrades)
		priorDeltaRatio := DeltaRatio(priorBuyVol, priorSellVol)
		snapshot.SellExhaustion = SellExhaustion(sellVol, priorSellVol, snapshot.VolumeDeltaRatio, priorDeltaRatio)
		snapshot.BuyRecovery = BuyRecovery(buyVol, priorBuyVol, snapshot.VolumeDeltaRatio, priorDeltaRatio)
		if priorSellVol > 0 {
			snapshot.DownsideVolumeExpansion = clamp((sellVol/priorSellVol)-1, 0, 3)
		}
		if snapshot.VolumeDeltaRatio < 0 && priorDeltaRatio < 0 {
			// Sell pressure persisted across both windows.
			snapshot.SellPressurePersistence = clamp((-snapshot.VolumeDeltaRatio-(-priorDeltaRatio))/2+0.5, 0, 1)
		}
	}

	snapshot.PriceResponseToSelling = price.PriceChangePct
	if buyVol > 0 {
		snapshot.PriceResponseToBuying = -price.PriceChangePct
	}

	_, sampleStatus := Validate(len(trades), snapshot.TotalNotional, window.Seconds(), sampleCfg)
	snapshot.SampleStatus = sampleStatus

	return snapshot
}
