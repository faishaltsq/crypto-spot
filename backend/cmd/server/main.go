package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/example/crypto-spot-signal/internal/ai"
	"github.com/example/crypto-spot-signal/internal/config"
	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/example/crypto-spot-signal/internal/exchange/gate"
	"github.com/example/crypto-spot-signal/internal/execution_simulation"
	"github.com/example/crypto-spot-signal/internal/features"
	"github.com/example/crypto-spot-signal/internal/httpapi"
	"github.com/example/crypto-spot-signal/internal/market"
	"github.com/example/crypto-spot-signal/internal/market/universe"
	"github.com/example/crypto-spot-signal/internal/mock"
	"github.com/example/crypto-spot-signal/internal/outcome"
	"github.com/example/crypto-spot-signal/internal/quality"
	"github.com/example/crypto-spot-signal/internal/realtime"
	"github.com/example/crypto-spot-signal/internal/recorder"
	runtimestate "github.com/example/crypto-spot-signal/internal/runtime"
	"github.com/example/crypto-spot-signal/internal/signals"
	"github.com/example/crypto-spot-signal/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	repo := mustConnectDatabase(ctx, cfg.DatabaseURL)
	defer repo.Close()

	var cache *storage.Cache
	cacheCandidate := storage.NewCache(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err := cacheCandidate.Ping(ctx); err != nil {
		log.Printf("redis unavailable, continuing without cache: %v", err)
		_ = cacheCandidate.Close()
	} else {
		cache = cacheCandidate
		defer cache.Close()
	}

	marketStore := market.NewStore()
	state := runtimestate.NewState()
	hub := realtime.NewHub()
	featureEngine := features.New(features.Config{
		MaxSpreadBPS:  cfg.MaxSpreadBPS,
		MinDepthQuote: cfg.MinDepthQuote,
	})
	aiClient := ai.New(cfg.AIEnabled, cfg.AIServiceURL, cfg.AITimeout, cfg.SignalMinScore)

	// Data quality gate
	qualityCfg := quality.QualityConfig{
		MinSignalScore:         cfg.DataQualityMinSignalScore,
		BlockSignalScore:       cfg.DataQualityBlockSignalScore,
		StaleTradeSec:          cfg.DataStaleTradeSec,
		StaleTickerSec:         cfg.DataStaleTickerSec,
		StaleOrderbookSec:      cfg.DataStaleOrderbookSec,
		StaleCandleSec:         cfg.DataStaleCandleSec,
		ReconnectCooldownSec:   cfg.DataReconnectCooldownSec,
		MaxPriceDeviationBPS:   cfg.DataMaxPriceDeviationBPS,
		MaxQueueUtilizationPct: cfg.DataMaxQueueUtilizationPct,
		MaxSpreadBPS:           cfg.MaxSpreadBPS,
	}
	qualitySvc := quality.NewService(qualityCfg)
	qualityMetrics := quality.NewMetrics()
	qualityRepo := quality.NewRepository(repo.Pool())

	signalEngine := signals.New(cfg.SignalMinScore, cfg.SignalPairCooldown, aiClient, qualitySvc, qualityMetrics)

	// Market Data Recorder
	recorderCfg := recorder.Config{
		Enabled:         cfg.MarketRecorderEnabled,
		BatchSize:       cfg.MarketRecorderBatchSize,
		FlushIntervalMs: cfg.MarketRecorderFlushIntMs,
		MaxBufferItems:  cfg.MarketRecorderMaxBufItems,
	}
	marketRecorder := recorder.NewService(recorderCfg, repo.Pool())
	go marketRecorder.Run(ctx)

	// Outcome and Simulation Services
	simulator := execution_simulation.NewSimulator(repo, marketStore, execution_simulation.Config{
		Enabled: cfg.PaperSimulationEnabled, Notionals: cfg.PaperNotionals, FeeBPS: cfg.PaperDefaultFeeBPS,
		IncludeEntryFee: cfg.PaperIncludeEntryFee, IncludeExitFee: cfg.PaperIncludeExitFee,
		IncludeEntrySlippage: cfg.PaperIncludeEntrySlippage, IncludeExitSlippage: cfg.PaperIncludeExitSlippage,
		AllowPartialFill: cfg.PaperAllowPartialFill,
	})
	outcomeSvc := outcome.NewService(repo, marketStore, simulator)
	go outcomeSvc.Run(ctx)

	// Universe service
	universeRepo := universe.NewRepository(repo.Pool())
	universeService := universe.NewService(cfg, universeRepo)

	// Initial universe load
	if err := universeService.Refresh(ctx); err != nil {
		log.Printf("initial universe refresh failed: %v", err)
	}

	// Ensure store has the initial pairs
	for _, p := range universeService.ActivePairs() {
		marketStore.EnsurePair(p.Symbol, p.Tier)
	}

	go universeService.StartRefresher(ctx)

	var dataSource domain.DataSource
	if cfg.MarketMode == "mock" {
		dataSource = domain.DataSourceMock
		var symbols []string
		for _, p := range universeService.ActivePairs() {
			symbols = append(symbols, p.Symbol)
		}
		go mock.New(marketStore, symbols, cfg.GateTimeframes).Run(ctx)
	} else {
		dataSource = domain.DataSourceGate
		connManager := gate.NewConnectionManager(cfg, marketStore, marketRecorder)
		connManager.UpdatePairs(ctx, universeService.ActivePairs())
		go func() {
			ticker := time.NewTicker(cfg.PairUniverseRefreshMin)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					connManager.Stop()
					return
				case <-ticker.C:
					pairs := universeService.ActivePairs()
					for _, p := range pairs {
						marketStore.EnsurePair(p.Symbol, p.Tier)
					}
					connManager.UpdatePairs(ctx, pairs)
				}
			}
		}()
	}

	go scannerLoop(
		ctx,
		cfg,
		marketStore,
		state,
		featureEngine,
		signalEngine,
		repo,
		cache,
		hub,
		qualitySvc,
		qualityMetrics,
		qualityRepo,
		dataSource,
		simulator,
	)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.New(cfg, state, marketStore, repo, hub, universeService, qualitySvc),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       75 * time.Second,
	}

	go func() {
		log.Printf("backend listening on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
	log.Print("backend stopped")
}

func scannerLoop(
	ctx context.Context,
	cfg config.Config,
	marketStore *market.Store,
	state *runtimestate.State,
	featureEngine *features.Engine,
	signalEngine *signals.Engine,
	repo *storage.Repository,
	cache *storage.Cache,
	hub *realtime.Hub,
	qualitySvc *quality.Service,
	qualityMetrics *quality.Metrics,
	qualityRepo *quality.Repository,
	dataSource domain.DataSource,
	simulator *execution_simulation.Simulator,
) {
	ticker := time.NewTicker(cfg.ScanInterval)
	defer ticker.Stop()

	run := func() {
		snapshots := marketStore.Snapshots(cfg.OrderbookDepthPercent)
		computed := make([]domain.FeatureSnapshot, 0, len(snapshots))
		for _, snapshot := range snapshots {
			if err := repo.SaveMarketSnapshot(ctx, snapshot); err != nil {
				log.Printf("save market snapshot %s failed: %v", snapshot.Symbol, err)
			}
			feature := featureEngine.ComputeWithSource(snapshot, dataSource)
			computed = append(computed, feature)
			if err := repo.SaveFeature(ctx, feature); err != nil {
				log.Printf("save feature %s failed: %v", feature.Symbol, err)
			}
		}
		sort.Slice(computed, func(i, j int) bool {
			return computed[i].RuleScore > computed[j].RuleScore
		})
		state.SetFeatures(computed)
		if cache != nil {
			if err := cache.SaveScanner(ctx, computed); err != nil {
				log.Printf("cache scanner failed: %v", err)
			}
		}
		hub.Broadcast("scanner.snapshot", computed)

		// Run data quality evaluation on all snapshots
		qualityReports := qualitySvc.EvaluateAll(snapshots)
		for _, report := range qualityReports {
			qualityMetrics.RecordEvaluation(report)
		}
		// Persist quality reports (best effort)
		if err := qualityRepo.SaveReports(ctx, qualityReports); err != nil {
			log.Printf("save quality reports failed: %v", err)
		}
		// Broadcast quality status to frontend
		hub.Broadcast("quality.snapshot", qualitySvc.AllReports())

		for _, feature := range computed {
			signal, created := signalEngine.Evaluate(ctx, feature)
			if !created {
				continue
			}
			if err := repo.SaveSignal(ctx, *signal); err != nil {
				log.Printf("save signal %s failed: %v", signal.ID, err)
				continue
			}
			signalID := signal.ID
			if err := repo.SaveAIReview(ctx, domain.AIReviewRecord{
				SignalID: &signalID, Pair: signal.Symbol, Timeframe: signal.PrimaryTimeframe,
				Review: signal.AI, ReviewedAt: signal.CreatedAt,
			}); err != nil {
				log.Printf("save AI review %s failed: %v", signal.ID, err)
			}
			if signal.Status == "CONFIRMED" {
				if err := simulator.SimulateEntry(ctx, signal); err != nil {
					log.Printf("simulate entry %s failed: %v", signal.ID, err)
				}
			}
			hub.Broadcast("signal.created", signal)
		}
	}

	run()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func outcomeLoop(
	ctx context.Context,
	marketStore *market.Store,
	repo *storage.Repository,
) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	run := func() {
		items, err := repo.ListOutcomeCandidates(ctx)
		if err != nil {
			log.Printf("list signal outcomes failed: %v", err)
			return
		}
		now := time.Now()
		for _, item := range items {
			snapshot, ok := marketStore.Snapshot(item.Symbol, 0.5)
			if !ok || snapshot.LastPrice <= 0 {
				continue
			}
			if err := repo.UpdateOutcome(ctx, item, snapshot.LastPrice, now); err != nil {
				log.Printf("update outcome %s failed: %v", item.ID, err)
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func mustConnectDatabase(ctx context.Context, databaseURL string) *storage.Repository {
	var lastErr error
	for attempt := 1; attempt <= 20; attempt++ {
		repo, err := storage.Connect(ctx, databaseURL)
		if err == nil {
			return repo
		}
		lastErr = err
		log.Printf("database connection attempt %d failed: %v", attempt, err)
		select {
		case <-ctx.Done():
			log.Fatalf("database connection cancelled: %v", ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}
	log.Fatalf("database unavailable: %v", lastErr)
	return nil
}
