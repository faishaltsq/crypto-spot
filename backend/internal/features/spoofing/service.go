package spoofing

import (
	"strings"
	"sync"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
)

// wallMemory remembers the last detected wall on each side of the book for
// one symbol, so the next scan cycle can tell whether it held or failed.
type wallMemory struct {
	bidPrice, bidQuote float64
	askPrice, askQuote float64
}

// Tracker maintains per-symbol wall memory across scan cycles. It must be
// long-lived (one instance shared across the scanner loop), matching the
// pattern used by market.OrderBook's own removalQuote decay state.
type Tracker struct {
	mu      sync.Mutex
	symbols map[string]*wallMemory
}

func NewTracker() *Tracker {
	return &Tracker{symbols: make(map[string]*wallMemory)}
}

// Analyze computes a WallAnalysis for one symbol given its current top
// bid/ask levels and existing domain.BookMetrics spoof score. It compares
// against the wall remembered from the previous call for this symbol to
// determine bid/ask wall failure, then updates memory for the next cycle.
func (t *Tracker) Analyze(symbol string, topBids, topAsks []domain.Level, book domain.BookMetrics) WallAnalysis {
	symbol = strings.ToUpper(symbol)
	t.mu.Lock()
	defer t.mu.Unlock()

	prev, ok := t.symbols[symbol]
	if !ok {
		prev = &wallMemory{}
		t.symbols[symbol] = prev
	}

	result := WallAnalysis{
		Symbol:       symbol,
		SpoofScore:   book.SpoofScore,
		CalculatedAt: time.Now(),
	}

	bidDetected, bidPrice, bidQuote := DetectWall(topBids)
	result.BidWallDetected = bidDetected
	result.BidWallPrice = bidPrice
	result.BidWallQuote = bidQuote

	askDetected, askPrice, askQuote := DetectWall(topAsks)
	result.AskWallDetected = askDetected
	result.AskWallPrice = askPrice
	result.AskWallQuote = askQuote

	if prev.bidQuote > 0 {
		currentAtPrevPrice := quoteAtPrice(topBids, prev.bidPrice)
		failed, confidence := WallFailure(prev.bidQuote, currentAtPrevPrice)
		result.BidWallFailed = failed
		result.BidWallFailureConfidence = confidence
	}
	if prev.askQuote > 0 {
		currentAtPrevPrice := quoteAtPrice(topAsks, prev.askPrice)
		failed, confidence := WallFailure(prev.askQuote, currentAtPrevPrice)
		result.AskWallFailed = failed
		result.AskWallFailureConfidence = confidence
	}

	if bidDetected {
		prev.bidPrice, prev.bidQuote = bidPrice, bidQuote
	}
	if askDetected {
		prev.askPrice, prev.askQuote = askPrice, askQuote
	}

	return result
}

func quoteAtPrice(levels []domain.Level, price float64) float64 {
	if price <= 0 {
		return 0
	}
	for _, lvl := range levels {
		if lvl.Price == price {
			return lvl.Price * lvl.Amount
		}
	}
	return 0
}
