package recorder

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RetentionManager handles pruning old market events.
// In TimescaleDB, this should ideally be handled by add_retention_policy(),
// but this struct provides an application-level fallback or management interface.
type RetentionManager struct {
	pool       *pgxpool.Pool
	retention  time.Duration
}

// NewRetentionManager creates a new retention manager.
func NewRetentionManager(pool *pgxpool.Pool, retention time.Duration) *RetentionManager {
	return &RetentionManager{
		pool:      pool,
		retention: retention,
	}
}

// Run starts a background loop that prunes old events.
func (rm *RetentionManager) Run(ctx context.Context) {
	// If TimescaleDB data retention is configured, this might just log status.
	// For now, we perform a manual DELETE query if needed, running hourly.
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rm.prune(ctx)
		}
	}
}

func (rm *RetentionManager) prune(ctx context.Context) {
	cutoff := time.Now().Add(-rm.retention)
	
	// WARNING: Manual delete on hypertables is slow. 
	// Production should use: SELECT add_retention_policy('market_events', INTERVAL '7 days');
	// This is a basic fallback.
	tag, err := rm.pool.Exec(ctx, "DELETE FROM market_events WHERE exchange_timestamp < $1", cutoff)
	if err != nil {
		log.Printf("[recorder] retention prune failed: %v", err)
		return
	}
	
	if tag.RowsAffected() > 0 {
		log.Printf("[recorder] pruned %d old events (cutoff: %v)", tag.RowsAffected(), cutoff)
	}
}
