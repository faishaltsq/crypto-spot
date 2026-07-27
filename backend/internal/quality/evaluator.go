package quality

import (
	"time"

	"github.com/example/crypto-spot-signal/internal/market"
)

// Evaluator runs quality checks against a market pair snapshot and produces a QualityReport.
type Evaluator struct {
	cfg QualityConfig
}

// NewEvaluator creates a new quality evaluator with the given configuration.
func NewEvaluator(cfg QualityConfig) *Evaluator {
	return &Evaluator{cfg: cfg}
}

// Evaluate runs all quality rules against the given pair snapshot and returns a report.
func (e *Evaluator) Evaluate(snapshot market.PairSnapshot, health PairHealthInput) QualityReport {
	now := time.Now()
	health.Now = now

	score, reasons, ruleResults := ComputeScore(health, e.cfg)
	status := statusFor(score, reasons)

	return QualityReport{
		Symbol:      snapshot.Symbol,
		Score:       score,
		Status:      status,
		Reasons:     reasons,
		RuleResults: ruleResults,
		// A non-VALID report never produces a signal, even if a weighted score
		// remains above the threshold after a non-critical rule penalty.
		SignalAllowed: status == StatusValid && score >= e.cfg.MinSignalScore,
		EvaluatedAt:   now,
		Freshness: FreshnessMetrics{
			Trade:  health.LastTradeTimestamp,
			Ticker: health.LastTickerTimestamp,
			Book:   health.LastBookTimestamp,
			Candle: health.LastCandleTimestamp,
		},
		Persistence: PersistenceMetrics{
			RedisLatencyMs: health.RedisLatencyMs,
			DBBacklogSize:  health.DBWriteBacklogSize,
		},
		Pipeline: PipelineMetrics{
			QueueUtilization: health.QueueUtilization,
		},
	}
}

func statusFor(score float64, reasons []ReasonCode) QualityStatus {
	for _, reason := range reasons {
		if reason == ReasonOrderbookUnsynced || reason == ReasonOrderbookResync {
			return StatusUnsynced
		}
	}
	for _, reason := range reasons {
		if reason == ReasonInsufficientData || reason == ReasonIncompleteFeatures {
			return StatusIncomplete
		}
	}
	for _, reason := range reasons {
		if reason == ReasonTradeStreamStale || reason == ReasonTickerStreamStale || reason == ReasonCandleGap || reason == ReasonReceiveTimestampLag {
			return StatusStale
		}
	}
	return ScoreToStatus(score)
}

// BuildHealthInput constructs a PairHealthInput from a market.PairSnapshot.
// Additional runtime metrics (queue utilization, redis latency, db backlog, reconnect time)
// must be set by the caller after this returns.
func BuildHealthInput(snapshot market.PairSnapshot) PairHealthInput {
	input := PairHealthInput{
		Symbol:            snapshot.Symbol,
		BookSynced:        snapshot.Book.Synced,
		BookLastUpdateID:  snapshot.Book.LastUpdateID,
		BookBestBid:       snapshot.Book.BestBid,
		BookBestAsk:       snapshot.Book.BestAsk,
		BookMidPrice:      snapshot.Book.MidPrice,
		BookSpreadBPS:     snapshot.Book.SpreadBPS,
		BookBidDepthQuote: snapshot.Book.BidDepthQuote,
		BookAskDepthQuote: snapshot.Book.AskDepthQuote,
		LastPrice:         snapshot.LastPrice,
		LastBookTimestamp: snapshot.Book.UpdatedAt,
	}

	// Last trade timestamp: use the most recent trade
	if len(snapshot.Trades) > 0 {
		input.LastTradeTimestamp = snapshot.Trades[len(snapshot.Trades)-1].Timestamp
		// Count trades in last minute
		cutoff := time.Now().Add(-time.Minute)
		for i := len(snapshot.Trades) - 1; i >= 0; i-- {
			if snapshot.Trades[i].Timestamp.Before(cutoff) {
				break
			}
			input.TradeCount1m++
		}
	}

	// Ticker timestamp (approximated from LastMarketUpdate)
	input.LastTickerTimestamp = snapshot.LastMarketUpdate

	// Candle data
	if candles1m, ok := snapshot.Candles["1m"]; ok {
		input.Candle1mCount = len(candles1m)
		if len(candles1m) > 0 {
			last := candles1m[len(candles1m)-1]
			input.LastCandleTimestamp = last.OpenTime
		}
	}
	input.CandleTimeframes = len(snapshot.Candles)

	return input
}
