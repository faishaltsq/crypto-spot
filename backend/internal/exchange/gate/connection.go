package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/example/crypto-spot-signal/internal/config"
	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/market"
	"github.com/example/crypto-spot-signal/internal/market/universe"
	"github.com/example/crypto-spot-signal/internal/recorder"
	"github.com/gorilla/websocket"
)

type Connection struct {
	cfg        config.Config
	store      *market.Store
	pairs      []universe.RankedPair
	recorder   *recorder.Service
	cancel     context.CancelFunc
	httpClient *http.Client
	
	snapshotMu       sync.Mutex
	snapshotInFlight map[string]bool
}

func NewConnection(cfg config.Config, store *market.Store, pairs []universe.RankedPair, rec *recorder.Service) *Connection {
	return &Connection{
		cfg:              cfg,
		store:            store,
		pairs:            pairs,
		recorder:         rec,
		httpClient:       &http.Client{Timeout: 12 * time.Second},
		snapshotInFlight: make(map[string]bool),
	}
}

func (c *Connection) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	defer cancel()

	backoff := c.cfg.GateWSReconnectMin
	for {
		if ctx.Err() != nil {
			return
		}
		if err := c.connectAndRead(ctx); err != nil && ctx.Err() == nil {
			log.Printf("gate websocket disconnected: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > c.cfg.GateWSReconnectMax {
			backoff = c.cfg.GateWSReconnectMax
		}
	}
}

func (c *Connection) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *Connection) connectAndRead(ctx context.Context) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.cfg.GateWSURL, nil)
	if err != nil {
		return fmt.Errorf("dial gate websocket: %w", err)
	}
	defer conn.Close()

	if err := c.subscribe(conn); err != nil {
		return err
	}

	conn.SetReadLimit(4 << 20)
	_ = conn.SetReadDeadline(time.Now().Add(c.cfg.GateWSStaleTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(c.cfg.GateWSStaleTimeout))
	})

	pingDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-pingDone:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
			}
		}
	}()
	defer close(pingDone)

	log.Printf("gate websocket connected for %d pairs", len(c.pairs))
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		_ = conn.SetReadDeadline(time.Now().Add(c.cfg.GateWSStaleTimeout))
		if err := c.handleMessage(ctx, raw); err != nil {
			log.Printf("gate message ignored: %v", err)
		}
	}
}

func (c *Connection) subscribe(conn *websocket.Conn) error {
	var requests []map[string]interface{}
	
	for _, p := range c.pairs {
		// Tier 1 and 2: spot.trades, spot.book_ticker, spot.candlesticks, spot.order_book_update
		// Tier 3: spot.tickers, spot.candlesticks, spot.trades (if capacity)
		
		requests = append(requests, map[string]interface{}{
			"time":    time.Now().Unix(),
			"channel": "spot.tickers",
			"event":   "subscribe",
			"payload": []string{p.Symbol},
		})
		
		// Simplify: we subscribe all to 1m, 5m, 15m, 1h, 4h, 1d to avoid excessive subscriptions
		for _, tf := range []string{"1m", "5m", "15m", "1h", "4h", "1d"} {
			requests = append(requests, map[string]interface{}{
				"time":    time.Now().Unix(),
				"channel": "spot.candlesticks",
				"event":   "subscribe",
				"payload": []string{tf, p.Symbol},
			})
		}
		
		if p.Tier <= 2 {
			requests = append(requests, map[string]interface{}{
				"time":    time.Now().Unix(),
				"channel": "spot.trades",
				"event":   "subscribe",
				"payload": []string{p.Symbol},
			})
			requests = append(requests, map[string]interface{}{
				"time":    time.Now().Unix(),
				"channel": "spot.book_ticker",
				"event":   "subscribe",
				"payload": []string{p.Symbol},
			})
			requests = append(requests, map[string]interface{}{
				"time":    time.Now().Unix(),
				"channel": "spot.order_book_update",
				"event":   "subscribe",
				"payload": []string{p.Symbol, "100ms"},
			})
		} else {
			// Tier 3: Basic public trades, no orderbook to save bandwidth
			requests = append(requests, map[string]interface{}{
				"time":    time.Now().Unix(),
				"channel": "spot.trades",
				"event":   "subscribe",
				"payload": []string{p.Symbol},
			})
		}
	}

	// Batch subscriptions
	batchSize := c.cfg.GateWSSubBatchSize
	for i := 0; i < len(requests); i += batchSize {
		end := i + batchSize
		if end > len(requests) {
			end = len(requests)
		}
		
		for _, req := range requests[i:end] {
			if err := conn.WriteJSON(req); err != nil {
				return fmt.Errorf("subscribe %v: %w", req, err)
			}
		}
		time.Sleep(c.cfg.GateWSSubBatchDelay)
	}
	
	return nil
}

