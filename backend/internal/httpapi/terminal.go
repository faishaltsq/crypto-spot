package httpapi

import (
	"net/http"
	"strings"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type TerminalSnapshot struct {
	Pair       *domain.FeatureSnapshot    `json:"pair"` // We can use feature snapshot for comprehensive pair data
	Diagnostic *domain.FeatureSnapshot    `json:"diagnostic"` // The same for diagnostic
	Candles    map[string][]domain.Candle `json:"candles"`
	Orderbook  domain.BookMetrics         `json:"orderbook"`
	TopBids    []domain.Level             `json:"topBids"`
	TopAsks    []domain.Level             `json:"topAsks"`
	Trades     []domain.Trade             `json:"trades"`
	Signals    []domain.Signal            `json:"signals"`
}

func (s *Server) terminalSnapshot(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(chi.URLParam(r, "symbol"))
	
	// 1. Get PairSnapshot from Market Store
	marketSnapshot, ok := s.market.Snapshot(symbol, s.cfg.OrderbookDepthPercent)
	if !ok {
		writeError(w, http.StatusNotFound, pgx.ErrNoRows)
		return
	}
	
	// 2. Get FeatureSnapshot (Diagnostic)
	var feature *domain.FeatureSnapshot
	for _, f := range s.state.Features() {
		if f.Symbol == symbol {
			feature = &f
			break
		}
	}
	
	// 3. Get Signals for this pair
	// Note: Our repository ListSignals might not filter by symbol yet, let's just get the recent ones and filter
	// In a real app we'd add ListSignalsBySymbol to the repository
	signals, err := s.repo.ListSignals(r.Context(), 500)
	var pairSignals []domain.Signal
	if err == nil {
		for _, sig := range signals {
			if sig.Symbol == symbol {
				pairSignals = append(pairSignals, sig)
			}
		}
	}
	if len(pairSignals) > 50 {
		pairSignals = pairSignals[:50] // Limit to recent 50
	}
	
	snapshot := TerminalSnapshot{
		Pair:       feature,
		Diagnostic: feature,
		Candles:    marketSnapshot.Candles,
		Orderbook:  marketSnapshot.Book,
		TopBids:    marketSnapshot.TopBids,
		TopAsks:    marketSnapshot.TopAsks,
		Trades:     marketSnapshot.Trades,
		Signals:    pairSignals,
	}
	
	writeJSON(w, http.StatusOK, snapshot)
}
