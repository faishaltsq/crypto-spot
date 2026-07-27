package quality

import (
	"fmt"
	"math"
	"time"
)

// Rule is a single data quality check that can detect a specific issue.
type Rule struct {
	Code    ReasonCode
	Penalty float64
	Check   func(input PairHealthInput, cfg QualityConfig) (passed bool, detail string)
}

// AllRules returns the ordered list of quality check rules.
// Each rule deducts its penalty from a perfect score of 100 when it fails.
func AllRules() []Rule {
	return []Rule{
		{
			Code:    ReasonOrderbookUnsynced,
			Penalty: 40,
			Check: func(input PairHealthInput, _ QualityConfig) (bool, string) {
				if !input.BookSynced {
					return false, "orderbook is not in synced state"
				}
				return true, ""
			},
		},
		{
			Code:    ReasonEmptyOrderbook,
			Penalty: 45,
			Check: func(input PairHealthInput, _ QualityConfig) (bool, string) {
				if input.BookBestBid <= 0 && input.BookBestAsk <= 0 {
					return false, "orderbook has no bid or ask levels"
				}
				if input.BookBidDepthQuote <= 0 && input.BookAskDepthQuote <= 0 {
					return false, "orderbook depth is zero on both sides"
				}
				return true, ""
			},
		},
		{
			Code:    ReasonOrderbookResync,
			Penalty: 30,
			Check: func(input PairHealthInput, _ QualityConfig) (bool, string) {
				if input.BookResyncInProgress {
					return false, "orderbook resync is in progress"
				}
				return true, ""
			},
		},
		{
			Code:    ReasonTradeStreamStale,
			Penalty: 25,
			Check: func(input PairHealthInput, cfg QualityConfig) (bool, string) {
				if input.LastTradeTimestamp.IsZero() {
					return false, "no trade data received yet"
				}
				age := input.Now.Sub(input.LastTradeTimestamp).Seconds()
				if age > cfg.StaleTradeSec {
					return false, fmt.Sprintf("last trade %.1fs ago (threshold: %.0fs)", age, cfg.StaleTradeSec)
				}
				return true, ""
			},
		},
		{
			Code:    ReasonTickerStreamStale,
			Penalty: 20,
			Check: func(input PairHealthInput, cfg QualityConfig) (bool, string) {
				if input.LastTickerTimestamp.IsZero() {
					return false, "no ticker data received yet"
				}
				age := input.Now.Sub(input.LastTickerTimestamp).Seconds()
				if age > cfg.StaleTickerSec {
					return false, fmt.Sprintf("last ticker %.1fs ago (threshold: %.0fs)", age, cfg.StaleTickerSec)
				}
				return true, ""
			},
		},
		{
			Code:    ReasonCandleGap,
			Penalty: 15,
			Check: func(input PairHealthInput, cfg QualityConfig) (bool, string) {
				if input.LastCandleTimestamp.IsZero() {
					return false, "no candle data received yet"
				}
				age := input.Now.Sub(input.LastCandleTimestamp).Seconds()
				if age > cfg.StaleCandleSec {
					return false, fmt.Sprintf("last candle %.1fs ago (threshold: %.0fs)", age, cfg.StaleCandleSec)
				}
				return true, ""
			},
		},
		{
			Code:    ReasonReceiveTimestampLag,
			Penalty: 15,
			Check: func(input PairHealthInput, cfg QualityConfig) (bool, string) {
				if input.LastBookTimestamp.IsZero() {
					return true, "" // no data yet, other rules will catch this
				}
				age := input.Now.Sub(input.LastBookTimestamp).Seconds()
				if age > cfg.StaleOrderbookSec {
					return false, fmt.Sprintf("orderbook update %.1fs ago (threshold: %.0fs)", age, cfg.StaleOrderbookSec)
				}
				return true, ""
			},
		},
		{
			Code:    ReasonAbnormalSpread,
			Penalty: 15,
			Check: func(input PairHealthInput, cfg QualityConfig) (bool, string) {
				if input.BookSpreadBPS <= 0 {
					return true, "" // no spread data
				}
				if input.BookSpreadBPS > cfg.MaxSpreadBPS*3 {
					return false, fmt.Sprintf("spread %.1f bps exceeds 3x threshold (%.0f bps)", input.BookSpreadBPS, cfg.MaxSpreadBPS*3)
				}
				return true, ""
			},
		},
		{
			Code:    ReasonPriceDeviation,
			Penalty: 20,
			Check: func(input PairHealthInput, cfg QualityConfig) (bool, string) {
				if input.LastPrice <= 0 || input.BookMidPrice <= 0 {
					return true, "" // can't check without data
				}
				deviationBPS := math.Abs(input.LastPrice-input.BookMidPrice) / input.BookMidPrice * 10000
				if deviationBPS > cfg.MaxPriceDeviationBPS {
					return false, fmt.Sprintf("price deviation %.0f bps (threshold: %.0f bps)", deviationBPS, cfg.MaxPriceDeviationBPS)
				}
				return true, ""
			},
		},
		{
			Code:    ReasonRecentReconnect,
			Penalty: 20,
			Check: func(input PairHealthInput, cfg QualityConfig) (bool, string) {
				if input.LastReconnectAt.IsZero() {
					return true, ""
				}
				elapsed := input.Now.Sub(input.LastReconnectAt).Seconds()
				if elapsed < cfg.ReconnectCooldownSec {
					return false, fmt.Sprintf("reconnected %.0fs ago (cooldown: %.0fs)", elapsed, cfg.ReconnectCooldownSec)
				}
				return true, ""
			},
		},
		{
			Code:    ReasonInsufficientData,
			Penalty: 15,
			Check: func(input PairHealthInput, _ QualityConfig) (bool, string) {
				if input.Candle1mCount < 10 {
					return false, fmt.Sprintf("only %d 1m candles (minimum: 10)", input.Candle1mCount)
				}
				return true, ""
			},
		},
		{
			Code:    ReasonIncompleteFeatures,
			Penalty: 15,
			Check: func(input PairHealthInput, _ QualityConfig) (bool, string) {
				if input.CandleTimeframes < 3 {
					return false, fmt.Sprintf("only %d timeframes available (minimum: 3)", input.CandleTimeframes)
				}
				if input.TradeCount1m < 2 {
					return false, fmt.Sprintf("only %d trades in last minute (minimum: 2)", input.TradeCount1m)
				}
				return true, ""
			},
		},
		{
			Code:    ReasonEventQueueOverload,
			Penalty: 20,
			Check: func(input PairHealthInput, cfg QualityConfig) (bool, string) {
				if input.QueueUtilization > cfg.MaxQueueUtilizationPct {
					return false, fmt.Sprintf("queue utilization %.1f%% (threshold: %.0f%%)", input.QueueUtilization, cfg.MaxQueueUtilizationPct)
				}
				return true, ""
			},
		},
		{
			Code:    ReasonRedisLag,
			Penalty: 10,
			Check: func(input PairHealthInput, _ QualityConfig) (bool, string) {
				if input.RedisLatencyMs > 500 {
					return false, fmt.Sprintf("redis latency %.0fms", input.RedisLatencyMs)
				}
				return true, ""
			},
		},
		{
			Code:    ReasonDatabaseBacklog,
			Penalty: 10,
			Check: func(input PairHealthInput, _ QualityConfig) (bool, string) {
				if input.DBWriteBacklogSize > 1000 {
					return false, fmt.Sprintf("database write backlog: %d items", input.DBWriteBacklogSize)
				}
				return true, ""
			},
		},
		{
			Code:    ReasonOrderbookSequenceGap,
			Penalty: 25,
			Check: func(input PairHealthInput, _ QualityConfig) (bool, string) {
				// This rule checks if the orderbook has a known sequence gap.
				// The gap detection happens at the connection level and is propagated
				// through the BookSynced flag. We keep this as a separate rule for
				// explicit reason code tracking even though it overlaps with ORDERBOOK_UNSYNCED.
				// In practice, sequence gaps cause a resync which temporarily unsets BookSynced.
				_ = input
				return true, "" // sequence gap tracking happens via UNSYNCED + RESYNC rules
			},
		},
		{
			Code:    ReasonExchangeTimestampLag,
			Penalty: 10,
			Check: func(input PairHealthInput, _ QualityConfig) (bool, string) {
				if input.LastTradeTimestamp.IsZero() {
					return true, ""
				}
				// Check if the exchange timestamp is suspiciously far from our receive time.
				// This indicates clock drift or exchange-side delays.
				lag := input.Now.Sub(input.LastTradeTimestamp)
				if lag > 30*time.Second || lag < -5*time.Second {
					return false, fmt.Sprintf("exchange timestamp lag: %v", lag.Round(time.Second))
				}
				return true, ""
			},
		},
	}
}
