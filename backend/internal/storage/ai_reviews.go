package storage

import (
	"context"
	"encoding/json"

	"github.com/example/crypto-spot-signal/internal/domain"
)

func (r *Repository) SaveAIReview(ctx context.Context, record domain.AIReviewRecord) error {
	supporting, _ := json.Marshal(record.Review.SupportingReasonCodes)
	contradicting, _ := json.Marshal(record.Review.ContradictingReasonCodes)
	riskFlags, _ := json.Marshal(record.Review.RiskFlags)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ai_reviews (
			signal_id, pair, timeframe, provider, model, decision, confidence, summary,
			supporting_reason_codes, contradicting_reason_codes, risk_flags,
			latency_ms, fallback, fallback_reason, provider_error_code,
			prompt_version, schema_version, reviewed_at
		) VALUES (
			$1::uuid,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10::jsonb,$11::jsonb,
			$12,$13,$14,$15,$16,$17,$18
		)`,
		record.SignalID, record.Pair, record.Timeframe, record.Review.Provider, record.Review.Model,
		record.Review.Decision, record.Review.Confidence, record.Review.Summary, supporting, contradicting, riskFlags,
		record.Review.LatencyMS, record.Review.Fallback, record.Review.FallbackReason,
		record.Review.ProviderErrorCode, record.Review.PromptVersion, record.Review.SchemaVersion,
		record.ReviewedAt,
	)
	return err
}
