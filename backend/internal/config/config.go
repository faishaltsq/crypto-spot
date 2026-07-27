package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv                string
	MarketMode            string
	HTTPAddr              string
	DatabaseURL           string
	RedisAddr             string
	RedisPassword         string
	RedisDB               int
	GateWSURL             string
	GateRESTURL           string
	GateTimeframes        []string
	GateOrderbookInterval string
	
	GateWSMaxPairsPerConn int
	GateWSReconnectMin    time.Duration
	GateWSReconnectMax    time.Duration
	GateWSStaleTimeout    time.Duration
	GateWSSubBatchSize    int
	GateWSSubBatchDelay   time.Duration

	MarketPairLimit        int
	MarketQuoteCurrency    string
	MarketStablecoins      []string
	MarketTierALimit       int
	MarketTierBLimit       int
	MarketTierCLimit       int
	PairUniverseRefreshMin time.Duration
	PairMinQuoteVolume24h  float64
	PairMaxSpreadBps       float64
	PairNewListingMinAge   time.Duration

	ScanInterval          time.Duration
	SignalMinScore        float64
	SignalConfirmScore    float64
	SignalMinModelProb    float64
	SignalPairCooldown    time.Duration
	SignalMaxActivePerPair int
	SignalMaxNotifPerHour int
	SignalSetupScore       float64
	SignalMinRuleScore     float64
	SignalMinDataQuality   float64
	SignalMaxSpoofScore    float64
	SignalMaxActiveGlobal  int
	SignalMaxActiveCluster int
	SignalMaxNewPerMinute  int
	OrderbookDepthPercent float64
	MaxSpreadBPS          float64
	MinDepthQuote         float64
	AIEnabled             bool
	AIServiceURL          string
	AITimeout             time.Duration

	// Data quality gate
	DataQualityMinSignalScore     float64
	DataQualityBlockSignalScore   float64
	DataStaleTradeSec             float64
	DataStaleTickerSec            float64
	DataStaleOrderbookSec         float64
	DataStaleCandleSec            float64
	DataReconnectCooldownSec      float64
	DataMaxPriceDeviationBPS      float64
	DataMaxQueueUtilizationPct    float64

	// Market Data Recorder
	MarketRecorderEnabled     bool
	MarketRecorderBatchSize   int
	MarketRecorderFlushIntMs  int
	MarketRecorderMaxBufItems int

	PaperSimulationEnabled      bool
	PaperNotionals              []float64
	PaperIncludeEntryFee        bool
	PaperIncludeExitFee         bool
	PaperIncludeEntrySlippage   bool
	PaperIncludeExitSlippage    bool
	PaperAllowPartialFill       bool
	PaperDefaultFeeBPS          float64
}

