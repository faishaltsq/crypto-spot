package execution_simulation

import (
	"time"
)

// SimulationContext provides the configuration and market data needed to run a simulation.
type SimulationContext struct {
	SignalID      string
	Symbol        string
	EntryPrice    float64
	SnapshotTime  time.Time
	FeeConfig     FeeConfig
	NotionalsUSD  []float64 // E.g. [50, 100, 250, 500, 1000]
}

// FeeConfig defines the trading fees.
type FeeConfig struct {
	TakerBPS float64
	MakerBPS float64
}

// FeeBreakdown details the exact fees paid.
type FeeBreakdown struct {
	EntryFeeUSD float64 `json:"entryFeeUsd"`
	EntryFeeBPS float64 `json:"entryFeeBps"`
	ExitFeeUSD  float64 `json:"exitFeeUsd"`
	ExitFeeBPS  float64 `json:"exitFeeBps"`
	TotalFeeUSD float64 `json:"totalFeeUsd"`
	TotalFeeBPS float64 `json:"totalFeeBps"`
}

// SlippageEstimate estimates how much the price moves against us when placing a market order.
type SlippageEstimate struct {
	NotionalUSD        float64 `json:"notionalUsd"`
	EstimatedFillPrice float64 `json:"estimatedFillPrice"`
	SlippageBPS        float64 `json:"slippageBps"`
	OrderbookLevelsHit int     `json:"orderbookLevelsHit"`
	FullyFilled        bool    `json:"fullyFilled"`
}

// CapacityResult indicates if the orderbook can support the desired size.
type CapacityResult struct {
	MaxSupportedNotionalUSD float64 `json:"maxSupportedNotionalUsd"`
	DepthDepletionPercent   float64 `json:"depthDepletionPercent"`
}

// Result is the comprehensive output of the execution simulation for a single signal.
type Result struct {
	SignalID             string                      `json:"signalId"`
	Symbol               string                      `json:"symbol"`
	SimulatedAt          time.Time                   `json:"simulatedAt"`
	BaseEntryPrice       float64                     `json:"baseEntryPrice"`
	Fees                 FeeBreakdown                `json:"fees"`
	SlippageByNotional   map[float64]SlippageEstimate `json:"slippageByNotional"` // keyed by notional USD
	Capacity             CapacityResult              `json:"capacity"`
	// Aggregate summary for quick filtering
	AvgSlippageBPS       float64                     `json:"avgSlippageBps"`
	TotalCostBPS         float64                     `json:"totalCostBps"` // Fee + Avg Slippage
}
