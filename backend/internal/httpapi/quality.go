package httpapi

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/example/crypto-spot-signal/internal/quality"
	"github.com/go-chi/chi/v5"
)

func (s *Server) qualityPairs(w http.ResponseWriter, r *http.Request) {
	if s.qualitySvc == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"pairs": []interface{}{}, "total": 0, "page": 1, "limit": 50})
		return
	}
	reports := s.qualitySvc.AllReports()
	status, tier, sortBy := strings.ToUpper(r.URL.Query().Get("status")), r.URL.Query().Get("tier"), r.URL.Query().Get("sort")
	allowed := r.URL.Query().Get("signal_allowed")
	minimumScore, _ := strconv.ParseFloat(r.URL.Query().Get("minimum_score"), 64)
	pairs := s.univSvc.ActivePairs()
	tiers := make(map[string]int, len(pairs))
	for _, pair := range pairs {
		tiers[pair.Symbol] = pair.Tier
	}
	filtered := make([]map[string]interface{}, 0, len(reports))
	for _, report := range reports {
		if status != "" && status != "ALL" && string(report.Status) != status {
			continue
		}
		if tier != "" && tier != strconv.Itoa(tiers[report.Symbol]) {
			continue
		}
		if allowed == "true" && !report.SignalAllowed {
			continue
		}
		if allowed == "false" && report.SignalAllowed {
			continue
		}
		if r.URL.Query().Get("minimum_score") != "" && report.Score < minimumScore {
			continue
		}
		filtered = append(filtered, qualityPairPayload(report, tiers[report.Symbol], s.cfg.MarketMode))
	}
	sort.Slice(filtered, func(i, j int) bool {
		if sortBy == "last_update" {
			return filtered[i]["updated_at"].(string) > filtered[j]["updated_at"].(string)
		}
		if sortBy == "blocked_signal_count" {
			return filtered[i]["signal_allowed"].(bool) == false && filtered[j]["signal_allowed"].(bool)
		}
		return filtered[i]["quality_score"].(float64) < filtered[j]["quality_score"].(float64)
	})
	page, limit := positiveInt(r.URL.Query().Get("page"), 1), positiveInt(r.URL.Query().Get("limit"), 50)
	if limit > 100 {
		limit = 100
	}
	start := (page - 1) * limit
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"pairs": filtered[start:end], "total": len(filtered), "page": page, "limit": limit})
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
	tier := 0
	for _, pair := range s.univSvc.ActivePairs() {
		if pair.Symbol == symbol {
			tier = pair.Tier
			break
		}
	}
	writeJSON(w, http.StatusOK, qualityPairPayload(report, tier, s.cfg.MarketMode))
}

func (s *Server) qualityStats(w http.ResponseWriter, _ *http.Request) {
	if s.qualitySvc == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{})
		return
	}
	stats := s.qualitySvc.Stats()
	var average interface{}
	if stats.Total > 0 {
		average = stats.AvgScore
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"market_mode": s.cfg.MarketMode, "active_pairs": len(s.univSvc.ActivePairs()), "healthy_pairs": stats.Valid, "degraded_pairs": stats.Degraded, "blocked_pairs": stats.Blocked, "stale_pairs": stats.Stale, "unsynced_orderbooks": stats.Unsynced, "incomplete_pairs": stats.Incomplete, "sequence_gaps": nil, "reconnect_count": nil, "dropped_events": nil, "average_quality_score": average, "signals_blocked": stats.BlockingSignals, "updated_at": time.Now().UTC()})
}

func (s *Server) qualitySummary(w http.ResponseWriter, r *http.Request) { s.qualityStats(w, r) }

func (s *Server) qualityReasons(w http.ResponseWriter, _ *http.Request) {
	if s.qualitySvc == nil {
		writeJSON(w, http.StatusOK, map[string]int{})
		return
	}
	totals := map[string]int{}
	for _, report := range s.qualitySvc.AllReports() {
		for _, reason := range report.Reasons {
			totals[string(reason)]++
		}
	}
	writeJSON(w, http.StatusOK, totals)
}

func (s *Server) qualityHistory(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(chi.URLParam(r, "symbol"))
	rows, err := s.repo.Pool().Query(r.Context(), `SELECT score, status, reasons, signal_allowed, evaluated_at FROM data_quality_history WHERE symbol = $1 ORDER BY evaluated_at DESC LIMIT 200`, symbol)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	items := []map[string]interface{}{}
	for rows.Next() {
		var score float64
		var status string
		var reasons []byte
		var allowed bool
		var evaluated time.Time
		if err := rows.Scan(&score, &status, &reasons, &allowed, &evaluated); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		var parsed interface{}
		_ = json.Unmarshal(reasons, &parsed)
		items = append(items, map[string]interface{}{"quality_score": score, "quality_status": status, "blocked_reasons": parsed, "signal_allowed": allowed, "updated_at": evaluated})
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"symbol": symbol, "history": items})
}

func qualityPairPayload(report quality.QualityReport, tier int, marketMode string) map[string]interface{} {
	var tierValue interface{}
	if tier > 0 {
		tierValue = tier
	}
	return map[string]interface{}{"symbol": report.Symbol, "tier": tierValue, "exchange": "GATE", "market_type": "SPOT", "data_source": strings.ToUpper(marketMode), "quality_score": report.Score, "quality_status": report.Status, "signal_allowed": report.SignalAllowed, "trade_stream": map[string]interface{}{"last_event_at": nullableTimestamp(report.Freshness.Trade)}, "ticker_stream": map[string]interface{}{"last_event_at": nullableTimestamp(report.Freshness.Ticker)}, "orderbook": map[string]interface{}{"status": orderbookStatus(report), "resync_count": nil, "reconnect_count": nil, "last_update_at": nullableTimestamp(report.Freshness.Book)}, "candle": map[string]interface{}{"last_closed_at": nullableTimestamp(report.Freshness.Candle)}, "queue_utilization": report.Pipeline.QueueUtilization, "redis_latency_ms": report.Persistence.RedisLatencyMs, "database_backlog_size": report.Persistence.DBBacklogSize, "missing_features": failedRuleCodes(report, quality.ReasonIncompleteFeatures), "blocked_reasons": report.Reasons, "rule_results": report.RuleResults, "updated_at": report.EvaluatedAt.UTC().Format(time.RFC3339)}
}
func nullableTimestamp(value time.Time) interface{} {
	if value.IsZero() {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}
func failedRuleCodes(report quality.QualityReport, code quality.ReasonCode) []string {
	for _, result := range report.RuleResults {
		if result.Code == code && !result.Passed {
			return []string{string(code)}
		}
	}
	return []string{}
}
func orderbookStatus(report quality.QualityReport) string {
	for _, reason := range report.Reasons {
		if reason == quality.ReasonOrderbookUnsynced || reason == quality.ReasonOrderbookResync {
			return "UNSYNCED"
		}
	}
	return "SYNCED"
}
func positiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
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
