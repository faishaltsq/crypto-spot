package sell

import (
	"sync"
	"time"
)

// BurstGuard prevents SELL signal bursts across all pairs, mirroring
// signals/engine.go's burstGuard exactly (same fixed 3s hard window) but as
// an independent instance so BUY and SELL signal issuance rates are tracked
// and rate-limited separately.
type BurstGuard struct {
	mu           sync.Mutex
	lastSignalAt time.Time
	lastMinuteAt time.Time
	countThisMin int
}

// BurstWindow is the minimum time between SELL-family signals across all
// pairs (anti-burst), matching the BUY engine's BurstWindow constant.
const BurstWindow = 3 * time.Second

func (b *BurstGuard) Allow(maxPerMin int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	if !b.lastSignalAt.IsZero() && now.Sub(b.lastSignalAt) < BurstWindow {
		return false
	}
	if now.Sub(b.lastMinuteAt) >= time.Minute {
		b.lastMinuteAt = now
		b.countThisMin = 0
	}
	if b.countThisMin >= maxPerMin {
		return false
	}
	b.lastSignalAt = now
	b.countThisMin++
	return true
}

// Cooldown tracks the last-issued-at time per pair per SELL signal type, so
// SELL_CONFIRMED, TAKE_PROFIT_SUGGESTED, AVOID_ENTRY, and EXIT_WARNING each
// have independent cooldowns rather than sharing one bucket per symbol
// (a pair should be able to get an AVOID_ENTRY warning even while a prior
// TAKE_PROFIT_SUGGESTED for that same pair is still in cooldown).
type Cooldown struct {
	mu   sync.Mutex
	last map[string]time.Time // key: symbol+"|"+signalType
}

func NewCooldown() *Cooldown {
	return &Cooldown{last: make(map[string]time.Time)}
}

func (c *Cooldown) Allow(symbol, signalType string, window time.Duration) bool {
	key := symbol + "|" + signalType
	c.mu.Lock()
	defer c.mu.Unlock()
	last, ok := c.last[key]
	if ok && time.Since(last) < window {
		return false
	}
	c.last[key] = time.Now()
	return true
}
