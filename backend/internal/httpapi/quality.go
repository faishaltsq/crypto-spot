package httpapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (s *Server) qualityPairs(w http.ResponseWriter, _ *http.Request) {
	if s.qualitySvc == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	writeJSON(w, http.StatusOK, s.qualitySvc.AllReports())
}

func (s *Server) qualityPair(w http.ResponseWriter, r *http.Request) {
	if s.qualitySvc == nil {
		writeError(w, http.StatusNotFound, nil)
		return
	}
	symbol := strings.ToUpper(chi.URLParam(r, "symbol"))
	report, ok := s.qualitySvc.GetReport(symbol)
	if !ok {
		writeError(w, http.StatusNotFound, nil)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) qualityStats(w http.ResponseWriter, _ *http.Request) {
	if s.qualitySvc == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{})
		return
	}
	writeJSON(w, http.StatusOK, s.qualitySvc.Stats())
}

func (s *Server) systemHealth(w http.ResponseWriter, _ *http.Request) {
	data := map[string]interface{}{
		"ws": map[string]interface{}{
			"activeConnections": s.hub.ClientCount(),
		},
		"quality": s.qualitySvc.Stats(),
		"status":  "HEALTHY",
	}
	writeJSON(w, http.StatusOK, data)
}
