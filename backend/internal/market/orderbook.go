package market

import (
	"errors"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
)

var ErrSequenceGap = errors.New("order book sequence gap")

type OrderBook struct {
	mu           sync.RWMutex
	bids         map[float64]float64
	asks         map[float64]float64
	lastUpdateID int64
	synced       bool
	removalQuote float64
	lastDecay    time.Time
	updatedAt    time.Time
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		bids:      make(map[float64]float64),
		asks:      make(map[float64]float64),
		lastDecay: time.Now(),
	}
}

func (b *OrderBook) IsSynced() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.synced
}

func (b *OrderBook) MarkUnsynced() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.synced = false
}

func (b *OrderBook) ApplySnapshot(id int64, bids, asks []domain.Level) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.bids = make(map[float64]float64, len(bids))
	b.asks = make(map[float64]float64, len(asks))
	for _, level := range bids {
		if level.Price > 0 && level.Amount > 0 {
			b.bids[level.Price] = level.Amount
		}
	}
	for _, level := range asks {
		if level.Price > 0 && level.Amount > 0 {
			b.asks[level.Price] = level.Amount
		}
	}
	b.lastUpdateID = id
	b.synced = true
	b.updatedAt = time.Now()
}

func (b *OrderBook) ApplyDelta(firstID, lastID int64, bids, asks []domain.Level, ts time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.synced {
		return ErrSequenceGap
	}
	if lastID <= b.lastUpdateID {
		return nil
	}
	expected := b.lastUpdateID + 1
	if firstID > expected || lastID < expected {
		b.synced = false
		return ErrSequenceGap
	}

	b.decayRemovalLocked(ts)
	for _, level := range bids {
		b.applyLevelLocked(b.bids, level)
	}
	for _, level := range asks {
		b.applyLevelLocked(b.asks, level)
	}
	b.lastUpdateID = lastID
	b.updatedAt = ts
	return nil
}

func (b *OrderBook) applyLevelLocked(side map[float64]float64, level domain.Level) {
	if level.Price <= 0 {
		return
	}
	previous := side[level.Price]
	if level.Amount <= 0 {
		if previous > 0 {
			b.removalQuote += previous * level.Price
		}
		delete(side, level.Price)
		return
	}
	if previous > level.Amount {
		b.removalQuote += (previous - level.Amount) * level.Price
	}
	side[level.Price] = level.Amount
}

func (b *OrderBook) decayRemovalLocked(now time.Time) {
	if b.lastDecay.IsZero() {
		b.lastDecay = now
		return
	}
	elapsed := now.Sub(b.lastDecay).Seconds()
	if elapsed <= 0 {
		return
	}
	// Half-life is roughly 60 seconds.
	b.removalQuote *= math.Pow(0.5, elapsed/60.0)
	b.lastDecay = now
}

func (b *OrderBook) Metrics(depthPercent, recentTradeQuote float64) domain.BookMetrics {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	b.decayRemovalLocked(now)

	bestBid := maxKey(b.bids)
	bestAsk := minKey(b.asks)
	mid := 0.0
	spread := 0.0
	if bestBid > 0 && bestAsk > 0 && bestAsk >= bestBid {
		mid = (bestBid + bestAsk) / 2
		spread = ((bestAsk - bestBid) / mid) * 10000
	}

	bidDepth, askDepth := 0.0, 0.0
	if mid > 0 {
		distance := depthPercent / 100
		bidFloor := mid * (1 - distance)
		askCeiling := mid * (1 + distance)
		for price, amount := range b.bids {
			if price >= bidFloor {
				bidDepth += price * amount
			}
		}
		for price, amount := range b.asks {
			if price <= askCeiling {
				askDepth += price * amount
			}
		}
	}

	imbalance := 0.0
	totalDepth := bidDepth + askDepth
	if totalDepth > 0 {
		imbalance = (bidDepth - askDepth) / totalDepth
	}

	removalRatio := b.removalQuote / math.Max(recentTradeQuote, 1)
	spoof := clamp(removalRatio*18, 0, 100)
	if totalDepth < 10000 {
		spoof = clamp(spoof+10, 0, 100)
	}

	return domain.BookMetrics{
		Synced:        b.synced,
		LastUpdateID:  b.lastUpdateID,
		BestBid:       bestBid,
		BestAsk:       bestAsk,
		MidPrice:      mid,
		SpreadBPS:     spread,
		BidDepthQuote: bidDepth,
		AskDepthQuote: askDepth,
		Imbalance:     imbalance,
		SpoofScore:    spoof,
		RemovalQuote:  b.removalQuote,
		UpdatedAt:     b.updatedAt,
	}
}

func (b *OrderBook) Top(levels int) (bids, asks []domain.Level) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	bidPrices := make([]float64, 0, len(b.bids))
	askPrices := make([]float64, 0, len(b.asks))
	for price := range b.bids {
		bidPrices = append(bidPrices, price)
	}
	for price := range b.asks {
		askPrices = append(askPrices, price)
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(bidPrices)))
	sort.Float64s(askPrices)
	if len(bidPrices) > levels {
		bidPrices = bidPrices[:levels]
	}
	if len(askPrices) > levels {
		askPrices = askPrices[:levels]
	}
	for _, price := range bidPrices {
		bids = append(bids, domain.Level{Price: price, Amount: b.bids[price]})
	}
	for _, price := range askPrices {
		asks = append(asks, domain.Level{Price: price, Amount: b.asks[price]})
	}
	return bids, asks
}

func maxKey(values map[float64]float64) float64 {
	max := 0.0
	for key := range values {
		if key > max {
			max = key
		}
	}
	return max
}

func minKey(values map[float64]float64) float64 {
	min := 0.0
	for key := range values {
		if min == 0 || key < min {
			min = key
		}
	}
	return min
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
