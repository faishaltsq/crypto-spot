package httpapi

import (
	"net/http"
	"strconv"
)

func (s *Server) universePairs(w http.ResponseWriter, r *http.Request) {
	tierQuery := r.URL.Query().Get("tier")
	pairs := s.univSvc.ActivePairs()
	
	if tierQuery != "" {
		tier, err := strconv.Atoi(tierQuery)
		if err == nil {
			var filtered []interface{}
			for _, p := range pairs {
				if p.Tier == tier {
					filtered = append(filtered, p)
				}
			}
			writeJSON(w, http.StatusOK, filtered)
			return
		}
	}
	
	writeJSON(w, http.StatusOK, pairs)
}

func (s *Server) universeStats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.univSvc.Stats())
}

func (s *Server) universeRefresh(w http.ResponseWriter, r *http.Request) {
	// In a real app, verify admin auth here
	
	if err := s.univSvc.Refresh(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "refresh triggered",
		"stats":  s.univSvc.Stats(),
	})
}