func Load() (Config, error) {
	var cfg Config
	cfg.AppEnv = get("APP_ENV", "development")
	cfg.MarketMode = strings.ToLower(get("MARKET_MODE", "gate"))
	cfg.HTTPAddr = get("HTTP_ADDR", ":8080")
	cfg.DatabaseURL = get("DATABASE_URL", "postgres://signal:signal@localhost:5432/signal?sslmode=disable")
	cfg.RedisAddr = get("REDIS_ADDR", "localhost:6379")
	cfg.RedisPassword = os.Getenv("REDIS_PASSWORD")
	cfg.RedisDB = getInt("REDIS_DB", 0)
	cfg.GateWSURL = get("GATE_WS_URL", "wss://api.gateio.ws/ws/v4/")
	cfg.GateRESTURL = get("GATE_REST_URL", "https://api.gateio.ws/api/v4")
	cfg.GateTimeframes = splitCSV(get("GATE_TIMEFRAMES", "10s,1m,5m,15m,30m,1h,4h,8h,1d,7d"), false)
	cfg.GateOrderbookInterval = get("GATE_ORDERBOOK_INTERVAL", "100ms")

	cfg.GateWSMaxPairsPerConn = getInt("GATE_WS_MAX_PAIRS_PER_CONNECTION", 50)
	cfg.GateWSReconnectMin = time.Duration(getInt("GATE_WS_RECONNECT_MIN_SECONDS", 2)) * time.Second
	cfg.GateWSReconnectMax = time.Duration(getInt("GATE_WS_RECONNECT_MAX_SECONDS", 60)) * time.Second
	cfg.GateWSStaleTimeout = time.Duration(getInt("GATE_WS_STALE_TIMEOUT_SECONDS", 30)) * time.Second
	cfg.GateWSSubBatchSize = getInt("GATE_WS_SUBSCRIBE_BATCH_SIZE", 20)
	cfg.GateWSSubBatchDelay = time.Duration(getInt("GATE_WS_SUBSCRIBE_BATCH_DELAY_MS", 250)) * time.Millisecond

	cfg.MarketPairLimit = getInt("MARKET_PAIR_LIMIT", 150)
	cfg.MarketQuoteCurrency = strings.ToUpper(get("MARKET_QUOTE_CURRENCY", "USDT"))
	cfg.MarketStablecoins = splitCSV(get("MARKET_STABLECOINS", "USDT,USDC,DAI,FDUSD,TUSD,USDE,PYUSD,USD1"), true)
	cfg.MarketTierALimit = getInt("MARKET_TIER_A_LIMIT", 30)
	cfg.MarketTierBLimit = getInt("MARKET_TIER_B_LIMIT", 50)
	cfg.MarketTierCLimit = getInt("MARKET_TIER_C_LIMIT", 70)
	cfg.PairUniverseRefreshMin = time.Duration(getInt("PAIR_UNIVERSE_REFRESH_MINUTES", 10)) * time.Minute
	cfg.PairMinQuoteVolume24h = getFloat("PAIR_MIN_QUOTE_VOLUME_24H", 100000)
	cfg.PairMaxSpreadBps = getFloat("PAIR_MAX_SPREAD_BPS", 80)
	cfg.PairNewListingMinAge = time.Duration(getInt("PAIR_NEW_LISTING_MIN_AGE_HOURS", 72)) * time.Hour

	var err error
	if cfg.ScanInterval, err = time.ParseDuration(get("SCAN_INTERVAL", "5s")); err != nil {
		return cfg, fmt.Errorf("invalid SCAN_INTERVAL: %w", err)
	}
	cfg.SignalMinScore = getFloat("SIGNAL_MIN_SCORE", 70)
	cfg.SignalConfirmScore = getFloat("SIGNAL_CONFIRM_SCORE", 80)
	cfg.SignalMinModelProb = getFloat("SIGNAL_MIN_MODEL_PROBABILITY", 0.60)
	cfg.SignalPairCooldown = time.Duration(getInt("SIGNAL_PAIR_COOLDOWN_MINUTES", 30)) * time.Minute
	cfg.SignalMaxActivePerPair = getInt("SIGNAL_MAX_ACTIVE_PER_PAIR", 1)
	cfg.SignalMaxNotifPerHour = getInt("SIGNAL_MAX_NOTIFICATIONS_PER_HOUR", 20)
	cfg.SignalSetupScore = getFloat("SIGNAL_SETUP_SCORE", 70)
	cfg.SignalMinRuleScore = getFloat("SIGNAL_MIN_RULE_SCORE", 65)
	cfg.SignalMinDataQuality = getFloat("SIGNAL_MIN_DATA_QUALITY", 75)
	cfg.SignalMaxSpoofScore = getFloat("SIGNAL_MAX_SPOOF_SCORE", 60)
	cfg.SignalMaxActiveGlobal = getInt("SIGNAL_MAX_ACTIVE_GLOBAL", 15)
	cfg.SignalMaxActiveCluster = getInt("SIGNAL_MAX_ACTIVE_PER_CLUSTER", 3)
	cfg.SignalMaxNewPerMinute = getInt("SIGNAL_MAX_NEW_PER_MINUTE", 5)

	cfg.OrderbookDepthPercent = getFloat("ORDERBOOK_DEPTH_PERCENT", 0.5)
	cfg.MaxSpreadBPS = getFloat("MAX_SPREAD_BPS", 35)
	cfg.MinDepthQuote = getFloat("MIN_DEPTH_QUOTE", 25000)
	cfg.AIEnabled = strings.EqualFold(get("AI_ENABLED", "false"), "true")
	cfg.AIServiceURL = strings.TrimRight(get("AI_SERVICE_URL", "http://localhost:8090"), "/")
	cfg.AITimeout = time.Duration(getInt("AI_TIMEOUT_SECONDS", 12)) * time.Second

	// Data quality gate
	cfg.DataQualityMinSignalScore = getFloat("DATA_QUALITY_MIN_SIGNAL_SCORE", 75)
	cfg.DataQualityBlockSignalScore = getFloat("DATA_QUALITY_BLOCK_SIGNAL_SCORE", 50)
	cfg.DataStaleTradeSec = getFloat("DATA_STALE_TRADE_SECONDS", 10)
	cfg.DataStaleTickerSec = getFloat("DATA_STALE_TICKER_SECONDS", 10)
	cfg.DataStaleOrderbookSec = getFloat("DATA_STALE_ORDERBOOK_SECONDS", 5)
	cfg.DataStaleCandleSec = getFloat("DATA_STALE_CANDLE_SECONDS", 90)
	cfg.DataReconnectCooldownSec = getFloat("DATA_RECONNECT_COOLDOWN_SECONDS", 30)
	cfg.DataMaxPriceDeviationBPS = getFloat("DATA_MAX_PRICE_DEVIATION_BPS", 100)
	cfg.DataMaxQueueUtilizationPct = getFloat("DATA_MAX_QUEUE_UTILIZATION_PERCENT", 85)

	// Market Data Recorder
	cfg.MarketRecorderEnabled = getBool("MARKET_RECORDER_ENABLED", true)
	cfg.MarketRecorderBatchSize = getInt("MARKET_RECORDER_BATCH_SIZE", 500)
	cfg.MarketRecorderFlushIntMs = getInt("MARKET_RECORDER_FLUSH_INTERVAL_MS", 1000)
	cfg.MarketRecorderMaxBufItems = getInt("MARKET_RECORDER_MAX_BUFFER_ITEMS", 50000)

	cfg.PaperSimulationEnabled = getBool("PAPER_SIMULATION_ENABLED", true)
	cfg.PaperDefaultFeeBPS = getFloat("PAPER_DEFAULT_FEE_BPS", 10)
	cfg.PaperIncludeEntryFee = getBool("PAPER_INCLUDE_ENTRY_FEE", true)
	cfg.PaperIncludeExitFee = getBool("PAPER_INCLUDE_EXIT_FEE", true)
	cfg.PaperIncludeEntrySlippage = getBool("PAPER_INCLUDE_ENTRY_SLIPPAGE", true)
	cfg.PaperIncludeExitSlippage = getBool("PAPER_INCLUDE_EXIT_SLIPPAGE", true)
	cfg.PaperAllowPartialFill = getBool("PAPER_ALLOW_PARTIAL_FILL", true)
	notionalsStr := get("PAPER_NOTIONALS", "50,100,250,500,1000")
	if notionalsStr != "" {
		for _, part := range strings.Split(notionalsStr, ",") {
			if val, err := strconv.ParseFloat(strings.TrimSpace(part), 64); err == nil {
				if val > 0 { cfg.PaperNotionals = append(cfg.PaperNotionals, val) }
			}
		}
	}
	if len(cfg.PaperNotionals) == 0 {
		cfg.PaperNotionals = []float64{50, 100, 250, 500, 1000}
	}

	if cfg.MarketMode != "gate" && cfg.MarketMode != "mock" {
		return cfg, fmt.Errorf("MARKET_MODE must be gate or mock")
	}
	if cfg.OrderbookDepthPercent <= 0 {
		return cfg, fmt.Errorf("ORDERBOOK_DEPTH_PERCENT must be positive")
	}
	if cfg.SignalConfirmScore < cfg.SignalSetupScore || cfg.SignalMinModelProb < 0 || cfg.SignalMinModelProb > 1 || cfg.SignalMinDataQuality < 0 || cfg.SignalMinDataQuality > 100 || cfg.SignalMaxSpoofScore < 0 || cfg.SignalMaxSpoofScore > 100 || cfg.SignalMaxActiveGlobal < 0 || cfg.SignalMaxActivePerPair < 0 || cfg.SignalMaxActiveCluster < 0 || cfg.SignalMaxNewPerMinute < 0 || cfg.SignalPairCooldown < 0 { return cfg, fmt.Errorf("invalid signal engine configuration") }
	return cfg, nil
}

func get(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string, toUpper bool) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if toUpper {
			item = strings.ToUpper(item)
		}
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func getInt(key string, fallback int) int {
	value, err := strconv.Atoi(get(key, strconv.Itoa(fallback)))
	if err != nil {
		return fallback
	}
	return value
}

func getFloat(key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(get(key, strconv.FormatFloat(fallback, 'f', -1, 64)), 64)
	if err != nil {
		return fallback
	}
	return value
}

func getBool(key string, fallback bool) bool {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return b
}
