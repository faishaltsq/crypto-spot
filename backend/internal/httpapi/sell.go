package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/crypto-spot-signal/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// listActiveSignals handles GET /api/v1/signals/active?direction=&strategy=&symbol=&timeframe=&limit=
// Returns the single unified BUY+SELL active-signal snapshot every client
// (Terminal, Signals page) must use — the backend, not the frontend,
// decides what "active" means (see domain.SignalIsActiveAt).
func (s *Server) listActiveSignals(w http.ResponseWriter, r *http.Request) {
	filter := storage.ActiveSignalFilter{
		Direction: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("direction"))),
		Strategy:  strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("strategy"))),
		Symbol:    strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol"))),
		Timeframe: strings.TrimSpace(r.URL.Query().Get("timeframe")),
	}
	if limit, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
		filter.Limit = limit
	}

	signals, err := s.repo.ListActiveSignals(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"signals":     signals,
		"total":       len(signals),
		"snapshot_at": time.Now().UTC().Format(time.RFC3339),
		"next_cursor": nil,
	})
}

// listSellSignals handles GET /api/v1/sell/signals?symbol=&limit=
// Returns SELL-family signals (SELL_SETUP, SELL_CONFIRMED, TAKE_PROFIT_SUGGESTED,
// AVOID_ENTRY, EXIT_WARNING) with their trade-flow/structure/spoof evidence.
func (s *Server) listSellSignals(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol")))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	signals, err := s.repo.ListSellSignals(r.Context(), symbol, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"signals": signals})
}

// getSellSignal handles GET /api/v1/sell/signals/{id}.
func (s *Server) getSellSignal(w http.ResponseWriter, r *http.Request) {
	detail, err := s.repo.GetSellSignal(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// getSellSignalOutcome handles GET /api/v1/sell/signals/{id}/outcome, the
// directional-accuracy result computed by signals/sell/outcome.go. Returns
// 404 if the signal hasn't been evaluated yet (too young, or never closed).
func (s *Server) getSellSignalOutcome(w http.ResponseWriter, r *http.Request) {
	outcome, err := s.repo.GetSellSignalOutcome(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, outcome)
}
