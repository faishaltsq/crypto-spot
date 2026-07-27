package domain

import "time"

// --- Reason Codes (deterministic, not AI text) ---

const (
	// Supporting evidence
	ReasonVolumeExpansion   = "VOLUME_EXPANSION"
	ReasonPositiveCVD       = "POSITIVE_CVD"
	ReasonNegativeCVD       = "NEGATIVE_CVD"
	ReasonBidAbsorption     = "BID_ABSORPTION"
	ReasonAskAbsorption     = "ASK_ABSORPTION"
	ReasonMultiTFAlignment  = "MULTI_TIMEFRAME_ALIGNMENT"
	ReasonLowSpoofRisk      = "LOW_SPOOF_RISK"
	ReasonAdequateLiquidity = "ADEQUATE_LIQUIDITY"

	// Risk / blocking reason codes
	ReasonMultiTFConflict   = "MULTI_TIMEFRAME_CONFLICT"
	ReasonHighSpoofRisk     = "HIGH_SPOOF_RISK"
	ReasonLowLiquidity      = "LOW_LIQUIDITY"
	ReasonWideSpread        = "WIDE_SPREAD"
	ReasonOverextendedPrice = "OVEREXTENDED_PRICE"
	ReasonOrderbookUnsynced = "ORDERBOOK_UNSYNCED"
	ReasonStaleTradeStream  = "STALE_TRADE_STREAM"
	ReasonMarketSellOff     = "MARKET_SELL_OFF"
	ReasonCorrelatedCluster = "CORRELATED_SIGNAL_CLUSTER"

	// Signal gate rejection codes (requested by user)
	ReasonInsufficientRuleScore = "INSUFFICIENT_RULE_SCORE"
	ReasonLowTrendAlignment     = "LOW_TREND_ALIGNMENT"
	ReasonMissingOrderbook      = "MISSING_ORDERBOOK"
	ReasonMissingMultiTimeframe = "MISSING_MULTI_TIMEFRAME"
	ReasonMissingAIReview       = "MISSING_AI_REVIEW"
	ReasonStaleMarketData       = "STALE_MARKET_DATA"
	ReasonLowDataQuality        = "LOW_DATA_QUALITY"
	ReasonSignalCooldown        = "SIGNAL_COOLDOWN"
	ReasonCorrelatedSuppressed  = "CORRELATED_SIGNAL_SUPPRESSED"

	// Data quality penalty reasons
	ReasonDataNotReady         = "DATA_NOT_READY"
	ReasonBookNotSynced        = "BOOK_NOT_SYNCED"
	ReasonLimitedCandleHistory = "LIMITED_CANDLE_HISTORY"
	ReasonLowRecentTradeCount  = "LOW_RECENT_TRADE_COUNT"
	ReasonMissingTimeframes    = "MISSING_TIMEFRAMES"
)

// --- Core Market Types ---

