package httpapi

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/crypto-spot-signal/internal/config"
	"github.com/example/crypto-spot-signal/internal/market"
	"github.com/example/crypto-spot-signal/internal/market/universe"
	"github.com/example/crypto-spot-signal/internal/quality"
	"github.com/example/crypto-spot-signal/internal/realtime"
	runtimestate "github.com/example/crypto-spot-signal/internal/runtime"
	"github.com/example/crypto-spot-signal/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5"
)

type Server struct {
	cfg        config.Config
	state      *runtimestate.State
	market     *market.Store
	repo       *storage.Repository
	hub        *realtime.Hub
	univSvc    *universe.Service
	qualitySvc *quality.Service
}

func New(
	cfg config.Config,
	state *runtimestate.State,
	marketStore *market.Store,
	repo *storage.Repository,
	hub *realtime.Hub,
	univSvc *universe.Service,
	qualitySvc *quality.Service,
) http.Handler {
	server := &Server{
		cfg:        cfg,
		state:      state,
		market:     marketStore,
		repo:       repo,
		hub:        hub,
		univSvc:    univSvc,
		qualitySvc: qualitySvc,
	}

	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "https://*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	router.Get("/health", server.health)
	router.Get("/ws", server.hub.ServeHTTP)

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/scanner", server.scanner)
		r.Get("/signals", server.listSignals)
		r.Get("/signals/{id}", server.getSignal)
		r.Post("/signals/export", server.exportSignals)
		r.Get("/pairs/{symbol}", server.pair)
		r.Get("/terminal/{symbol}", server.terminalSnapshot)
		r.Get("/config", server.publicConfig)
		r.Get("/performance/summary", server.performanceSummary)

		// Data quality endpoints
		r.Get("/quality/pairs", server.qualityPairs)
		r.Get("/quality/pairs/{symbol}", server.qualityPair)
		r.Get("/quality/stats", server.qualityStats)
		r.Get("/health/system", server.systemHealth)

		r.Route("/market/universe", func(r chi.Router) {
			r.Get("/", server.universePairs)
			r.Get("/stats", server.universeStats)
			r.Post("/refresh", server.universeRefresh)
		})
	})
	return router
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "ok",
		"mode":       "paper-signal-only",
		"marketMode": s.cfg.MarketMode,
	})
}

func (s *Server) scanner(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.state.Features())
}

func (s *Server) listSignals(w http.ResponseWriter, r *http.Request) {
	filter := parseSignalFilter(r)
	signals, total, err := s.repo.ListSignalsFiltered(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"signals": signals,
		"total":   total,
	})
}

func (s *Server) exportSignals(w http.ResponseWriter, r *http.Request) {
	filter := parseSignalFilter(r)
	signals, _, err := s.repo.ListSignalsFiltered(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=signals.csv")
	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{"id", "symbol", "type", "status", "primary_timeframe", "entry_price", "target_price_1", "target_price_2", "invalidation_price", "rule_score", "paper_notional", "net_return_decimal", "simulation_status", "created_at", "expires_at"}
	if err := writer.Write(header); err != nil {
		return
	}

	for _, sig := range signals {
		notional, netReturn, simulationStatus := "", "", ""
		if len(sig.Simulations) > 0 { simulation := sig.Simulations[0]; notional = fmt.Sprintf("%.2f", simulation.Notional); simulationStatus = simulation.SimulationStatus; if simulation.NetReturn != nil { netReturn = fmt.Sprintf("%.8f", *simulation.NetReturn) } }
		row := []string{
			sig.ID,
			sig.Symbol,
			sig.Type,
			sig.Status,
			sig.PrimaryTimeframe,
			fmt.Sprintf("%.8f", sig.EntryPrice),
			fmt.Sprintf("%.8f", sig.Target1),
			fmt.Sprintf("%.8f", sig.Target2),
			fmt.Sprintf("%.8f", sig.Invalidation),
			fmt.Sprintf("%.4f", sig.RuleScore),
			notional, netReturn, simulationStatus,
			sig.CreatedAt.Format(time.RFC3339),
			sig.ExpiresAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return
		}
	}
}

func parseSignalFilter(r *http.Request) storage.SignalFilter {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	var filter storage.SignalFilter
	filter.Limit = limit
	filter.Offset = offset
	filter.OrderBy = r.URL.Query().Get("orderBy")

	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}
	if v := r.URL.Query().Get("symbol"); v != "" {
		filter.Symbol = &v
	}
	if v := r.URL.Query().Get("signalType"); v != "" {
		filter.SignalType = &v
	}
	if v := r.URL.Query().Get("createdFrom"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.CreatedFrom = &t
		}
	}
	if v := r.URL.Query().Get("createdTo"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.CreatedTo = &t
		}
	}
	if v := r.URL.Query().Get("scoreMin"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			filter.ScoreMin = &f
		}
	}
	if v := r.URL.Query().Get("scoreMax"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			filter.ScoreMax = &f
		}
	}

	return filter
}

func (s *Server) getSignal(w http.ResponseWriter, r *http.Request) {
	signal, err := s.repo.GetSignal(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, signal)
}

func (s *Server) pair(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(chi.URLParam(r, "symbol"))
	snapshot, ok := s.market.Snapshot(symbol, s.cfg.OrderbookDepthPercent)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "pair not found"})
		return
	}
	feature, _ := s.state.Feature(symbol)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"market":  snapshot,
		"feature": feature,
	})
}

func (s *Server) performanceSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.repo.PerformanceSummary(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) publicConfig(w http.ResponseWriter, _ *http.Request) {
	activePairs := s.univSvc.ActivePairs()
	pairNames := make([]string, len(activePairs))
	for i, p := range activePairs {
		pairNames[i] = p.Symbol
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"marketMode":            s.cfg.MarketMode,
		"pairs":                 pairNames,
		"timeframes":            s.cfg.GateTimeframes,
		"scanInterval":          s.cfg.ScanInterval.String(),
		"signalMinScore":        s.cfg.SignalMinScore,
		"signalCooldown":        s.cfg.SignalPairCooldown.String(),
		"orderbookDepthPercent": s.cfg.OrderbookDepthPercent,
		"maxSpreadBps":          s.cfg.MaxSpreadBPS,
		"minDepthQuote":         s.cfg.MinDepthQuote,
		"aiEnabled":             s.cfg.AIEnabled,
		"executionEnabled":      false,
	})
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
