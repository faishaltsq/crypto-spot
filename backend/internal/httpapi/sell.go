package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

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
