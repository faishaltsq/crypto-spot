package universe

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/crypto-spot-signal/internal/config"
)

type GateTicker struct {
	CurrencyPair string `json:"currency_pair"`
	Last         string `json:"last"`
	LowestAsk    string `json:"lowest_ask"`
	HighestBid   string `json:"highest_bid"`
	ChangeVol    string `json:"change_percentage"`
	BaseVolume   string `json:"base_volume"`
	QuoteVolume  string `json:"quote_volume"`
}

type GateCurrencyPair struct {
	Id          string `json:"id"`
	Base        string `json:"base"`
	Quote       string `json:"quote"`
	TradeStatus string `json:"trade_status"`
}

type Service struct {
	cfg     config.Config
	repo    *Repository
	client  *http.Client
	stats   UniverseStats
	active  []RankedPair
}

func NewService(cfg config.Config, repo *Repository) *Service {
	return &Service{
		cfg:    cfg,
		repo:   repo,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *Service) Refresh(ctx context.Context) error {
	log.Println("universe: starting refresh")
	
	pairs, err := s.fetchCurrencyPairs(ctx)
	if err != nil {
		return err
	}
	
	tickers, err := s.fetchTickers(ctx)
	if err != nil {
		return err
	}

	tickerMap := make(map[string]GateTicker)
	for _, t := range tickers {
		tickerMap[t.CurrencyPair] = t
	}

	var candidates []PairCandidate
	for _, p := range pairs {
		if p.TradeStatus != "tradable" {
			continue
		}
		
		t, ok := tickerMap[p.Id]
		if !ok {
			continue
		}

		quoteVol, _ := strconv.ParseFloat(t.QuoteVolume, 64)
		bestBid, _ := strconv.ParseFloat(t.HighestBid, 64)
		bestAsk, _ := strconv.ParseFloat(t.LowestAsk, 64)
		
		spreadBps := CalculateSpreadBps(bestBid, bestAsk)

		valid, _ := IsValidSpotPair(p.Base, p.Quote, s.cfg.PairMinQuoteVolume24h, quoteVol, s.cfg.MarketStablecoins, spreadBps, s.cfg.PairMaxSpreadBps)
		if !valid {
			continue
		}

		candidates = append(candidates, PairCandidate{
			Symbol:         p.Id,
			Base:           p.Base,
			Quote:          p.Quote,
			QuoteVolume24h: quoteVol,
			BestBid:        bestBid,
			BestAsk:        bestAsk,
		})
	}

	ranked := RankCandidates(candidates, s.cfg.PairMaxSpreadBps)
	
	tierLimits := TierLimits{
		LimitA: 30,
		LimitB: 50,
		LimitC: 70,
	}
	
	active := AssignTiers(ranked, tierLimits)
	
	if len(active) > s.cfg.MarketPairLimit {
		active = active[:s.cfg.MarketPairLimit]
	}

	if err := s.repo.UpsertPairs(ctx, active); err != nil {
		return fmt.Errorf("upsert pairs: %w", err)
	}

	s.active = active
	
	var a, b, c int
	for _, p := range active {
		switch p.Tier {
		case 1: a++
		case 2: b++
		case 3: c++
		}
	}

	s.stats = UniverseStats{
		RequestedLimit: s.cfg.MarketPairLimit,
		QualifiedPairs: len(active),
		ActivePairs:    len(active),
		TierA:          a,
		TierB:          b,
		TierC:          c,
		LastRefresh:    time.Now(),
	}

	log.Printf("universe: refresh complete. active=%d, tierA=%d, tierB=%d, tierC=%d", 
		len(active), a, b, c)

	return nil
}

func (s *Service) ActivePairs() []RankedPair {
	return s.active
}

func (s *Service) Stats() UniverseStats {
	return s.stats
}

func (s *Service) fetchCurrencyPairs(ctx context.Context) ([]GateCurrencyPair, error) {
	url := strings.TrimRight(s.cfg.GateRESTURL, "/") + "/spot/currency_pairs"
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch pairs: %w", err)
	}
	defer resp.Body.Close()
	
	var pairs []GateCurrencyPair
	if err := json.NewDecoder(resp.Body).Decode(&pairs); err != nil {
		return nil, fmt.Errorf("decode pairs: %w", err)
	}
	return pairs, nil
}

func (s *Service) fetchTickers(ctx context.Context) ([]GateTicker, error) {
	url := strings.TrimRight(s.cfg.GateRESTURL, "/") + "/spot/tickers"
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch tickers: %w", err)
	}
	defer resp.Body.Close()
	
	var tickers []GateTicker
	if err := json.NewDecoder(resp.Body).Decode(&tickers); err != nil {
		return nil, fmt.Errorf("decode tickers: %w", err)
	}
	return tickers, nil
}
