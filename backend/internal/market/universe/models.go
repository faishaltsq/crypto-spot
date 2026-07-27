package universe

import "time"

type RankedPair struct {
	Symbol          string
	Rank            int
	RankScore       float64
	Tier            int
	Qualified       bool
	QuoteVolume24h  float64
	SpreadBps       float64
	DepthScore      float64
	ActivityScore   float64
	SelectionReason map[string]interface{}
	RejectionReason string
}

type UniverseStats struct {
	RequestedLimit int
	QualifiedPairs int
	ActivePairs    int
	TierA          int
	TierB          int
	TierC          int
	StalePairs     int
	LastRefresh    time.Time
}