// ... handleMessage and parse logic (same as previous client.go)

type envelope struct {
	Time    int64           `json:"time"`
	Channel string          `json:"channel"`
	Event   string          `json:"event"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
}

type tradeResult struct {
	ID           int64  `json:"id"`
	CreateTime   int64  `json:"create_time"`
	CreateTimeMS string `json:"create_time_ms"`
	CurrencyPair string `json:"currency_pair"`
	Side         string `json:"side"`
	Amount       string `json:"amount"`
	Price        string `json:"price"`
}

type bookTickerResult struct {
	T int64  `json:"t"`
	U int64  `json:"u"`
	S string `json:"s"`
	B string `json:"b"`
	Bf string `json:"B"`
	A string `json:"a"`
	Af string `json:"A"`
}

type tickerResult struct {
	CurrencyPair     string `json:"currency_pair"`
	Last             string `json:"last"`
	LowestAsk        string `json:"lowest_ask"`
	HighestBid       string `json:"highest_bid"`
	ChangePercentage string `json:"change_percentage"`
	BaseVolume       string `json:"base_volume"`
	QuoteVolume      string `json:"quote_volume"`
}

type candleResult struct {
	T string `json:"t"`
	V string `json:"v"`
	C string `json:"c"`
	H string `json:"h"`
	L string `json:"l"`
	O string `json:"o"`
	N string `json:"n"`
	A string `json:"a"`
	W bool   `json:"w"`
}

type orderBookUpdateResult struct {
	T    int64      `json:"t"`
	E    string     `json:"e"`
	E2   int64      `json:"E"`
	S    string     `json:"s"`
	U    int64      `json:"U"`
	Last int64      `json:"u"`
	Bids [][]string `json:"b"`
	Asks [][]string `json:"a"`
}

type restSnapshot struct {
	ID      int64      `json:"id"`
	Current int64      `json:"current"`
	Bids    [][]string `json:"bids"`
	Asks    [][]string `json:"asks"`
}

func (s restSnapshot) SequenceID() int64 {
	if s.ID > 0 {
		return s.ID
	}
	return s.Current
}

func (c *Connection) handleMessage(ctx context.Context, raw []byte) error {
	var message envelope
	if err := json.Unmarshal(raw, &message); err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}
	if len(message.Error) > 0 && string(message.Error) != "null" {
		return fmt.Errorf("gate error: %s", string(message.Error))
	}
	if message.Event != "update" {
		return nil
	}

	switch message.Channel {
	case "spot.trades":
		var result tradeResult
		if err := json.Unmarshal(message.Result, &result); err != nil {
			return err
		}
		price := parseFloat(result.Price)
		amount := parseFloat(result.Amount)
		ts := unixMilliString(result.CreateTimeMS, result.CreateTime)
		c.store.ApplyTrade(domain.Trade{
			ID:        result.ID,
			Symbol:    result.CurrencyPair,
			Side:      strings.ToLower(result.Side),
			Price:     price,
			Amount:    amount,
			Quote:     price * amount,
			Timestamp: ts,
		})
		if c.recorder != nil {
			c.recorder.Record(
				recorder.NewMarketEvent(recorder.EventTrade, "gate", result.CurrencyPair, ts, "ws", result.ID, message.Result),
				result,
			)
		}
	case "spot.book_ticker":
		var result bookTickerResult
		if err := json.Unmarshal(message.Result, &result); err != nil {
			return err
		}
		c.store.ApplyBookTicker(result.S, parseFloat(result.B), parseFloat(result.A), time.UnixMilli(result.T))
		if c.recorder != nil {
			c.recorder.Record(
				recorder.NewMarketEvent(recorder.EventTicker, "gate", result.S, time.UnixMilli(result.T), "ws", result.U, message.Result),
				result,
			)
		}
	case "spot.tickers":
		var result tickerResult
		if err := json.Unmarshal(message.Result, &result); err != nil {
			return err
		}
		now := time.Now()
		c.store.ApplyTicker(
			result.CurrencyPair,
			parseFloat(result.Last),
			parseFloat(result.ChangePercentage),
			parseFloat(result.QuoteVolume),
			now,
		)
		if c.recorder != nil {
			c.recorder.Record(
				recorder.NewMarketEvent(recorder.EventTicker, "gate", result.CurrencyPair, now, "ws", 0, message.Result),
				result,
			)
		}
	case "spot.candlesticks":
		var result candleResult
		if err := json.Unmarshal(message.Result, &result); err != nil {
			return err
		}
		parts := strings.SplitN(result.N, "_", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid candle name %q", result.N)
		}
		openUnix, _ := strconv.ParseInt(result.T, 10, 64)
		c.store.ApplyCandle(domain.Candle{
			Symbol:      parts[1],
			Timeframe:   parts[0],
			OpenTime:    time.Unix(openUnix, 0),
			Open:        parseFloat(result.O),
			High:        parseFloat(result.H),
			Low:         parseFloat(result.L),
			Close:       parseFloat(result.C),
			BaseVolume:  parseFloat(result.A),
			QuoteVolume: parseFloat(result.V),
			Closed:      result.W,
		})
		if c.recorder != nil {
			c.recorder.Record(
				recorder.NewMarketEvent(recorder.EventCandle, "gate", parts[1], time.Unix(openUnix, 0), "ws", 0, message.Result),
				result,
			)
		}
	case "spot.order_book_update":
		var result orderBookUpdateResult
		if err := json.Unmarshal(message.Result, &result); err != nil {
			return err
		}
		
		if c.recorder != nil {
			c.recorder.Record(
				recorder.NewMarketEvent(recorder.EventOrderbook, "gate", result.S, time.UnixMilli(result.T), "ws", result.U, message.Result),
				result,
			)
		}

		pair := c.store.Pair(result.S)
		if pair == nil {
			// Might happen if store hasn't initialized it yet, just skip
			return nil
		}
		if !pair.Book.IsSynced() {
			if err := c.ensureSnapshot(ctx, result.S); err != nil {
				return err
			}
		}
		err := pair.Book.ApplyDelta(
			result.U,
			result.Last,
			parseLevels(result.Bids),
			parseLevels(result.Asks),
			time.UnixMilli(result.T),
		)
		if err == market.ErrSequenceGap {
			pair.Book.MarkUnsynced()
			go func(symbol string) {
				if snapshotErr := c.ensureSnapshot(context.Background(), symbol); snapshotErr != nil {
					log.Printf("snapshot resync %s failed: %v", symbol, snapshotErr)
				}
			}(result.S)
			return nil
		}
		return err
	}
	return nil
}

func (c *Connection) ensureSnapshot(ctx context.Context, symbol string) error {
	c.snapshotMu.Lock()
	if c.snapshotInFlight[symbol] {
		c.snapshotMu.Unlock()
		return nil
	}
	c.snapshotInFlight[symbol] = true
	c.snapshotMu.Unlock()
	defer func() {
		c.snapshotMu.Lock()
		delete(c.snapshotInFlight, symbol)
		c.snapshotMu.Unlock()
	}()

	pair := c.store.Pair(symbol)
	if pair == nil || pair.Book.IsSynced() {
		return nil
	}

	endpoint, _ := url.Parse(strings.TrimRight(c.cfg.GateRESTURL, "/") + "/spot/order_book")
	query := endpoint.Query()
	query.Set("currency_pair", symbol)
	query.Set("limit", "100")
	query.Set("with_id", "true")
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("fetch order book snapshot: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("snapshot returned status %d", response.StatusCode)
	}

	var snapshot restSnapshot
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		return fmt.Errorf("decode snapshot: %w", err)
	}
	sequenceID := snapshot.SequenceID()
	if sequenceID <= 0 {
		return fmt.Errorf("snapshot did not include an order book sequence id")
	}
	pair.Book.ApplySnapshot(sequenceID, parseLevels(snapshot.Bids), parseLevels(snapshot.Asks))
	return nil
}

func parseLevels(raw [][]string) []domain.Level {
	levels := make([]domain.Level, 0, len(raw))
	for _, item := range raw {
		if len(item) < 2 {
			continue
		}
		price := parseFloat(item[0])
		amount := parseFloat(item[1])
		if math.IsNaN(price) || math.IsNaN(amount) {
			continue
		}
		levels = append(levels, domain.Level{Price: price, Amount: amount})
	}
	return levels
}

func parseFloat(value string) float64 {
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return number
}

func unixMilliString(value string, fallbackSeconds int64) time.Time {
	if value == "" {
		return time.Unix(fallbackSeconds, 0)
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return time.Unix(fallbackSeconds, 0)
	}
	return time.UnixMilli(int64(number))
}
