package execution_simulation

import (
	"context"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/market"
)

type Storage interface {
	SaveEntrySimulations(context.Context, []Simulation) error
	SimulationsForSignal(context.Context, string) ([]Simulation, error)
	UpdateExitSimulations(context.Context, []Simulation) error
}

type Simulator struct { storage Storage; market *market.Store; cfg Config }

func NewSimulator(storage Storage, marketStore *market.Store, cfg Config) *Simulator {
	if len(cfg.Notionals) == 0 { cfg.Notionals = []float64{50, 100, 250, 500, 1000} }
	if cfg.FeeBPS <= 0 { cfg.FeeBPS = 10 }
	return &Simulator{storage: storage, market: marketStore, cfg: cfg}
}

func (s *Simulator) SimulateEntry(ctx context.Context, signal *domain.Signal) error {
	if !s.cfg.Enabled { return nil }
	snapshot, ok := s.market.Snapshot(signal.Symbol, 0.5)
	rows := make([]Simulation, 0, len(s.cfg.Notionals))
	for _, notional := range s.cfg.Notionals { rows = append(rows, s.entry(signal, snapshot, ok, notional)) }
	return s.storage.SaveEntrySimulations(ctx, rows)
}

func (s *Simulator) entry(signal *domain.Signal, snapshot market.PairSnapshot, available bool, notional float64) Simulation {
	now := time.Now().UTC()
	row := Simulation{SignalID: signal.ID, Notional: notional, Status: StatusIncomplete, SimulatedAt: now}
	if notional <= 0 || signal.EntryPrice <= 0 || !available || !snapshot.Book.Synced || len(snapshot.TopAsks) == 0 { return row }
	row.ReferencePrice = ptr(signal.EntryPrice)
	filled := walkQuote(snapshot.TopAsks, notional)
	if filled.base == 0 { return row }
	entry := filled.quote / filled.base
	coverage := filled.quote / notional
	max := 0.0; for _, level := range snapshot.TopAsks { if level.Price > 0 && level.Amount > 0 { max += level.Price * level.Amount } }
	confidence := min(1, coverage) * min(1, max/(notional*2))
	row.EstimatedEntryPrice, row.FilledNotional, row.UnfilledNotional = ptr(entry), ptr(filled.quote), ptr(notional-filled.quote)
	row.MaximumSupportedNotional, row.DepthCoverage, row.LiquidityConfidence = ptr(max), ptr(coverage), ptr(confidence)
	row.BaseFilled = filled.base
	if s.cfg.IncludeEntrySlippage {
		value := maxFloat(0, (entry-signal.EntryPrice)*filled.base); bps := value / filled.quote * 10000
		row.EntrySlippage, row.EntrySlippageBPS = ptr(value), ptr(bps)
	}
	row.EntryFee = fee(filled.quote, s.cfg.FeeBPS, s.cfg.IncludeEntryFee)
	if filled.quote+0.000001 < notional {
		row.PartialFill = true; row.Status = StatusPartialFill
		if !s.cfg.AllowPartialFill { row.Status = StatusIncomplete }
	} else { row.Status = StatusComplete }
	return row
}

func (s *Simulator) EvaluateExit(ctx context.Context, signalID, symbol string) ([]Simulation, error) {
	rows, err := s.storage.SimulationsForSignal(ctx, signalID); if err != nil || len(rows) == 0 { return rows, err }
	snapshot, available := s.market.Snapshot(symbol, 0.5)
	for i := range rows { s.exit(&rows[i], snapshot, available) }
	return rows, s.storage.UpdateExitSimulations(ctx, rows)
}

func (s *Simulator) exit(row *Simulation, snapshot market.PairSnapshot, available bool) {
	row.SimulatedAt = time.Now().UTC()
	if row.Status == StatusIncomplete || row.BaseFilled <= 0 || row.ReferencePrice == nil || !available || !snapshot.Book.Synced || len(snapshot.TopBids) == 0 { row.Status = StatusIncomplete; return }
	filled := walkBase(snapshot.TopBids, row.BaseFilled)
	if filled.base == 0 { row.Status = StatusIncomplete; return }
	exit := filled.quote / filled.base
	referenceExit := snapshot.LastPrice
	if referenceExit <= 0 { referenceExit = snapshot.Book.MidPrice }
	if referenceExit <= 0 { row.Status = StatusIncomplete; return }
	row.EstimatedExitPrice = ptr(exit)
	if s.cfg.IncludeExitSlippage { value := maxFloat(0, (referenceExit-exit)*filled.base); bps := value/(referenceExit*filled.base)*10000; row.ExitSlippage, row.ExitSlippageBPS = ptr(value), ptr(bps) }
	row.ExitFee = fee(filled.quote, s.cfg.FeeBPS, s.cfg.IncludeExitFee)
	entryReferenceValue := *row.ReferencePrice * row.BaseFilled
	if entryReferenceValue > 0 {
		gross := (referenceExit*filled.base - entryReferenceValue) / entryReferenceValue
		net := gross - (deref(row.EntryFee)+deref(row.ExitFee)+deref(row.EntrySlippage)+deref(row.ExitSlippage))/entryReferenceValue
		row.GrossReturn, row.NetReturn = ptr(gross), ptr(net)
	}
	if filled.base+0.00000001 < row.BaseFilled { row.PartialFill = true; row.Status = StatusPartialFill } else if row.Status != StatusPartialFill { row.Status = StatusComplete }
}

func deref(value *float64) float64 { if value == nil { return 0 }; return *value }
func maxFloat(a, b float64) float64 { if a > b { return a }; return b }
