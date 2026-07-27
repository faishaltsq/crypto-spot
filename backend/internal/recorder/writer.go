package recorder

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Writer handles database persistence for market events.
type Writer struct {
	pool *pgxpool.Pool
}

// NewWriter creates a new database writer for market events.
func NewWriter(pool *pgxpool.Pool) *Writer {
	return &Writer{pool: pool}
}

// SaveBatch persists a batch of market events using the fast COPY protocol (via CopyFrom).
func (w *Writer) SaveBatch(ctx context.Context, events []MarketEvent) error {
	if len(events) == 0 {
		return nil
	}

	rows := make([][]interface{}, 0, len(events))
	for _, e := range events {
		rows = append(rows, []interface{}{
			e.EventID,
			e.EventType,
			e.Exchange,
			e.Symbol,
			e.ExchangeTimestamp,
			e.ReceivedTimestamp,
			e.ProcessedTimestamp,
			e.ConnectionID,
			e.Sequence,
			e.Payload,
			e.SchemaVersion,
			e.Compressed,
		})
	}

	_, err := w.pool.CopyFrom(
		ctx,
		pgx.Identifier{"market_events"},
		[]string{
			"event_id", "event_type", "exchange", "symbol",
			"exchange_timestamp", "received_timestamp", "processed_timestamp",
			"connection_id", "sequence", "payload", "schema_version", "compressed",
		},
		pgx.CopyFromRows(rows),
	)

	return err
}
