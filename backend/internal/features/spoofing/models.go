package spoofing

import "time"

// WallAnalysis extends the existing domain.BookMetrics spoof score with
// SELL-specific interpretation: whether a large bid wall visible in the book
// actually held (absorbed sell flow) or failed (was pulled/consumed,
// confirming real breakdown pressure rather than a fake wall).
type WallAnalysis struct {
	Symbol string `json:"symbol"`

	BidWallDetected   bool    `json:"bidWallDetected"`
	BidWallPrice      float64 `json:"bidWallPrice"`
	BidWallQuote      float64 `json:"bidWallQuote"`
	BidWallFailed     bool    `json:"bidWallFailed"`
	BidWallFailureConfidence float64 `json:"bidWallFailureConfidence"`

	AskWallDetected bool    `json:"askWallDetected"`
	AskWallPrice    float64 `json:"askWallPrice"`
	AskWallQuote    float64 `json:"askWallQuote"`
	AskWallFailed   bool    `json:"askWallFailed"`
	AskWallFailureConfidence float64 `json:"askWallFailureConfidence"`

	IcebergSuspected     bool    `json:"icebergSuspected"`
	IcebergConfidence    float64 `json:"icebergConfidence"`

	SpoofScore  float64 `json:"spoofScore"` // carried over from domain.BookMetrics
	CalculatedAt time.Time `json:"calculatedAt"`
}
