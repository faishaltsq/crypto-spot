package sell

import (
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/features/spoofing"
	"github.com/example/crypto-spot-signal/internal/features/structure"
	"github.com/example/crypto-spot-signal/internal/features/tradeflow"
)

// FeatureSnapshot is the SELL-side counterpart to domain.FeatureSnapshot.
// It reuses the direction-agnostic fields already computed by the BUY
// feature engine (trend, data quality, liquidity, spoof score — see
// features/engine.go) rather than recomputing them, and adds the
// SELL-specific evidence: executed trade flow, bearish price structure,
// and orderbook wall analysis.
type FeatureSnapshot struct {
	Symbol string `json:"symbol"`
	Tier   int    `json:"tier"`
	Price  float64 `json:"price"`

	// Reused direction-agnostic evidence (see docs/audit: features/engine.go
	// computes these once per pair regardless of BUY/SELL direction).
	TrendByTimeframe  map[string]string `json:"trendByTimeframe"`
	TrendAlignment    float64           `json:"trendAlignment"` // negative = bearish-aligned
	LiquidityScore    float64           `json:"liquidityScore"`
	DataQualityScore  float64           `json:"dataQualityScore"`
	DataQualityStatus domain.DataQualityStatus `json:"dataQualityStatus"`
	SpoofScoreRaw     float64           `json:"spoofScoreRaw"`
	SpoofStatus       domain.SpoofStatus `json:"spoofStatus"`
	MarketRegime      string            `json:"marketRegime"`
	VolatilityPercentile float64        `json:"volatilityPercentile"`
	CorrelationState  string            `json:"correlationState"`
	OrderbookSynced   bool              `json:"orderbookSynced"`
	MissingFeatures   []string          `json:"missingFeatures"`

	// SELL-specific evidence.
	TradeFlow          tradeflow.SellFlowSnapshot            `json:"tradeFlow"`
	TradeFlowByWindow  tradeflow.MultiWindowSnapshot          `json:"tradeFlowByWindow"`
	Structure          structure.BearishStructure             `json:"structure"`
	StructureByTimeframe map[string]structure.BearishStructure `json:"structureByTimeframe"`
	Walls              spoofing.WallAnalysis                  `json:"walls"`

	SellRuleScore float64 `json:"sellRuleScore"`
	CalculatedAt  time.Time `json:"calculatedAt"`
}

// Result is what Engine.Evaluate returns: the emitted signal (nil if none),
// plus which of the four SELL signal types (if any) fired.
type Result struct {
	Signal *domain.Signal
	Type   string // one of the domain.SellSignal*/TakeProfit*/AvoidEntry*/ExitWarning* constants
}
