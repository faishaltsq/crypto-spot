package recorder

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Service coordinates the ingestion and persistence of market events.
// It uses a non-blocking ring buffer to ingest events rapidly without slowing
// down the WebSocket readers, and asynchronously batches them to the database.
type Service struct {
	cfg    Config
	buffer *RingBuffer
	writer *Writer
	
	wg sync.WaitGroup
}

// NewService creates a new recorder service.
func NewService(cfg Config, pool *pgxpool.Pool) *Service {
	if cfg.MaxBufferItems == 0 {
		cfg.MaxBufferItems = 50000 // default 50k items
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 1000
	}
	if cfg.FlushIntervalMs == 0 {
		cfg.FlushIntervalMs = 1000 // 1 second
	}
	
	return &Service{
		cfg:    cfg,
		buffer: NewRingBuffer(cfg.MaxBufferItems),
		writer: NewWriter(pool),
	}
}

// Record queues an event for recording.
// This method is safe to call from hot paths (like WebSocket message handlers).
// If the buffer is full, the event is dropped and an error is logged.
func (s *Service) Record(event MarketEvent, rawPayload interface{}) {
	if !s.cfg.Enabled {
		return
	}

	// Serialization and compression happen inline before queuing.
	// This trades CPU time on the reader goroutine for less memory in the buffer.
	if err := Serialize(&event, rawPayload); err != nil {
		log.Printf("[recorder] serialization failed: %v", err)
		return
	}

	if err := s.buffer.Enqueue(event); err != nil {
		// Log sparingly to avoid log spam when buffer is full
		// (A real system might rate-limit this log)
		log.Printf("[recorder] drop event: %v", err)
	}
}

// Run starts the background batch writer loop.
func (s *Service) Run(ctx context.Context) {
	if !s.cfg.Enabled {
		log.Printf("[recorder] disabled")
		return
	}
	
	s.wg.Add(1)
	defer s.wg.Done()

	log.Printf("[recorder] starting with batch_size=%d, flush_ms=%d", s.cfg.BatchSize, s.cfg.FlushIntervalMs)
	
	flushTicker := time.NewTicker(time.Duration(s.cfg.FlushIntervalMs) * time.Millisecond)
	defer flushTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.flushRemaining(context.Background())
			return
		case <-flushTicker.C:
			s.flushBatch(ctx)
		}
	}
}

func (s *Service) flushBatch(ctx context.Context) {
	batch := s.buffer.DequeueBatch(s.cfg.BatchSize)
	if len(batch) == 0 {
		return
	}

	now := time.Now()
	for i := range batch {
		batch[i].ProcessedTimestamp = now
	}

	if err := s.writer.SaveBatch(ctx, batch); err != nil {
		log.Printf("[recorder] failed to save batch of %d events: %v", len(batch), err)
		// We drop the batch on error to avoid halting ingestion.
		// A more robust system might use a dead-letter queue or retry.
	}
}

func (s *Service) flushRemaining(ctx context.Context) {
	log.Printf("[recorder] flushing remaining events before shutdown...")
	s.buffer.Close()
	
	// Drain the buffer
	for {
		batch := s.buffer.DequeueBatch(s.cfg.BatchSize)
		if len(batch) == 0 {
			break
		}
		now := time.Now()
		for i := range batch {
			batch[i].ProcessedTimestamp = now
		}
		if err := s.writer.SaveBatch(ctx, batch); err != nil {
			log.Printf("[recorder] final flush failed: %v", err)
		}
	}
}

// Stats returns current performance statistics.
func (s *Service) Stats() (enqueued, dequeued, dropped, currentSize int64) {
	return s.buffer.Stats()
}
