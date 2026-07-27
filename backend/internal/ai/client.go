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

type Client struct {
	enabled bool
	url     string
	client  *http.Client
}

func New(enabled bool, url string, timeout time.Duration) *Client {
	return &Client{
		enabled: enabled,
		url:     url,
		client:  &http.Client{Timeout: timeout},
	}
}

func (c *Client) Review(ctx context.Context, feature domain.FeatureSnapshot) domain.AIReview {
	if !c.enabled {
		return deterministic(feature, "disabled")
	}

	payload := map[string]interface{}{
		"symbol":   feature.Symbol,
		"features": feature,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return deterministic(feature, "fallback")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/review", bytes.NewReader(body))
	if err != nil {
		return deterministic(feature, "fallback")
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.client.Do(request)
	if err != nil {
		return deterministic(feature, "fallback")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return deterministic(feature, "fallback")
	}

	var review domain.AIReview
	if err := json.NewDecoder(response.Body).Decode(&review); err != nil {
		return deterministic(feature, "fallback")
	}
	if review.Decision == "" {
		return deterministic(feature, "fallback")
	}
	return review
}

func deterministic(feature domain.FeatureSnapshot, provider string) domain.AIReview {
	decision := "WAIT"
	risk := "HIGH"
	confidence := feature.RuleScore / 100
	if feature.DataQualityScore >= 70 && feature.RuleScore >= 80 && feature.SpoofScore < 50 {
		decision = "CONFIRM"
		risk = "MEDIUM"
	} else if feature.DataQualityScore < 60 || feature.SpoofScore >= 70 {
		decision = "REJECT"
		risk = "HIGH"
	} else if feature.RuleScore >= 70 {
		risk = "MEDIUM"
	}
	return domain.AIReview{
		Decision:    decision,
		Confidence:  confidence,
		RiskLevel:   risk,
		ReasonCodes: feature.Reasons,
		RiskFlags:   feature.RiskFlags,
		Summary:     fmt.Sprintf("Deterministic review based on rule score %.1f and data quality %.1f.", feature.RuleScore, feature.DataQualityScore),
		Provider:    provider,
		Model:       "rule-review-v1",
	}
}
