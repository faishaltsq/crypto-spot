package execution_simulation

import (
	"context"
	"log"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/market"
)

// Storage interface for reading signals/market data and writing simulation results.
type Storage interface {
	SaveSimulation(ctx context.Context, result Result) error
}

// Simulator manages execution simulations for newly created signals.
type Simulator struct {
	storage     Storage
	marketStore *market.Store
	cfg         FeeConfig
	notionals   []float64
}

// NewSimulator creates a new execution simulator.
func NewSimulator(storage Storage, marketStore *market.Store, cfg FeeConfig, notionals []float64) *Simulator {
	if len(notionals) == 0 {
		notionals = []float64{50, 100, 250, 500, 1000}
	}
	return &Simulator{
		storage:     storage,
		marketStore: marketStore,
		cfg:         cfg,
		notionals:   notionals,
	}
}

// Simulate runs fee, slippage, and capacity simulations for a signal.
func (s *Simulator) Simulate(ctx context.Context, signal *domain.Signal) {
	snapshot, ok := s.marketStore.Snapshot(signal.Symbol, 0)
	if !ok {
		log.Printf("[simulation] missing market data for %s", signal.Symbol)
		return
	}

	result := Result{
		SignalID:           signal.ID,
		Symbol:             signal.Symbol,
		SimulatedAt:        time.Now(),
		BaseEntryPrice:     signal.EntryPrice,
		SlippageByNotional: make(map[float64]SlippageEstimate),
	}

	totalSlippage := 0.0
	validEstimates := 0

	for _, notional := range s.notionals {
		est := CalculateSlippage(notional, snapshot)
		result.SlippageByNotional[notional] = est
		if est.FullyFilled {
			totalSlippage += est.SlippageBPS
			validEstimates++
		}
	}

	if validEstimates > 0 {
		result.AvgSlippageBPS = totalSlippage / float64(validEstimates)
	}

	// Assuming average notional for the primary fee reporting
	avgNotional := s.notionals[len(s.notionals)/2]
	result.Fees = CalculateFee(avgNotional, s.cfg)

	result.Capacity = CalculateCapacity(snapshot)
	result.TotalCostBPS = result.Fees.TotalFeeBPS + result.AvgSlippageBPS

	if err := s.storage.SaveSimulation(ctx, result); err != nil {
		log.Printf("[simulation] failed to save result for %s: %v", signal.ID, err)
	}
}
