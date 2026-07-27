package recorder

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType categorizes the type of market data event.
type EventType string

const (
	EventTrade          EventType = "TRADE"
	EventOrderbook      EventType = "ORDERBOOK"
	EventTicker         EventType = "TICKER"
	EventCandle         EventType = "CANDLE"
	EventConnectionDrop EventType = "CONNECTION_DROP"
	EventConnectionSync EventType = "CONNECTION_SYNC"
)

// MarketEvent represents a single market data update as received from the exchange.
// It is designed to be stored as a time-series event for auditing and replay.
type MarketEvent struct {
	EventID            string          `json:"eventId"`
	EventType          EventType       `json:"eventType"`
	Exchange           string          `json:"exchange"`
	Symbol             string          `json:"symbol"`
	ExchangeTimestamp  time.Time       `json:"exchangeTimestamp"` // When the event occurred at the exchange
	ReceivedTimestamp  time.Time       `json:"receivedTimestamp"` // When our system received it
	ProcessedTimestamp time.Time       `json:"processedTimestamp"` // When our system finished processing it
	ConnectionID       string          `json:"connectionId"`
	Sequence           int64           `json:"sequence"`      // Sequence number from the exchange, if available
	Payload            json.RawMessage `json:"payload"`       // The raw or normalized JSON payload
	SchemaVersion      int             `json:"schemaVersion"` // Version of the payload schema
	Compressed         bool            `json:"compressed"`    // Whether the payload is compressed (e.g. gzip)
}

// NewMarketEvent creates a new MarketEvent with auto-generated ID and receive time.
func NewMarketEvent(
	eventType EventType,
	exchange, symbol string,
	exchangeTime time.Time,
	connectionID string,
	sequence int64,
	payload json.RawMessage,
) MarketEvent {
	return MarketEvent{
		EventID:           uuid.NewString(),
		EventType:         eventType,
		Exchange:          exchange,
		Symbol:            symbol,
		ExchangeTimestamp: exchangeTime,
		ReceivedTimestamp: time.Now(),
		ConnectionID:      connectionID,
		Sequence:          sequence,
		Payload:           payload,
		SchemaVersion:     1,
		Compressed:        false, // Determined later during serialization
	}
}

// Config holds configuration for the market data recorder.
type Config struct {
	Enabled         bool
	BatchSize       int
	FlushIntervalMs int
	MaxBufferItems  int
}
