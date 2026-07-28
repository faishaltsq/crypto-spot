package sell

import (
	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/features/spoofing"
	"github.com/example/crypto-spot-signal/internal/features/structure"
	"github.com/example/crypto-spot-signal/internal/features/tradeflow"
	"github.com/example/crypto-spot-signal/internal/market"
)

// Builder assembles a sell.FeatureSnapshot for one pair per scan cycle. It
// owns the long-lived tradeflow/structure/spoofing services so their
// internal per-symbol state (wall memory, sample config) persists across
// cycles, exactly like features.Engine is a long-lived singleton for BUY.
type Builder struct {
	tradeFlow *tradeflow.Service
	walls     *spoofing.Tracker
}

func NewBuilder(sampleCfg tradeflow.SampleConfig) *Builder {
	return &Builder{
		tradeFlow: tradeflow.NewService(sampleCfg),
		walls:     spoofing.NewTracker(),
	}
}

// Build computes a full sell.FeatureSnapshot from a market snapshot and the
// BUY engine's already-computed domain.FeatureSnapshot for the same pair
// (features/engine.go runs first in the scanner loop; SELL reuses its
// direction-agnostic outputs rather than recomputing trend/liquidity/data
// quality from scratch).
func (b *Builder) Build(snapshot market.PairSnapshot, buyFeature domain.FeatureSnapshot) FeatureSnapshot {
	priceChangePct := 0.0
	if candles, ok := snapshot.Candles["1m"]; ok && len(candles) >= 2 {
		closed := structure.ClosedCandles(candles)
		if len(closed) >= 2 {
			prev := closed[len(closed)-2].Close
			last := closed[len(closed)-1].Close
			if prev > 0 {
				priceChangePct = (last - prev) / prev * 100
			}
		}
	}

	priceCtx := tradeflow.PriceContext{
		PriceChangePct:            priceChangePct,
		ExpectedDeclinePctPerUSDT: expectedMovePerUSDT(snapshot),
		ExpectedRisePctPerUSDT:    expectedMovePerUSDT(snapshot),
	}

	primaryFlow, multiFlow := b.tradeFlow.Analyze(snapshot.Symbol, snapshot.Trades, priceCtx)

	structureByTF := make(map[string]structure.BearishStructure, len(snapshot.Candles))
	for tf, candles := range snapshot.Candles {
		structureByTF[tf] = structure.Compute(snapshot.Symbol, tf, candles)
	}
	primaryStructure := structureByTF["15m"]
	if primaryStructure.CalculatedAt.IsZero() {
		primaryStructure = structureByTF["5m"]
	}

	walls := b.walls.Analyze(snapshot.Symbol, snapshot.TopBids, snapshot.TopAsks, snapshot.Book)

	f := FeatureSnapshot{
		Symbol:               snapshot.Symbol,
		Tier:                 snapshot.Tier,
		Price:                buyFeature.Price,
		TrendByTimeframe:     buyFeature.TrendByTimeframe,
		TrendAlignment:       buyFeature.TrendAlignment,
		LiquidityScore:       buyFeature.LiquidityScore,
		DataQualityScore:     buyFeature.DataQualityScore,
		DataQualityStatus:    buyFeature.DataQualityStatus,
		SpoofScoreRaw:        snapshot.Book.SpoofScore,
		SpoofStatus:          buyFeature.SpoofStatus,
		MarketRegime:         buyFeature.MarketRegime,
		VolatilityPercentile: buyFeature.VolatilityPercentile,
		CorrelationState:     buyFeature.CorrelationState,
		OrderbookSynced:      snapshot.Book.Synced,
		MissingFeatures:      buyFeature.MissingFeatures,
		TradeFlow:            primaryFlow,
		TradeFlowByWindow:    multiFlow,
		Structure:            primaryStructure,
		StructureByTimeframe: structureByTF,
		Walls:                walls,
		CalculatedAt:         buyFeature.CalculatedAt,
	}
	f.SellRuleScore = RuleScore(f)
	return f
}

// expectedMovePerUSDT calibrates absorption scoring to each pair's own
// liquidity: a pair with deeper book depth needs proportionally more sell
// volume to move price by the same percentage, so we scale the expected
// move inversely to total book depth rather than using one constant across
// all pairs of wildly different liquidity.
func expectedMovePerUSDT(snapshot market.PairSnapshot) float64 {
	depth := snapshot.Book.BidDepthQuote + snapshot.Book.AskDepthQuote
	if depth <= 0 {
		return 0
	}
	// Calibration: a trade equal to 1% of visible depth is assumed to move
	// price by roughly 0.05% in a non-absorbed market.
	return 0.05 / (depth * 0.01)
}
