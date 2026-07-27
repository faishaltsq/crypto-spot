package quality

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository persists quality reports for historical tracking.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new quality repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// SaveReport persists a quality report to the data_quality_history table.
func (r *Repository) SaveReport(ctx context.Context, report QualityReport) error {
	reasonsJSON, _ := json.Marshal(report.Reasons)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO data_quality_history (
			symbol, score, status, reasons, signal_allowed, evaluated_at
		) VALUES ($1, $2, $3, $4::jsonb, $5, $6)
		ON CONFLICT (symbol, evaluated_at) DO UPDATE SET
			score = EXCLUDED.score,
			status = EXCLUDED.status,
			reasons = EXCLUDED.reasons,
			signal_allowed = EXCLUDED.signal_allowed
	`,
		report.Symbol,
		report.Score,
		string(report.Status),
		reasonsJSON,
		report.SignalAllowed,
		report.EvaluatedAt,
	)
	return err
}

// SaveReports persists multiple quality reports in a batch.
func (r *Repository) SaveReports(ctx context.Context, reports map[string]QualityReport) error {
	for _, report := range reports {
		if err := r.SaveReport(ctx, report); err != nil {
			return err
		}
	}
	return nil
}
