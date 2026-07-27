package market

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
)

const (
	maxTrades  = 25000
	maxCandles = 300
)

type PairState struct {
	mu               sync.RWMutex
	Symbol           string
	Tier             int
	LastPrice        float64
	BestBid          float64
	BestAsk          float64
	Change24hPercent float64
	QuoteVolume24h   float64
	LastMarketUpdate time.Time
	Trades           []domain.Trade
	Candles          map[string][]domain.Candle
	Book             *OrderBook
}

type Store struct {
	mu    sync.RWMutex
	pairs map[string]*PairState
}

type PairSnapshot struct {
	Symbol           string                     `json:"symbol"`
	Tier             int                        `json:"tier"`
	LastPrice        float64                    `json:"lastPrice"`
	BestBid          float64                    `json:"bestBid"`
	BestAsk          float64                    `json:"bestAsk"`
	Change24hPercent float64                    `json:"change24hPercent"`
	QuoteVolume24h   float64                    `json:"quoteVolume24h"`
	LastMarketUpdate time.Time                  `json:"lastMarketUpdate"`
	Trades           []domain.Trade             `json:"trades"`
	Candles          map[string][]domain.Candle `json:"candles"`
	Book             domain.BookMetrics         `json:"book"`
	TopBids          []domain.Level             `json:"topBids"`
	TopAsks          []domain.Level             `json:"topAsks"`
}

func NewStore() *Store {
	return &Store{pairs: make(map[string]*PairState)}
}

func (s *Store) EnsurePair(symbol string, tier int) {
	symbol = strings.ToUpper(symbol)
	s.mu.Lock()
	defer s.mu.Unlock()
	if state, ok := s.pairs[symbol]; ok {
		state.Tier = tier
	} else {
		s.pairs[symbol] = &PairState{
			Symbol:  symbol,
			Tier:    tier,
			Candles: make(map[string][]domain.Candle),
			Book:    NewOrderBook(),
		}
	}
}

func (s *Store) Pair(symbol string) *PairState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pairs[strings.ToUpper(symbol)]
}

func (s *Store) Symbols() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.pairs))
	for symbol := range s.pairs {
		out = append(out, symbol)
	}
	sort.Strings(out)
	return out
}

func (s *Store) ApplyTrade(trade domain.Trade) {
	pair := s.Pair(trade.Symbol)
	if pair == nil {
		return
	}
	pair.mu.Lock()
	defer pair.mu.Unlock()

	pair.LastPrice = trade.Price
	pair.LastMarketUpdate = trade.Timestamp
	pair.Trades = append(pair.Trades, trade)
	if len(pair.Trades) > maxTrades {
		pair.Trades = append([]domain.Trade(nil), pair.Trades[len(pair.Trades)-maxTrades:]...)
	}
	cutoff := time.Now().Add(-20 * time.Minute)
	first := 0
	for first < len(pair.Trades) && pair.Trades[first].Timestamp.Before(cutoff) {
		first++
	}
	if first > 0 {
		pair.Trades = append([]domain.Trade(nil), pair.Trades[first:]...)
	}
}

func (s *Store) ApplyTicker(symbol string, last, changePercent, quoteVolume float64, ts time.Time) {
	pair := s.Pair(symbol)
	if pair == nil {
		return
	}
	pair.mu.Lock()
	defer pair.mu.Unlock()
	if last > 0 {
		pair.LastPrice = last
	}
	pair.Change24hPercent = changePercent
	pair.QuoteVolume24h = quoteVolume
	pair.LastMarketUpdate = ts
}

func (s *Store) ApplyBookTicker(symbol string, bid, ask float64, ts time.Time) {
	pair := s.Pair(symbol)
	if pair == nil {
		return
	}
	pair.mu.Lock()
	defer pair.mu.Unlock()
	pair.BestBid = bid
	pair.BestAsk = ask
	pair.LastMarketUpdate = ts
}

func (s *Store) ApplyCandle(candle domain.Candle) {
	pair := s.Pair(candle.Symbol)
	if pair == nil {
		return
	}
	pair.mu.Lock()
	defer pair.mu.Unlock()

	candles := pair.Candles[candle.Timeframe]
	replaced := false
	for i := len(candles) - 1; i >= 0 && i >= len(candles)-3; i-- {
		if candles[i].OpenTime.Equal(candle.OpenTime) {
			candles[i] = candle
			replaced = true
			break
		}
	}
	if !replaced {
		candles = append(candles, candle)
	}
	sort.Slice(candles, func(i, j int) bool {
		return candles[i].OpenTime.Before(candles[j].OpenTime)
	})
	if len(candles) > maxCandles {
		candles = append([]domain.Candle(nil), candles[len(candles)-maxCandles:]...)
	}
	pair.Candles[candle.Timeframe] = candles
	if candle.Close > 0 {
		pair.LastPrice = candle.Close
	}
	pair.LastMarketUpdate = time.Now()
}

func (s *Store) Snapshot(symbol string, depthPercent float64) (PairSnapshot, bool) {
	pair := s.Pair(symbol)
	if pair == nil {
		return PairSnapshot{}, false
	}

	pair.mu.RLock()
	trades := append([]domain.Trade(nil), pair.Trades...)
	candles := make(map[string][]domain.Candle, len(pair.Candles))
	for timeframe, values := range pair.Candles {
		candles[timeframe] = append([]domain.Candle(nil), values...)
	}
	snapshot := PairSnapshot{
		Symbol:           pair.Symbol,
		Tier:             pair.Tier,
		LastPrice:        pair.LastPrice,
		BestBid:          pair.BestBid,
		BestAsk:          pair.BestAsk,
		Change24hPercent: pair.Change24hPercent,
		QuoteVolume24h:   pair.QuoteVolume24h,
		LastMarketUpdate: pair.LastMarketUpdate,
		Trades:           trades,
		Candles:          candles,
	}
	pair.mu.RUnlock()

	recentQuote := tradeWindow(trades, time.Now().Add(-time.Minute)).TotalQuote
	snapshot.Book = pair.Book.Metrics(depthPercent, recentQuote)
	snapshot.TopBids, snapshot.TopAsks = pair.Book.Top(100) // Increase top levels for UI selection
	if snapshot.BestBid == 0 {
		snapshot.BestBid = snapshot.Book.BestBid
	}
	if snapshot.BestAsk == 0 {
		snapshot.BestAsk = snapshot.Book.BestAsk
	}
	return snapshot, true
}

func (s *Store) Snapshots(depthPercent float64) []PairSnapshot {
	symbols := s.Symbols()
	out := make([]PairSnapshot, 0, len(symbols))
	for _, symbol := range symbols {
		if snapshot, ok := s.Snapshot(symbol, depthPercent); ok {
			out = append(out, snapshot)
		}
	}
	return out
}

func tradeWindow(trades []domain.Trade, cutoff time.Time) domain.TradeWindow {
	var window domain.TradeWindow
	for i := len(trades) - 1; i >= 0; i-- {
		trade := trades[i]
		if trade.Timestamp.Before(cutoff) {
			break
		}
		if trade.Side == "buy" {
			window.BuyQuote += trade.Quote
		} else {
			window.SellQuote += trade.Quote
		}
		window.Count++
	}
	window.TotalQuote = window.BuyQuote + window.SellQuote
	if window.TotalQuote > 0 {
		window.DeltaRatio = (window.BuyQuote - window.SellQuote) / window.TotalQuote
	}
	return window
}

func TradeWindow(trades []domain.Trade, duration time.Duration) domain.TradeWindow {
	return tradeWindow(trades, time.Now().Add(-duration))
}
