package execution_simulation

import (
	"context"
	"math"
	"testing"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/market"
)

type memoryStore struct { rows []Simulation }
func (m *memoryStore) SaveEntrySimulations(_ context.Context, rows []Simulation) error { m.rows = append(m.rows, rows...); return nil }
func (m *memoryStore) SimulationsForSignal(_ context.Context, _ string) ([]Simulation, error) { return m.rows, nil }
func (m *memoryStore) UpdateExitSimulations(_ context.Context, rows []Simulation) error { m.rows = rows; return nil }

func TestWalkQuoteUsesMultipleAskLevels(t *testing.T) {
	fill := walkQuote([]domain.Level{{Price: 10, Amount: 5}, {Price: 11, Amount: 10}}, 100)
	if math.Abs(fill.base-(50.0/10+50.0/11)) > 1e-9 || fill.quote != 100 || fill.levels != 2 { t.Fatalf("unexpected buy fill: %#v", fill) }
}

func TestWalkBaseUsesBidLevels(t *testing.T) {
	fill := walkBase([]domain.Level{{Price: 10, Amount: 5}, {Price: 9, Amount: 10}}, 10)
	if fill.quote != 95 || fill.base != 10 || fill.levels != 2 { t.Fatalf("unexpected sell fill: %#v", fill) }
}

func TestEntrySimulationCalculatesFeeSlippageAndPartialFill(t *testing.T) {
	store := &memoryStore{}
	marketStore := market.NewStore(); marketStore.EnsurePair("BTC_USDT", 1)
	pair := marketStore.Pair("BTC_USDT")
	pair.Book.ApplySnapshot(1, []domain.Level{{Price: 99, Amount: 10}}, []domain.Level{{Price: 101, Amount: 0.5}})
	simulator := NewSimulator(store, marketStore, Config{Enabled: true, Notionals: []float64{100}, FeeBPS: 10, IncludeEntryFee: true, IncludeEntrySlippage: true, AllowPartialFill: true})
	signal := &domain.Signal{ID: "00000000-0000-0000-0000-000000000001", Symbol: "BTC_USDT", EntryPrice: 100}
	if err := simulator.SimulateEntry(context.Background(), signal); err != nil { t.Fatal(err) }
	row := store.rows[0]
	if row.Status != StatusPartialFill || !row.PartialFill || row.EntryFee == nil || *row.EntryFee <= 0 || row.EntrySlippage == nil || *row.EntrySlippage <= 0 { t.Fatalf("unexpected entry simulation: %#v", row) }
}

func TestEntrySimulationMarksMissingBookIncomplete(t *testing.T) {
	store := &memoryStore{}; marketStore := market.NewStore(); marketStore.EnsurePair("BTC_USDT", 1)
	simulator := NewSimulator(store, marketStore, Config{Enabled: true, Notionals: []float64{50}})
	if err := simulator.SimulateEntry(context.Background(), &domain.Signal{ID: "00000000-0000-0000-0000-000000000002", Symbol: "BTC_USDT", EntryPrice: 100}); err != nil { t.Fatal(err) }
	if store.rows[0].Status != StatusIncomplete || store.rows[0].EntryFee != nil { t.Fatalf("missing book must remain incomplete: %#v", store.rows[0]) }
}

func TestEntrySimulationCreatesEachConfiguredNotional(t *testing.T) {
	store := &memoryStore{}; marketStore := market.NewStore(); marketStore.EnsurePair("BTC_USDT", 1)
	pair := marketStore.Pair("BTC_USDT")
	pair.Book.ApplySnapshot(1, []domain.Level{{Price: 99, Amount: 20}}, []domain.Level{{Price: 101, Amount: 20}})
	simulator := NewSimulator(store, marketStore, Config{Enabled: true, Notionals: []float64{50, 100, 250}})
	if err := simulator.SimulateEntry(context.Background(), &domain.Signal{ID: "00000000-0000-0000-0000-000000000004", Symbol: "BTC_USDT", EntryPrice: 100}); err != nil { t.Fatal(err) }
	if len(store.rows) != 3 { t.Fatalf("expected 3 notionals, got %d", len(store.rows)) }
}

func TestEntrySimulationRejectsZeroReferencePrice(t *testing.T) {
	store := &memoryStore{}; marketStore := market.NewStore(); marketStore.EnsurePair("BTC_USDT", 1)
	pair := marketStore.Pair("BTC_USDT")
	pair.Book.ApplySnapshot(1, []domain.Level{{Price: 99, Amount: 2}}, []domain.Level{{Price: 101, Amount: 2}})
	simulator := NewSimulator(store, marketStore, Config{Enabled: true, Notionals: []float64{50}})
	if err := simulator.SimulateEntry(context.Background(), &domain.Signal{ID: "00000000-0000-0000-0000-000000000005", Symbol: "BTC_USDT"}); err != nil { t.Fatal(err) }
	if store.rows[0].Status != StatusIncomplete { t.Fatalf("zero price must be incomplete: %#v", store.rows[0]) }
}

func TestFeeUsesConfiguredBasisPoints(t *testing.T) {
	result := fee(100, 10, true)
	if result == nil || *result != 0.1 { t.Fatalf("expected 0.1 USDT, got %#v", result) }
}

func TestExitSimulationProducesNegativeNetReturn(t *testing.T) {
	entry := 100.0; filled := 100.0
	store := &memoryStore{rows: []Simulation{{ID: "00000000-0000-0000-0000-000000000003", SignalID: "signal", Notional: 100, ReferencePrice: &entry, FilledNotional: &filled, BaseFilled: 1, Status: StatusComplete}}}
	marketStore := market.NewStore(); marketStore.EnsurePair("BTC_USDT", 1)
	pair := marketStore.Pair("BTC_USDT")
	pair.Book.ApplySnapshot(1, []domain.Level{{Price: 90, Amount: 2}}, []domain.Level{{Price: 101, Amount: 2}})
	simulator := NewSimulator(store, marketStore, Config{Enabled: true, FeeBPS: 10, IncludeExitFee: true, IncludeExitSlippage: true})
	rows, err := simulator.EvaluateExit(context.Background(), "signal", "BTC_USDT")
	if err != nil { t.Fatal(err) }
	if rows[0].GrossReturn == nil || rows[0].NetReturn == nil || *rows[0].GrossReturn >= 0 || *rows[0].NetReturn >= 0 { t.Fatalf("expected negative returns: %#v", rows[0]) }
}
