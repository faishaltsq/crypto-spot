package tradeflow

// SampleConfig defines the minimum sample requirements before trade-flow
// derived scores can be trusted (mirrors SELL_MIN_TRADE_COUNT,
// SELL_MIN_TRADE_NOTIONAL_USDT, SELL_MIN_OBSERVATION_SECONDS).
type SampleConfig struct {
	MinTradeCount         int
	MinTradeNotionalUSDT  float64
	MinObservationSeconds int
}

// Validate checks whether a trade sample is large enough to trust derived
// sell-flow scores. It never fabricates data: callers must treat every
// derived field as untrustworthy when this returns false, rather than
// treating a zero value as "no selling pressure".
func Validate(tradeCount int, totalNotional float64, windowSeconds float64, cfg SampleConfig) (bool, string) {
	if tradeCount < cfg.MinTradeCount {
		return false, SampleInsufficient
	}
	if totalNotional < cfg.MinTradeNotionalUSDT {
		return false, SampleInsufficient
	}
	if windowSeconds < float64(cfg.MinObservationSeconds) {
		return false, SampleInsufficient
	}
	return true, SampleValid
}
