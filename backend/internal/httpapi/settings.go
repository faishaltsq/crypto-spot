package httpapi

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/example/crypto-spot-signal/internal/settings"
	"github.com/google/uuid"
)

type requestIdentity struct{ ID, Role string }

type settingsRateLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func (l *settingsRateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	recent := l.hits[key][:0]
	for _, hit := range l.hits[key] {
		if now.Sub(hit) < time.Minute {
			recent = append(recent, hit)
		}
	}
	if len(recent) >= 30 {
		l.hits[key] = recent
		return false
	}
	l.hits[key] = append(recent, now)
	return true
}

func (s *Server) identity(r *http.Request) (requestIdentity, bool) {
	// Development adapter only. Production must receive identity from real auth middleware.
	if s.cfg.AppEnv != "development" {
		return requestIdentity{}, false
	}
	id, role := strings.TrimSpace(r.Header.Get("X-User-ID")), strings.ToLower(strings.TrimSpace(r.Header.Get("X-User-Role")))
	if _, err := uuid.Parse(id); err != nil || (role != "admin" && role != "analyst" && role != "viewer") {
		return requestIdentity{}, false
	}
	return requestIdentity{ID: id, Role: role}, true
}

func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, ok := s.identity(r)
		if !ok || identity.Role != "admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
			return
		}
		next(w, r)
	}
}

func (s *Server) preferenceIdentity(r *http.Request) (requestIdentity, bool) { return s.identity(r) }

func (s *Server) settingsPreferences(w http.ResponseWriter, r *http.Request) {
	identity, ok := s.preferenceIdentity(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{"error": "authentication unavailable", "storage": "local"})
		return
	}
	preferences, found, err := s.repo.UserPreferences(r.Context(), identity.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"preferences": preferences, "storage": "database", "found": found})
}

func (s *Server) saveSettingsPreferences(w http.ResponseWriter, r *http.Request) {
	identity, ok := s.preferenceIdentity(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{"error": "authentication unavailable", "storage": "local"})
		return
	}
	var input struct {
		Preferences map[string]interface{} `json:"preferences"`
	}
	if !decodeSettingsJSON(w, r, &input) {
		return
	}
	if input.Preferences == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "preferences required"})
		return
	}
	if err := settings.ValidatePreferences(input.Preferences); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	if err := s.repo.SaveUserPreferences(r.Context(), identity.ID, input.Preferences); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"preferences": input.Preferences, "storage": "database"})
}

func (s *Server) systemSettings(w http.ResponseWriter, r *http.Request) {
	items, err := s.repo.ListSystemSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	activePairs := s.univSvc.ActivePairs()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"marketMode": s.cfg.MarketMode, "activePairLimit": s.cfg.MarketPairLimit, "discoveryLimit": s.cfg.MarketPairLimit,
		"tierAllocation": map[string]int{"A": s.cfg.MarketTierALimit, "B": s.cfg.MarketTierBLimit, "C": s.cfg.MarketTierCLimit},
		"gate":           map[string]interface{}{"status": "configured", "activePairs": len(activePairs)},
		"database":       map[string]string{"status": "connected"}, "redis": map[string]string{"status": "configured"},
		"ai":               map[string]interface{}{"enabled": s.cfg.AIEnabled, "status": "configured"},
		"signalThresholds": map[string]float64{"setup": s.cfg.SignalMinScore, "confirm": s.cfg.SignalConfirmScore, "minimumModelProbability": s.cfg.SignalMinModelProb},
		"paperSimulation":  map[string]bool{"enabled": s.cfg.PaperSimulationEnabled}, "recorder": map[string]bool{"enabled": s.cfg.MarketRecorderEnabled},
		"replay": map[string]bool{"enabled": false}, "applicationVersion": "development", "backendVersion": "development", "frontendVersion": "1.0.0", "schemaVersion": "11", "buildCommitHash": "unavailable", "serverTimezone": time.Now().Location().String(), "runtimeSettings": items,
	})
}

func (s *Server) adminSettings(w http.ResponseWriter, r *http.Request) {
	items, err := s.repo.ListSystemSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"settings": items, "allowedKeys": settings.Keys()})
}

func (s *Server) saveAdminSettings(w http.ResponseWriter, r *http.Request) {
	identity, _ := s.identity(r)
	if !s.settingsLimiter.allow(clientKey(r)) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "settings update rate limit exceeded"})
		return
	}
	var input struct {
		Settings map[string]interface{} `json:"settings"`
		Reason   string                 `json:"reason"`
	}
	if !decodeSettingsJSON(w, r, &input) {
		return
	}
	if len(input.Settings) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "settings required"})
		return
	}
	items, err := s.repo.SaveSystemSettings(r.Context(), input.Settings, identity.ID, strings.TrimSpace(input.Reason))
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"settings": items, "validation": "passed"})
}

func (s *Server) adminSettingsHistory(w http.ResponseWriter, r *http.Request) {
	versions, err := s.repo.ListSettingVersions(r.Context(), 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	audit, err := s.repo.ListSettingAuditLogs(r.Context(), 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"versions": versions, "auditLogs": audit})
}

func decodeSettingsJSON(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return false
	}
	return true
}
func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func (s *Server) resetAdminSettings(w http.ResponseWriter, r *http.Request) {
	identity, _ := s.identity(r)
	var input struct {
		Reason string `json:"reason"`
	}
	if !decodeSettingsJSON(w, r, &input) {
		return
	}
	if err := s.repo.ResetSystemSettings(r.Context(), identity.ID, strings.TrimSpace(input.Reason)); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"result": "runtime overrides reset; restart applies restart-required defaults"})
}
