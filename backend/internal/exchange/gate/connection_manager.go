package gate

import (
	"context"
	"sync"
	"time"

	"github.com/example/crypto-spot-signal/internal/config"
	"github.com/example/crypto-spot-signal/internal/market"
	"github.com/example/crypto-spot-signal/internal/market/universe"
	"github.com/example/crypto-spot-signal/internal/recorder"
)

type ConnectionManager struct {
	cfg            config.Config
	store          *market.Store
	recorder       *recorder.Service
	mu             sync.Mutex
	conns          []*Connection
	active         []universe.RankedPair
	historyFetcher *HistoryFetcher
}

func NewConnectionManager(cfg config.Config, store *market.Store, recorderSvc *recorder.Service) *ConnectionManager {
	return &ConnectionManager{
		cfg:            cfg,
		store:          store,
		recorder:       recorderSvc,
		historyFetcher: NewHistoryFetcher(store, cfg.GateRESTURL, cfg.GateTimeframes),
	}
}

func (cm *ConnectionManager) Start(ctx context.Context) {
	// The manager watches the universe active pairs and reconciles connections.
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			cm.Stop()
			return
		case <-ticker.C:
			// In a real implementation, this would get active pairs from the universe service
			// cm.Reconcile(ctx, activePairs)
		}
	}
}

func (cm *ConnectionManager) UpdatePairs(ctx context.Context, pairs []universe.RankedPair) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// If these are new pairs, start background history fetching
	if len(cm.active) == 0 && len(pairs) > 0 {
		go cm.historyFetcher.Backfill(ctx, pairs)
	}

	cm.active = pairs

	// Disconnect all and reconnect for simplicity in this implementation.
	// In production, we'd do a diff and use subscription_manager.go
	cm.stopAll()

	var batch []universe.RankedPair
	for _, p := range pairs {
		batch = append(batch, p)
		if len(batch) >= cm.cfg.GateWSMaxPairsPerConn {
			cm.startConnection(ctx, batch)
			batch = nil
		}
	}
	if len(batch) > 0 {
		cm.startConnection(ctx, batch)
	}
}

func (cm *ConnectionManager) startConnection(ctx context.Context, pairs []universe.RankedPair) {
	conn := NewConnection(cm.cfg, cm.store, pairs, cm.recorder)
	cm.conns = append(cm.conns, conn)
	go conn.Run(ctx)
}

func (cm *ConnectionManager) stopAll() {
	for _, conn := range cm.conns {
		conn.Stop()
	}
	cm.conns = nil
}

func (cm *ConnectionManager) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.stopAll()
}
