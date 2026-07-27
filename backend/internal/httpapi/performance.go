package httpapi

import (
	"net/http"
	"time"

	"github.com/example/crypto-spot-signal/internal/performance"
)

func (s *Server) performance(w http.ResponseWriter, r *http.Request) {
	filters := performance.Filters{}
	q := r.URL.Query()
	for key, destination := range map[string]*string{"pair": &filters.Pair, "tier": &filters.Tier, "timeframe": &filters.Timeframe, "signalStatus": &filters.SignalStatus, "scoreBucket": &filters.ScoreBucket, "marketRegime": &filters.MarketRegime, "ruleVersion": &filters.RuleVersion, "modelVersion": &filters.ModelVersion, "aiDecision": &filters.AIDecision, "notional": &filters.Notional} {
		*destination = q.Get(key)
	}
	for key, destination := range map[string]**time.Time{"dateFrom": &filters.DateFrom, "dateTo": &filters.DateTo} {
		if value := q.Get(key); value != "" {
			parsed, err := time.Parse(time.RFC3339, value)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": key + " must be RFC3339"})
				return
			}
			*destination = &parsed
		}
	}
	report, err := s.repo.PerformanceReport(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"metrics": report.Metrics, "breakdowns": report.Breakdowns, "returnHorizons": report.Horizons, "edgeScore": report.EdgeScore, "warnings": report.Warnings, "statusCounts": report.StatusCounts, "reliabilityStatus": report.Reliability, "reliabilityDefinition": report.ReliabilityDefinition, "calculationTimestamp": report.CalculationTimestamp, "filters": q, "dateRange": map[string]interface{}{"from": filters.DateFrom, "to": filters.DateTo}, "charts": map[string]interface{}{"cumulativeNetReturn": report.CumulativeNet, "cumulativeGrossReturn": report.CumulativeGross, "drawdown": report.Drawdown}, "unit": "decimal return: 0.01 = 1%"})
}
