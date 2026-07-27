package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/market"
	"github.com/example/crypto-spot-signal/internal/market/universe"
)

type HistoryFetcher struct {
	client     *http.Client
	store      *market.Store
	restURL    string
	timeframes []string
}

func NewHistoryFetcher(store *market.Store, restURL string, timeframes []string) *HistoryFetcher {
	return &HistoryFetcher{
		client:     &http.Client{Timeout: 10 * time.Second},
		store:      store,
		restURL:    restURL,
		timeframes: timeframes,
	}
}

// Backfill runs in the background and populates historical candles for the given pairs
func (h *HistoryFetcher) Backfill(ctx context.Context, pairs []universe.RankedPair) {
	log.Printf("history: starting backfill for %d pairs across %d timeframes", len(pairs), len(h.timeframes))

	// Rate limiting: 10 concurrent workers
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup

	for _, pair := range pairs {
		for _, tf := range h.timeframes {
			wg.Add(1)
			go func(symbol, timeframe string) {
				defer wg.Done()

				// Acquire semaphore
				select {
				case sem <- struct{}{}:
				case <-ctx.Done():
					return
				}
				defer func() { <-sem }()

				// Add a tiny delay to spread requests
				time.Sleep(50 * time.Millisecond)

				candles, err := h.fetchCandles(ctx, symbol, timeframe, 300)
				if err != nil {
					// Don't spam logs for every failure, just log occasionally or on timeout
					return
				}

				for _, candle := range candles {
					h.store.ApplyCandle(candle)
				}
			}(pair.Symbol, tf)
		}
	}

	go func() {
		wg.Wait()
		log.Println("history: backfill completed")
	}()
}

func (h *HistoryFetcher) fetchCandles(ctx context.Context, symbol, timeframe string, limit int) ([]domain.Candle, error) {
	url := fmt.Sprintf("%s/spot/candlesticks?currency_pair=%s&interval=%s&limit=%d", h.restURL, symbol, timeframe, limit)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var rawData [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return nil, err
	}

	var candles []domain.Candle
	for _, row := range rawData {
		if len(row) < 7 {
			continue
		}

		// Parse row [timestamp, quote_vol, close, high, low, open, base_vol, closed?]
		tsStr, _ := row[0].(string)
		quoteVolStr, _ := row[1].(string)
		closeStr, _ := row[2].(string)
		highStr, _ := row[3].(string)
		lowStr, _ := row[4].(string)
		openStr, _ := row[5].(string)
		baseVolStr, _ := row[6].(string)

		tsSec, _ := strconv.ParseInt(tsStr, 10, 64)
		openTime := time.Unix(tsSec, 0).UTC()
		
		open, _ := strconv.ParseFloat(openStr, 64)
		high, _ := strconv.ParseFloat(highStr, 64)
		low, _ := strconv.ParseFloat(lowStr, 64)
		closePrice, _ := strconv.ParseFloat(closeStr, 64)
		baseVol, _ := strconv.ParseFloat(baseVolStr, 64)
		quoteVol, _ := strconv.ParseFloat(quoteVolStr, 64)

		isClosed := false
		if len(row) >= 8 {
			if closedStr, ok := row[7].(string); ok && closedStr == "true" {
				isClosed = true
			} else if closedBool, ok := row[7].(bool); ok {
				isClosed = closedBool
			}
		}

		candles = append(candles, domain.Candle{
			Symbol:      symbol,
			Timeframe:   timeframe,
			OpenTime:    openTime,
			Open:        open,
			High:        high,
			Low:         low,
			Close:       closePrice,
			BaseVolume:  baseVol,
			QuoteVolume: quoteVol,
			Closed:      isClosed,
		})
	}

	return candles, nil
}
