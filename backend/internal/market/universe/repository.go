package universe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) UpsertPairs(ctx context.Context, pairs []RankedPair) error {
	if len(pairs) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// First, mark all existing pairs as inactive
	_, err = tx.Exec(ctx, `UPDATE market_pairs SET is_active = FALSE`)
	if err != nil {
		return fmt.Errorf("deactivate existing pairs: %w", err)
	}

	updatedAt := time.Now()

	for _, p := range pairs {
		selReason, _ := json.Marshal(p.SelectionReason)
		_, err := tx.Exec(ctx, `
			INSERT INTO market_pairs (
				symbol, rank_position, rank_score, tier, qualified, 
				quote_volume_24h, spread_bps, depth_score, activity_score, 
				selection_reason, rejection_reason, is_active, universe_updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
			)
			ON CONFLICT (symbol) DO UPDATE SET
				rank_position = EXCLUDED.rank_position,
				rank_score = EXCLUDED.rank_score,
				tier = EXCLUDED.tier,
				qualified = EXCLUDED.qualified,
				quote_volume_24h = EXCLUDED.quote_volume_24h,
				spread_bps = EXCLUDED.spread_bps,
				depth_score = EXCLUDED.depth_score,
				activity_score = EXCLUDED.activity_score,
				selection_reason = EXCLUDED.selection_reason,
				rejection_reason = EXCLUDED.rejection_reason,
				is_active = EXCLUDED.is_active,
				universe_updated_at = EXCLUDED.universe_updated_at
		`,
			p.Symbol, p.Rank, p.RankScore, p.Tier, p.Qualified,
			p.QuoteVolume24h, p.SpreadBps, p.DepthScore, p.ActivityScore,
			selReason, p.RejectionReason, true, updatedAt,
		)
		if err != nil {
			return fmt.Errorf("upsert pair %s: %w", p.Symbol, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetActivePairs(ctx context.Context) ([]RankedPair, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT 
			symbol, rank_position, rank_score, tier, qualified,
			quote_volume_24h, spread_bps, depth_score, activity_score,
			selection_reason, rejection_reason
		FROM market_pairs
		WHERE is_active = TRUE
		ORDER BY rank_position ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query active pairs: %w", err)
	}
	defer rows.Close()

	var pairs []RankedPair
	for rows.Next() {
		var p RankedPair
		var selReason []byte
		if err := rows.Scan(
			&p.Symbol, &p.Rank, &p.RankScore, &p.Tier, &p.Qualified,
			&p.QuoteVolume24h, &p.SpreadBps, &p.DepthScore, &p.ActivityScore,
			&selReason, &p.RejectionReason,
		); err != nil {
			return nil, fmt.Errorf("scan pair: %w", err)
		}
		if len(selReason) > 0 {
			_ = json.Unmarshal(selReason, &p.SelectionReason)
		}
		pairs = append(pairs, p)
	}
	return pairs, nil
}
