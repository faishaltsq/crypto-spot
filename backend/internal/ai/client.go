package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
)

const (
	promptVersion = "ai-review-v1"
	schemaVersion = "ai-review-schema-v1"
)

type Client struct {
	enabled          bool
	url              string
	client           *http.Client
	dynamicThreshold float64
}

type reviewRequest struct {
	FeatureVersion string         `json:"feature_version"`
	PromptVersion  string         `json:"prompt_version"`
	Features       featureSummary `json:"features"`
}

// featureSummary is an explicit prompt allowlist. Raw books, trades, prices,
// credentials, and unbounded feature payloads cannot cross this boundary.
type featureSummary struct {
	Pair                  string             `json:"pair"`
	Timeframe             string             `json:"timeframe"`
	MarketRegime          string             `json:"market_regime"`
	FinalScore            float64            `json:"final_score"`
	DynamicThreshold      float64            `json:"dynamic_threshold"`
	RuleScore             float64            `json:"rule_score"`
	ModelProbability      float64            `json:"model_probability"`
	DataQuality           float64            `json:"data_quality"`
	VolumeFeatures        map[string]float64 `json:"volume_features"`
	OrderFlowFeatures     map[string]float64 `json:"order_flow_features"`
	LiquidityFeatures     map[string]float64 `json:"liquidity_features"`
	SpoofRisk             float64            `json:"spoof_risk"`
	SupportingEvidence    []string           `json:"supporting_evidence"`
	ContradictingEvidence []string           `json:"contradicting_evidence"`
	RiskFlags             []string           `json:"risk_flags"`
}

func New(enabled bool, url string, timeout time.Duration, dynamicThreshold float64) *Client {
	return &Client{enabled: enabled, url: url, client: &http.Client{Timeout: timeout}, dynamicThreshold: dynamicThreshold}
}

func (c *Client) Review(ctx context.Context, feature domain.FeatureSnapshot) domain.AIReview {
	if len(feature.BlockedReasons) > 0 || feature.Status == "BLOCKED" {
		review := deterministic(feature, "deterministic", "AI_BLOCKED_BY_RULES")
		review.Decision = "UNAVAILABLE"
		return review
	}
	if !c.enabled {
		return deterministic(feature, "deterministic", "AI_DISABLED")
	}

	body, err := json.Marshal(reviewRequest{
		FeatureVersion: "feature-v2",
		PromptVersion:  promptVersion,
		Features:       summarize(feature, c.dynamicThreshold),
	})
	if err != nil {
		return deterministic(feature, "deterministic", "AI_REQUEST_INVALID")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/review", bytes.NewReader(body))
	if err != nil {
		return deterministic(feature, "deterministic", "AI_REQUEST_INVALID")
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return deterministic(feature, "deterministic", "AI_PROVIDER_ERROR")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return deterministic(feature, "deterministic", "AI_PROVIDER_ERROR")
	}

	var review domain.AIReview
	if err := json.NewDecoder(response.Body).Decode(&review); err != nil || !validReview(review) {
		return deterministic(feature, "deterministic", "AI_RESPONSE_INVALID")
	}
	return review
}

func summarize(feature domain.FeatureSnapshot, dynamicThreshold float64) featureSummary {
	regime := "neutral"
	if feature.TrendAlignment >= 0.2 {
		regime = "bullish"
	} else if feature.TrendAlignment < 0 {
		regime = "bearish"
	}
	return featureSummary{
		Pair:             feature.Symbol,
		Timeframe:        primaryTimeframe(feature),
		MarketRegime:     regime,
		FinalScore:       feature.RuleScore,
		DynamicThreshold: dynamicThreshold,
		RuleScore:        feature.RuleScore,
		ModelProbability: 0,
		DataQuality:      feature.DataQualityScore,
		VolumeFeatures: map[string]float64{
			"relative_volume_1m": feature.RelativeVolume1m,
			"buy_ratio_1m":       feature.BuyRatio1m,
		},
		OrderFlowFeatures: map[string]float64{
			"volume_delta_ratio_1m": feature.VolumeDeltaRatio1m,
			"orderbook_imbalance":   feature.OrderbookImbalance,
		},
		LiquidityFeatures: map[string]float64{
			"liquidity_score": feature.LiquidityScore,
			"spread_bps":      feature.SpreadBPS,
			"bid_depth_quote": feature.BidDepthQuote,
			"ask_depth_quote": feature.AskDepthQuote,
		},
		SpoofRisk:             feature.SpoofScore,
		SupportingEvidence:    limited(feature.Reasons),
		ContradictingEvidence: limited(append(append([]string{}, feature.RiskFlags...), feature.MissingFeatures...)),
		RiskFlags:             limited(feature.RiskFlags),
	}
}

func primaryTimeframe(feature domain.FeatureSnapshot) string {
	if feature.TrendByTimeframe["15m"] == "bullish" {
		return "15m"
	}
	return "1m"
}

func limited(values []string) []string {
	if len(values) > 8 {
		return append([]string(nil), values[:8]...)
	}
	return append([]string(nil), values...)
}

func validReview(review domain.AIReview) bool {
	if review.Confidence < 0 || review.Confidence > 1 || review.Summary == "" ||
		review.Provider == "" || review.Model == "" || review.LatencyMS < 0 ||
		review.PromptVersion != promptVersion || review.SchemaVersion != schemaVersion ||
		len(review.SupportingReasonCodes) > 8 || len(review.ContradictingReasonCodes) > 8 || len(review.RiskFlags) > 8 {
		return false
	}
	switch review.Decision {
	case "CONFIRM", "REJECT", "WAIT", "UNAVAILABLE":
		return true
	default:
		return false
	}
}

func deterministic(feature domain.FeatureSnapshot, provider, fallbackReason string) domain.AIReview {
	decision := "WAIT"
	if feature.DataQualityScore < 60 || feature.SpoofScore >= 70 {
		decision = "REJECT"
	} else if feature.DataQualityScore >= 70 && feature.RuleScore >= 80 && feature.SpoofScore < 50 {
		decision = "CONFIRM"
	}
	return domain.AIReview{
		Decision:                 decision,
		Confidence:               clamp(feature.RuleScore/100, 0, 1),
		Summary:                  fmt.Sprintf("Deterministic review of supplied rule and data-quality features."),
		SupportingReasonCodes:    limited(feature.Reasons),
		ContradictingReasonCodes: limited(feature.RiskFlags),
		RiskFlags:                limited(feature.RiskFlags),
		Provider:                 provider,
		Model:                    "rule-review-v1",
		Fallback:                 true,
		FallbackReason:           fallbackReason,
		ProviderErrorCode:        fallbackReason,
		PromptVersion:            promptVersion,
		SchemaVersion:            schemaVersion,
	}
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
