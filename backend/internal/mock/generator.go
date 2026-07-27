package mock

import (
	"context"
	"hash/fnv"
	"math"
	"math/rand"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/market"
)

type Generator struct {
	store      *market.Store
	symbols    []string
	timeframes []string
	prices     map[string]float64
	random     *rand.Rand
	sequence   int64
}

func New(store *market.Store, symbols, timeframes []string) *Generator {
	prices := make(map[string]float64, len(symbols))
	for _, symbol := range symbols {
		prices[symbol] = seedPrice(symbol)
	}
	return &Generator{
		store:      store,
		symbols:    symbols,
		timeframes: timeframes,
		prices:     prices,
		random:     rand.New(rand.NewSource(42)),
		sequence:   1,
	}
}

func (g *Generator) Run(ctx context.Context) {
	g.seedHistory()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			g.tick(now)
		}
	}
}

func (g *Generator) seedHistory() {
	now := time.Now().UTC()
	for _, symbol := range g.symbols {
		base := g.prices[symbol]
		for _, timeframe := range g.timeframes {
			step := timeframeDuration(timeframe)
			if step <= 0 {
				continue
			}
			price := base * 0.96
			for index := 90; index >= 0; index-- {
				openTime := now.Add(-time.Duration(index) * step).Truncate(step)
				drift := 1 + 0.0008 + math.Sin(float64(index)/8)*0.0005
				open := price
				closePrice := open * drift
				high := math.Max(open, closePrice) * 1.0015
				low := math.Min(open, closePrice) * 0.9985
				g.store.ApplyCandle(domain.Candle{
					Symbol:      symbol,
					Timeframe:   timeframe,
					OpenTime:    openTime,
					Open:        open,
					High:        high,
					Low:         low,
					Close:       closePrice,
					BaseVolume:  0.5 + float64(index)*0.01,
					QuoteVolume: (0.5 + float64(index)*0.01) * closePrice,
					Closed:      index > 0,
				})
				price = closePrice
			}
		}
		g.publishBook(symbol, base, now)
	}
}

func (g *Generator) tick(now time.Time) {
	for _, symbol := range g.symbols {
		price := g.prices[symbol]
		move := g.random.NormFloat64()*0.0007 + 0.00008
		price *= 1 + move
		g.prices[symbol] = price

		side := "buy"
		if g.random.Float64() < 0.42 {
			side = "sell"
		}
		amount := 1 + g.random.Float64()*5
		g.store.ApplyTrade(domain.Trade{
			ID:        now.UnixNano(),
			Symbol:    symbol,
			Side:      side,
			Price:     price,
			Amount:    amount,
			Quote:     amount * price,
			Timestamp: now,
		})
		g.store.ApplyTicker(symbol, price, move*100, price*250000, now)
		g.store.ApplyBookTicker(symbol, price*0.9999, price*1.0001, now)
		g.publishBook(symbol, price, now)
		g.updateCandles(symbol, price, amount, now)
	}
}

func (g *Generator) publishBook(symbol string, price float64, now time.Time) {
	pair := g.store.Pair(symbol)
	if pair == nil {
		return
	}
	bids := make([]domain.Level, 0, 25)
	asks := make([]domain.Level, 0, 25)
	for index := 1; index <= 25; index++ {
		distance := float64(index) * 0.0001
		bidAmount := 3 + float64(25-index)*0.25 + g.random.Float64()
		askAmount := 2.5 + float64(25-index)*0.20 + g.random.Float64()
		bids = append(bids, domain.Level{Price: price * (1 - distance), Amount: bidAmount})
		asks = append(asks, domain.Level{Price: price * (1 + distance), Amount: askAmount})
	}
	g.sequence++
	pair.Book.ApplySnapshot(g.sequence, bids, asks)
	_ = now
}

func (g *Generator) updateCandles(symbol string, price, amount float64, now time.Time) {
	for _, timeframe := range g.timeframes {
		step := timeframeDuration(timeframe)
		if step <= 0 {
			continue
		}
		openTime := now.UTC().Truncate(step)
		g.store.ApplyCandle(domain.Candle{
			Symbol:      symbol,
			Timeframe:   timeframe,
			OpenTime:    openTime,
			Open:        price,
			High:        price * 1.0005,
			Low:         price * 0.9995,
			Close:       price,
			BaseVolume:  amount,
			QuoteVolume: amount * price,
			Closed:      false,
		})
	}
}

func timeframeDuration(value string) time.Duration {
	switch value {
	case "10s":
		return 10 * time.Second
	case "1m":
		return time.Minute
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "30m":
		return 30 * time.Minute
	case "1h":
		return time.Hour
	case "4h":
		return 4 * time.Hour
	case "8h":
		return 8 * time.Hour
	case "1d":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	default:
		return 0
	}
}

func seedPrice(symbol string) float64 {
	known := map[string]float64{
		"BTC_USDT":  65000,
		"ETH_USDT":  3500,
		"SOL_USDT":  150,
		"XRP_USDT":  0.6,
		"DOGE_USDT": 0.14,
	}
	if value, ok := known[symbol]; ok {
		return value
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(symbol))
	return 1 + float64(hasher.Sum32()%5000)/10
}
