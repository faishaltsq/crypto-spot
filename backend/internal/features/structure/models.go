package structure

import "time"

// BearishStructure captures the price-structure evidence used by the SELL
// engine: lower highs/lows, support breakdown, and failed reclaim attempts.
// All fields are derived strictly from CLOSED candles (never the currently
// forming candle) to avoid false breakdown signals on intra-candle noise.
type BearishStructure struct {
	Symbol    string `json:"symbol"`
	Timeframe string `json:"timeframe"`

	SupportLevel       float64 `json:"supportLevel"`
	SupportBrokenPrice  float64 `json:"supportBrokenPrice"`
	SupportBroken       bool    `json:"supportBroken"`
	ClosedCandleConfirmed bool  `json:"closedCandleConfirmed"`

	LowerHighDetected bool    `json:"lowerHighDetected"`
	LowerHighPrice    float64 `json:"lowerHighPrice"`
	PriorHighPrice    float64 `json:"priorHighPrice"`

	LowerLowDetected bool    `json:"lowerLowDetected"`
	LowerLowPrice    float64 `json:"lowerLowPrice"`
	PriorLowPrice    float64 `json:"priorLowPrice"`

	ReclaimAttempted bool    `json:"reclaimAttempted"`
	ReclaimFailed    bool    `json:"reclaimFailed"`
	ReclaimHighPrice float64 `json:"reclaimHighPrice"`

	BreakdownFollowThrough bool    `json:"breakdownFollowThrough"`
	FollowThroughPct       float64 `json:"followThroughPct"`

	StructureScore float64   `json:"structureScore"` // 0-100
	CalculatedAt   time.Time `json:"calculatedAt"`
}
