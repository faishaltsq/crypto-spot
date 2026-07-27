package recorder

import (
	"errors"
	"sync"
)

// ErrBufferFull is returned when the event buffer is full and cannot accept more events.
var ErrBufferFull = errors.New("recorder buffer is full")

// RingBuffer provides a thread-safe, fixed-size circular buffer for market events.
// It uses a channel internally for non-blocking enqueue operations, dropping events if full
// to ensure the critical path (market data ingestion) is never blocked.
type RingBuffer struct {
	events chan MarketEvent
	
	// Metrics
	mu          sync.Mutex
	dropped     int64
	enqueued    int64
	dequeued    int64
}

// NewRingBuffer creates a new ring buffer with the specified capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		events: make(chan MarketEvent, capacity),
	}
}

// Enqueue adds an event to the buffer. If the buffer is full, it drops the event
// and returns ErrBufferFull. This is intentionally non-blocking.
func (b *RingBuffer) Enqueue(event MarketEvent) error {
	select {
	case b.events <- event:
		b.mu.Lock()
		b.enqueued++
		b.mu.Unlock()
		return nil
	default:
		b.mu.Lock()
		b.dropped++
		b.mu.Unlock()
		return ErrBufferFull
	}
}

// DequeueBatch removes up to maxItems from the buffer and returns them.
// If the buffer is empty, it returns an empty slice immediately.
func (b *RingBuffer) DequeueBatch(maxItems int) []MarketEvent {
	batch := make([]MarketEvent, 0, maxItems)
	
	for i := 0; i < maxItems; i++ {
		select {
		case event := <-b.events:
			batch = append(batch, event)
		default:
			// Buffer is empty, return what we have
			if len(batch) > 0 {
				b.mu.Lock()
				b.dequeued += int64(len(batch))
				b.mu.Unlock()
			}
			return batch
		}
	}
	
	b.mu.Lock()
	b.dequeued += int64(len(batch))
	b.mu.Unlock()
	return batch
}

// WaitAndDequeueBatch waits for at least one item, then takes up to maxItems.
// It blocks until an item is available or the context/channel is closed.
func (b *RingBuffer) WaitAndDequeueBatch(maxItems int) []MarketEvent {
	batch := make([]MarketEvent, 0, maxItems)
	
	// Block for the first item
	event, ok := <-b.events
	if !ok {
		return batch
	}
	batch = append(batch, event)

	// Try to get more items up to maxItems without blocking
	for i := 1; i < maxItems; i++ {
		select {
		case evt := <-b.events:
			batch = append(batch, evt)
		default:
			break // No more items ready right now
		}
	}
	
	b.mu.Lock()
	b.dequeued += int64(len(batch))
	b.mu.Unlock()
	return batch
}

// Close closes the underlying channel.
func (b *RingBuffer) Close() {
	close(b.events)
}

// Stats returns current buffer statistics.
func (b *RingBuffer) Stats() (enqueued, dequeued, dropped, currentSize int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.enqueued, b.dequeued, b.dropped, int64(len(b.events))
}
