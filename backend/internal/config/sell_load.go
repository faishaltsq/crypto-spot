package config

// loadSellConfig reads all SELL_*/TAKE_PROFIT_* environment variables into a
// SellConfig, following the same get/getInt/getFloat/getBool helper pattern
// used by the rest of this package.
func loadSellConfig() SellConfig {
	var c SellConfig
	c.Enabled = getBool("SELL_SIGNAL_ENABLED", true)
	c.RequireClosedCandle = getBool("SELL_REQUIRE_CLOSED_CANDLE", true)
	c.RequireExecutedTradeConfirm = getBool("SELL_REQUIRE_EXECUTED_TRADE_CONFIRMATION", true)
	c.RequireOrderbookSync = getBool("SELL_REQUIRE_ORDERBOOK_SYNC", true)
	c.RequireFailedReclaim = getBool("SELL_REQUIRE_FAILED_RECLAIM", true)

	c.SetupScore = getFloat("SELL_SETUP_SCORE", 70)
	c.ConfirmScore = getFloat("SELL_CONFIRM_SCORE", 82)
	c.MinRuleScore = getFloat("SELL_MIN_RULE_SCORE", 68)
	c.MinModelProbability = getFloat("SELL_MIN_MODEL_PROBABILITY", 0.60)
	// MinDataQuality lowered 80 -> 75 to match the BUY engine's
	// MinDataQualityForSignal (signals/engine.go); an 80 floor hard-gated SELL
	// out of pairs whose data quality (75-79) was good enough for BUY.
	c.MinDataQuality = getFloat("SELL_MIN_DATA_QUALITY", 75)
	c.MaxSpoofScore = getFloat("SELL_MAX_SPOOF_SCORE", 60)
	c.MinTradeflowScore = getFloat("SELL_MIN_TRADEFLOW_SCORE", 70)
	// MinTimeframeAlignment lowered 60 -> 40. The old -0.60 weighted-trend
	// requirement was rarely met early in a selloff because high-timeframe EMA
	// crossovers lag; -0.40 plus the low-timeframe override in
	// bearishTrendConfirmed lets fresh breakdowns surface.
	c.MinTimeframeAlignment = getFloat("SELL_MIN_TIMEFRAME_ALIGNMENT", 40)

	c.PairCooldown = durationMinutes("SELL_PAIR_COOLDOWN_MINUTES", 30)
	c.MaxActiveGlobal = getInt("SELL_MAX_ACTIVE_GLOBAL", 15)
	c.MaxActivePerPair = getInt("SELL_MAX_ACTIVE_PER_PAIR", 1)
	c.MaxActivePerCluster = getInt("SELL_MAX_ACTIVE_PER_CLUSTER", 3)
	c.MaxNewPerMinute = getInt("SELL_MAX_NEW_PER_MINUTE", 5)

	c.MinTradeCount = getInt("SELL_MIN_TRADE_COUNT", 20)
	c.MinTradeNotionalUSDT = getFloat("SELL_MIN_TRADE_NOTIONAL_USDT", 10000)
	c.MinObservationSeconds = getInt("SELL_MIN_OBSERVATION_SECONDS", 60)

	c.TakeProfitEnabled = getBool("TAKE_PROFIT_ENABLED", true)
	c.TakeProfitSetupScore = getFloat("TAKE_PROFIT_SETUP_SCORE", 68)
	c.TakeProfitConfirmScore = getFloat("TAKE_PROFIT_CONFIRM_SCORE", 78)
	c.TakeProfitMinOverextension = getFloat("TAKE_PROFIT_MIN_OVEREXTENSION_SCORE", 65)
	c.TakeProfitMinExhaustion = getFloat("TAKE_PROFIT_MIN_EXHAUSTION_SCORE", 60)
	c.TakeProfitRequireCVDDivergence = getBool("TAKE_PROFIT_REQUIRE_CVD_DIVERGENCE", false)

	return c
}
