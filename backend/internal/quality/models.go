package quality

import "time"

// QualityStatus represents the overall data health status for a pair.
type QualityStatus string

const (
	StatusValid      QualityStatus = "VALID"
	StatusDegraded   QualityStatus = "DEGRADED"
	StatusStale      QualityStatus = "STALE"
	StatusUnsynced   QualityStatus = "UNSYNCED"
	StatusIncomplete QualityStatus = "INCOMPLETE"
	StatusBlocked    QualityStatus = "BLOCKED"
)

// ReasonCode is a deterministic identifier for a quality issue.
type ReasonCode string

const (
	ReasonOrderbookUnsynced      ReasonCode = "ORDERBOOK_UNSYNCED"
	ReasonOrderbookSequenceGap   ReasonCode = "ORDERBOOK_SEQUENCE_GAP"
	ReasonTradeStreamStale       ReasonCode = "TRADE_STREAM_STALE"
	ReasonTickerStreamStale      ReasonCode = "TICKER_STREAM_STALE"
	ReasonCandleGap              ReasonCode = "CANDLE_GAP"
	ReasonExchangeTimestampLag   ReasonCode = "EXCHANGE_TIMESTAMP_LAG"
	ReasonReceiveTimestampLag    ReasonCode = "RECEIVE_TIMESTAMP_LAG"
	ReasonAbnormalSpread         ReasonCode = "ABNORMAL_SPREAD"
	ReasonPriceDeviation         ReasonCode = "PRICE_DEVIATION"
	ReasonEmptyOrderbook         ReasonCode = "EMPTY_ORDERBOOK"
	ReasonRedisLag               ReasonCode = "REDIS_LAG"
	ReasonDatabaseBacklog        ReasonCode = "DATABASE_WRITE_BACKLOG"
	ReasonRecentReconnect        ReasonCode = "RECENT_RECONNECT"
	ReasonInsufficientData       ReasonCode = "INSUFFICIENT_DATA"
	ReasonIncompleteFeatures     ReasonCode = "INCOMPLETE_FEATURES"
	ReasonEventQueueOverload     ReasonCode = "EVENT_QUEUE_OVERLOAD"
	ReasonOrderbookResync        ReasonCode = "ORDERBOOK_RESYNC"
)

// QualityReport is the result of evaluating a single pair's data health.
type QualityReport struct {
	Symbol         string        `json:"symbol"`
	Score          float64       `json:"score"`
	Status         QualityStatus `json:"status"`
	Reasons        []ReasonCode  `json:"reasons"`
	RuleResults    []RuleResult  `json:"ruleResults"`
	SignalAllowed  bool          `json:"signalAllowed"`
	EvaluatedAt    time.Time     `json:"evaluatedAt"`
	Freshness      FreshnessMetrics `json:"freshness"`
	Sequence       SequenceMetrics  `json:"sequence"`
	Pipeline       PipelineMetrics  `json:"pipeline"`
	Persistence    PersistenceMetrics `json:"persistence"`
	LastValidAt    *time.Time    `json:"lastValidAt,omitempty"`
	LastDegradedAt *time.Time    `json:"lastDegradedAt,omitempty"`
	LastBlockedAt  *time.Time    `json:"lastBlockedAt,omitempty"`
}

type FreshnessMetrics struct {
	Trade  time.Time `json:"trade"`
	Ticker time.Time `json:"ticker"`
	Book   time.Time `json:"book"`
	Candle time.Time `json:"candle"`
}

type SequenceMetrics struct {
	ResyncCount    int `json:"resyncCount"`
	ReconnectCount int `json:"reconnectCount"`
}

type PipelineMetrics struct {
	QueueUtilization float64 `json:"queueUtilization"`
	// DroppedEvents int // Add later if available
}

type PersistenceMetrics struct {
	RedisLatencyMs float64 `json:"redisLatencyMs"`
	DBBacklogSize  int     `json:"dbBacklogSize"`
}

// RuleResult is the outcome of a single quality check rule.
type RuleResult struct {
	Rule    string     `json:"rule"`
	Code    ReasonCode `json:"code"`
	Passed  bool       `json:"passed"`
	Penalty float64    `json:"penalty"`
	Score   float64    `json:"score"`
	Reason  string     `json:"reason,omitempty"`
}

// QualityConfig holds thresholds for the data quality gate.
type QualityConfig struct {
	MinSignalScore          float64
	BlockSignalScore        float64
	StaleTradeSec           float64
	StaleTickerSec          float64
	StaleOrderbookSec       float64
	StaleCandleSec          float64
	ReconnectCooldownSec    float64
	MaxPriceDeviationBPS    float64
	MaxQueueUtilizationPct  float64
	MaxSpreadBPS            float64
}

// PairHealthInput is the data needed to evaluate a pair's quality.
// It is extracted from market.PairSnapshot and additional runtime state.
type PairHealthInput struct {
	Symbol              string
	BookSynced          bool
	BookLastUpdateID    int64
	BookBestBid         float64
	BookBestAsk         float64
	BookMidPrice        float64
	BookSpreadBPS       float64
	BookBidDepthQuote   float64
	BookAskDepthQuote   float64
	LastTradeTimestamp   time.Time
	LastTickerTimestamp  time.Time
	LastBookTimestamp    time.Time
	LastCandleTimestamp  time.Time
	LastReconnectAt     time.Time
	LastPrice           float64
	Candle1mCount       int
	CandleTimeframes    int
	TradeCount1m        int
	QueueUtilization    float64 // 0-100
	RedisLatencyMs      float64
	DBWriteBacklogSize  int
	BookResyncInProgress bool
	Now                 time.Time
}