type Trade struct {
	ID        int64     `json:"id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Price     float64   `json:"price"`
	Amount    float64   `json:"amount"`
	Quote     float64   `json:"quote"`
	Timestamp time.Time `json:"timestamp"`
}

type Candle struct {
	Symbol      string    `json:"symbol"`
	Timeframe   string    `json:"timeframe"`
	OpenTime    time.Time `json:"openTime"`
	Open        float64   `json:"open"`
	High        float64   `json:"high"`
	Low         float64   `json:"low"`
	Close       float64   `json:"close"`
	BaseVolume  float64   `json:"baseVolume"`
	QuoteVolume float64   `json:"quoteVolume"`
	Closed      bool      `json:"closed"`
}

type Level struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
}

type BookMetrics struct {
	Synced        bool      `json:"synced"`
	LastUpdateID  int64     `json:"lastUpdateId"`
	BestBid       float64   `json:"bestBid"`
	BestAsk       float64   `json:"bestAsk"`
	MidPrice      float64   `json:"midPrice"`
	SpreadBPS     float64   `json:"spreadBps"`
	BidDepthQuote float64   `json:"bidDepthQuote"`
	AskDepthQuote float64   `json:"askDepthQuote"`
	Imbalance     float64   `json:"imbalance"`
	SpoofScore    float64   `json:"spoofScore"`
	RemovalQuote  float64   `json:"removalQuote"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type TradeWindow struct {
	BuyQuote   float64 `json:"buyQuote"`
	SellQuote  float64 `json:"sellQuote"`
	TotalQuote float64 `json:"totalQuote"`
	Count      int     `json:"count"`
	DeltaRatio float64 `json:"deltaRatio"`
}

// DataQualityStatus is the classified health status of the market data for a pair.
type DataQualityStatus string

const (
	DataQualityValid       DataQualityStatus = "VALID"
	DataQualityDegraded    DataQualityStatus = "DEGRADED"
	DataQualityStale       DataQualityStatus = "STALE"
	DataQualityBlocked     DataQualityStatus = "BLOCKED"
	DataQualityUnavailable DataQualityStatus = "UNAVAILABLE"
)

// SpoofStatus classifies the spoof risk level.
type SpoofStatus string

const (
	SpoofStatusLow    SpoofStatus = "LOW"
	SpoofStatusMedium SpoofStatus = "MEDIUM"
	SpoofStatusHigh   SpoofStatus = "HIGH"
)

// SpoofStatusFrom maps a numeric spoof score to a SpoofStatus string.
func SpoofStatusFrom(score float64) SpoofStatus {
	switch {
	case score > 60:
		return SpoofStatusHigh
	case score > 30:
		return SpoofStatusMedium
	default:
		return SpoofStatusLow
	}
}

// DataSource labels whether data originates from a live exchange or a mock feed.
type DataSource string

const (
	DataSourceGate DataSource = "GATE"
	DataSourceMock DataSource = "MOCK"
)

// FeatureSnapshot is the output of the feature engine for a single pair.
// All scores are 0-100. Null-able fields use -1 as sentinel for "UNAVAILABLE".
type FeatureSnapshot struct {
	Symbol               string             `json:"symbol"`
	Tier                 int                `json:"tier"`
	DataSource           DataSource         `json:"dataSource"` // GATE or MOCK
	Price                float64            `json:"price"`
	Change24hPercent     float64            `json:"change24hPercent"`
	QuoteVolume24h       float64            `json:"quoteVolume24h"`
	SpreadBPS            float64            `json:"spreadBps"`
	BidDepthQuote        float64            `json:"bidDepthQuote"`
	AskDepthQuote        float64            `json:"askDepthQuote"`
	OrderbookImbalance   float64            `json:"orderbookImbalance"`
	SpoofScore           float64            `json:"spoofScore"`
	SpoofStatus          SpoofStatus        `json:"spoofStatus"`
	BuyRatio1m           float64            `json:"buyRatio1m"`
	VolumeDeltaRatio1m   float64            `json:"volumeDeltaRatio1m"`
	RelativeVolume1m     float64            `json:"relativeVolume1m"`
	TrendByTimeframe     map[string]string  `json:"trendByTimeframe"`
	EMA9ByTimeframe      map[string]float64 `json:"ema9ByTimeframe"`
	EMA20ByTimeframe     map[string]float64 `json:"ema20ByTimeframe"`
	TrendAlignment       float64            `json:"trendAlignment"`
	MarketRegime         string             `json:"marketRegime"`
	VolatilityPercentile float64            `json:"volatilityPercentile"`
	CorrelationState     string             `json:"correlationState"`
	LiquidityScore       float64            `json:"liquidityScore"`
	VolumeScore          float64            `json:"volumeScore"`
	OrderFlowScore       float64            `json:"orderFlowScore"`
	TrendScore           float64            `json:"trendScore"`
	DataQualityScore     float64            `json:"dataQualityScore"`
	DataQualityStatus    DataQualityStatus  `json:"dataQualityStatus"`
	RuleScore            float64            `json:"ruleScore"`
	Status               string             `json:"status"`
	Reasons              []string           `json:"reasons"`
	RiskFlags            []string           `json:"riskFlags"`
	MissingFeatures      []string           `json:"missingFeatures"` // features not yet available
	BlockedReasons       []string           `json:"blockedReasons"`  // reasons this pair is blocked from signaling
	CalculatedAt         time.Time          `json:"calculatedAt"`
}

// IsDataReady returns true only if the feature snapshot was computed with valid, complete data.
func (f FeatureSnapshot) IsDataReady() bool {
	return f.DataQualityStatus == DataQualityValid || f.DataQualityStatus == DataQualityDegraded
}

type AIReview struct {
	Decision                 string   `json:"decision"`
	Confidence               float64  `json:"confidence"`
	Summary                  string   `json:"summary"`
	SupportingReasonCodes    []string `json:"supporting_reason_codes"`
	ContradictingReasonCodes []string `json:"contradicting_reason_codes"`
	RiskFlags                []string `json:"risk_flags"`
	Provider                 string   `json:"provider"`
	Model                    string   `json:"model"`
	LatencyMS                int      `json:"latency_ms"`
	Fallback                 bool     `json:"fallback"`
	FallbackReason           string   `json:"fallback_reason,omitempty"`
	ProviderErrorCode        string   `json:"provider_error_code,omitempty"`
	PromptVersion            string   `json:"prompt_version"`
	SchemaVersion            string   `json:"schema_version"`
}

// AIReviewRecord is audit metadata only. It never drives signal state.
type AIReviewRecord struct {
	SignalID   *string
	Pair       string
	Timeframe  string
	Review     AIReview
	ReviewedAt time.Time
}

// --- Signal Versioning ---

// SignalVersion records the exact version of each pipeline component
// that was active when a signal was generated.
type SignalVersion struct {
	SignalVersion      string `json:"signalVersion"`
	FeatureVersion     string `json:"featureVersion"`
	RuleVersion        string `json:"ruleVersion"`
	ModelVersion       string `json:"modelVersion"`
	LabelVersion       string `json:"labelVersion,omitempty"`
	DataSchemaVersion  string `json:"dataSchemaVersion"`
	AIPromptVersion    string `json:"aiPromptVersion,omitempty"`
	AIProvider         string `json:"aiProvider,omitempty"`
	QualityGateVersion string `json:"qualityGateVersion"`
}

// CurrentSignalVersion returns the current pipeline versions.
func CurrentSignalVersion() SignalVersion {
	return SignalVersion{
		SignalVersion:      "signal-v3",
		FeatureVersion:     "feature-v2",
		RuleVersion:        "rule-v2",
		ModelVersion:       "deterministic-v2",
		DataSchemaVersion:  "schema-v4",
		QualityGateVersion: "quality-v2",
	}
}

// --- Signal Evidence ---

// SignalEvidence captures the structured proof for why a signal was generated.
type SignalEvidence struct {
	SupportingEvidence    []string `json:"supportingEvidence"`
	ContradictingEvidence []string `json:"contradictingEvidence"`
	PassedRules           []string `json:"passedRules"`
	FailedRules           []string `json:"failedRules"`
	BlockedRules          []string `json:"blockedRules,omitempty"`
	ReasonCodes           []string `json:"reasonCodes"`
}

// --- Threshold Detail ---

// ThresholdDetail records the dynamic threshold calculation for a signal.
type ThresholdDetail struct {
	ThresholdVersion      string   `json:"thresholdVersion"`
	BaseThreshold         float64  `json:"baseThreshold"`
	TierAdjustment        float64  `json:"tierAdjustment"`
	RegimeAdjustment      float64  `json:"regimeAdjustment"`
	VolatilityAdjustment  float64  `json:"volatilityAdjustment"`
	SpoofAdjustment       float64  `json:"spoofAdjustment"`
	LiquidityAdjustment   float64  `json:"liquidityAdjustment"`
	CorrelationAdjustment float64  `json:"correlationAdjustment"`
	FinalThreshold        float64  `json:"finalThreshold"`
	ActualScore           float64  `json:"actualScore"`
	Passed                bool     `json:"passed"`
	BlockedByThreshold    bool     `json:"blockedByThreshold"`
	ThresholdReasonCodes  []string `json:"thresholdReasonCodes"`
	// New detailed fields for auditability
	TrendAlignmentPct float64           `json:"trendAlignmentPct"`
	DataQualityScore  float64           `json:"dataQualityScoreAtSignal"`
	DataQualityStatus DataQualityStatus `json:"dataQualityStatusAtSignal"`
	SpoofScore        float64           `json:"spoofScoreAtSignal"`
	SpoofStatus       SpoofStatus       `json:"spoofStatusAtSignal"`
}

// --- Signal ---

type Signal struct {
	ID                string            `json:"id"`
	Symbol            string            `json:"symbol"`
	Type              string            `json:"type"`
	Status            string            `json:"status"`
	PrimaryTimeframe  string            `json:"primaryTimeframe"`
	EntryPrice        float64           `json:"entryPrice"`
	Invalidation      float64           `json:"invalidationPrice"`
	Target1           float64           `json:"targetPrice1"`
	Target2           float64           `json:"targetPrice2"`
	RuleScore         float64           `json:"ruleScore"`
	AI                AIReview          `json:"ai"`
	Reasons           []string          `json:"reasons"`
	RiskFlags         []string          `json:"riskFlags"`
	MissingFeatures   []string          `json:"missingFeatures"`
	BlockedReasons    []string          `json:"blockedReasons"`
	Features          FeatureSnapshot   `json:"features"`
	Version           SignalVersion     `json:"version"`
	Evidence          SignalEvidence    `json:"evidence"`
	Threshold         ThresholdDetail   `json:"threshold"`
	DataQualityScore  float64           `json:"dataQualityScore"`
	DataQualityStatus DataQualityStatus `json:"dataQualityStatus"`
	DataSource        DataSource        `json:"dataSource"`
	CreatedAt         time.Time         `json:"createdAt"`
	ExpiresAt         time.Time         `json:"expiresAt"`
	Simulations       []PaperSimulation `json:"simulations,omitempty"`
}

type PaperSimulation struct {
	Notional         float64  `json:"notional"`
	EntryFee         *float64 `json:"entryFee,omitempty"`
	ExitFee          *float64 `json:"exitFee,omitempty"`
	EntrySlippage    *float64 `json:"entrySlippage,omitempty"`
	ExitSlippage     *float64 `json:"exitSlippage,omitempty"`
	EntrySlippageBPS *float64 `json:"entrySlippageBps,omitempty"`
	ExitSlippageBPS  *float64 `json:"exitSlippageBps,omitempty"`
	GrossReturn      *float64 `json:"grossReturn,omitempty"`
	NetReturn        *float64 `json:"netReturn,omitempty"`
	SimulationStatus string   `json:"simulationStatus"`
}

type WSMessage struct {
	Event     string      `json:"event"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

type OutcomeCandidate struct {
	ID           string
	Symbol       string
	EntryPrice   float64
	Target1      float64
	Invalidation float64
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

type PerformanceSummary struct {
	TotalSignals     int64   `json:"totalSignals"`
	EvaluatedSignals int64   `json:"evaluatedSignals"`
	TargetHits       int64   `json:"targetHits"`
	InvalidationHits int64   `json:"invalidationHits"`
	TargetHitRate    float64 `json:"targetHitRate"`
	AverageReturn5m  float64 `json:"averageReturn5m"`
	AverageReturn15m float64 `json:"averageReturn15m"`
	AverageReturn1h  float64 `json:"averageReturn1h"`
	AverageReturn4h  float64 `json:"averageReturn4h"`
	AverageMFE       float64 `json:"averageMfe"`
	AverageMAE       float64 `json:"averageMae"`
}
