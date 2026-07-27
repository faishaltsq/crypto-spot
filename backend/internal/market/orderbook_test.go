package market

import (
	"testing"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
)

func TestOrderBookSnapshotAndDelta(t *testing.T) {
	book := NewOrderBook()
	book.ApplySnapshot(
		100,
		[]domain.Level{{Price: 99, Amount: 2}, {Price: 98, Amount: 3}},
		[]domain.Level{{Price: 101, Amount: 2}, {Price: 102, Amount: 3}},
	)

	if !book.IsSynced() {
		t.Fatal("snapshot must mark the book as synced")
	}

	err := book.ApplyDelta(
		101,
		102,
		[]domain.Level{{Price: 99, Amount: 1.5}, {Price: 100, Amount: 1}},
		[]domain.Level{{Price: 101, Amount: 0}},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("valid delta failed: %v", err)
	}

	metrics := book.Metrics(2, 1000)
	if metrics.LastUpdateID != 102 {
		t.Fatalf("unexpected sequence id: %d", metrics.LastUpdateID)
	}
	if metrics.BestBid != 100 {
		t.Fatalf("unexpected best bid: %f", metrics.BestBid)
	}
	if metrics.BestAsk != 102 {
		t.Fatalf("unexpected best ask: %f", metrics.BestAsk)
	}
	if metrics.RemovalQuote <= 0 {
		t.Fatal("removed liquidity should be recorded")
	}
}

func TestOrderBookRejectsSequenceGap(t *testing.T) {
	book := NewOrderBook()
	book.ApplySnapshot(50, nil, nil)

	err := book.ApplyDelta(53, 54, nil, nil, time.Now())
	if err != ErrSequenceGap {
		t.Fatalf("expected sequence gap, got %v", err)
	}
	if book.IsSynced() {
		t.Fatal("book must become unsynced after a sequence gap")
	}
}
