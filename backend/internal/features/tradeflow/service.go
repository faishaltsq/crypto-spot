package tradeflow

import "github.com/example/crypto-spot-signal/internal/domain"

// Service is the public entry point the SELL engine uses to pull trade-flow
// evidence for a symbol. It wraps Aggregator so callers never construct
// SellFlowSnapshot fields by hand.
type Service struct {
	aggregator *Aggregator
}

func NewService(cfg SampleConfig) *Service {
	return &Service{aggregator: NewAggregator(cfg)}
}

// Analyze returns the primary (1m) window snapshot used for gating, plus the
// full multi-window breakdown used for the Trade Flow diagnostic tab.
func (s *Service) Analyze(symbol string, allTrades []domain.Trade, price PriceContext) (SellFlowSnapshot, MultiWindowSnapshot) {
	multi := s.aggregator.ComputeAllWindows(symbol, allTrades, price)
	primary := multi.Windows["1m"]
	return primary, multi
}
