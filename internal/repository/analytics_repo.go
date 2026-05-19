package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"nuhach/internal/domain"

	"go.uber.org/zap"
)

// AnalyticsRepo implements domain.AnalyticsRepository.
type AnalyticsRepo struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewAnalyticsRepo creates a new AnalyticsRepo.
func NewAnalyticsRepo(db *sql.DB, logger *zap.Logger) *AnalyticsRepo {
	return &AnalyticsRepo{db: db, logger: logger}
}

// ComputeDailyMetrics computes metrics for a specific date and surface.
func (r *AnalyticsRepo) ComputeDailyMetrics(ctx context.Context, date string, surface string) (*domain.AnalyticsMetrics, error) {
	metrics := &domain.AnalyticsMetrics{
		Date:    date,
		Surface: surface,
	}

	// Get impressions count
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) 
		FROM impression_logs 
		WHERE DATE(created_at) = $1 AND surface = $2
	`, date, surface).Scan(&metrics.Impressions)
	if err != nil {
		return nil, fmt.Errorf("failed to count impressions: %w", err)
	}

	// Clicks must be attributed to a specific impression via request_id and
	// fall inside an attribution window after the impression. Without
	// request_id matching, clicks from any surface get counted for every
	// surface the user touched.
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM user_events ue
		JOIN impression_logs il ON il.request_id = ue.request_id
		WHERE DATE(il.created_at) = $1
		  AND il.surface = $2
		  AND ue.event_type = 'click'
		  AND ue.created_at >= il.created_at
		  AND ue.created_at <  il.created_at + INTERVAL '24 hours'
	`, date, surface).Scan(&metrics.Clicks)
	if err != nil {
		return nil, fmt.Errorf("failed to count clicks: %w", err)
	}

	// Calculate CTR
	if metrics.Impressions > 0 {
		metrics.CTR = float64(metrics.Clicks) / float64(metrics.Impressions)
	}

	// Precision@K: for each impression, fraction of its top-K shown items
	// that the same user engaged with (click/like/save) within 24h, joined
	// strictly by request_id so surfaces don't bleed into each other.
	err = r.db.QueryRowContext(ctx, `
		WITH impressions_with_hits AS (
			SELECT
				il.id,
				array_length(il.perfume_ids, 1) AS list_length,
				COALESCE((
					SELECT COUNT(DISTINCT ue.perfume_id)
					FROM user_events ue
					WHERE ue.request_id = il.request_id
					  AND ue.event_type IN ('click', 'like', 'save')
					  AND ue.perfume_id = ANY(il.perfume_ids)
					  AND ue.created_at >= il.created_at
					  AND ue.created_at <  il.created_at + INTERVAL '24 hours'
				), 0) AS hits_in_list
			FROM impression_logs il
			WHERE DATE(il.created_at) = $1 AND il.surface = $2
		)
		SELECT COALESCE(AVG(CAST(hits_in_list AS FLOAT) / NULLIF(list_length, 0)), 0)
		FROM impressions_with_hits
	`, date, surface).Scan(&metrics.PrecisionK)
	if err != nil {
		r.logger.Warn("failed to compute precision@k", zap.Error(err))
	}

	// Compute Coverage (unique items recommended / total catalog)
	err = r.db.QueryRowContext(ctx, `
		WITH unique_shown AS (
			SELECT DISTINCT unnest(perfume_ids) as perfume_id
			FROM impression_logs
			WHERE DATE(created_at) = $1 AND surface = $2
		),
		total_catalog AS (
			SELECT COUNT(*) as total FROM perfumes
		)
		SELECT 
			CAST((SELECT COUNT(*) FROM unique_shown) AS FLOAT) / 
			NULLIF((SELECT total FROM total_catalog), 0)
	`, date, surface).Scan(&metrics.Coverage)
	if err != nil {
		r.logger.Warn("failed to compute coverage", zap.Error(err))
	}

	// Compute Novelty (inverse popularity using rating_count)
	// Higher novelty = recommending items with lower rating_count
	err = r.db.QueryRowContext(ctx, `
		WITH shown_items AS (
			SELECT DISTINCT unnest(perfume_ids) as perfume_id
			FROM impression_logs
			WHERE DATE(created_at) = $1 AND surface = $2
		)
		SELECT COALESCE(AVG(1.0 / NULLIF(LOG(2 + COALESCE(p.rating_count, 0)), 0)), 0)
		FROM shown_items si
		JOIN perfumes p ON p.id = si.perfume_id
	`, date, surface).Scan(&metrics.Novelty)
	if err != nil {
		r.logger.Warn("failed to compute novelty", zap.Error(err))
	}

	return metrics, nil
}

// GetMetrics retrieves metrics for a date range.
func (r *AnalyticsRepo) GetMetrics(ctx context.Context, startDate, endDate string, surface string) ([]domain.AnalyticsMetrics, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT date, surface, ctr, precision_k, recall_k, map_k, ndcg_k, 
		       coverage, diversity, novelty, impressions, clicks
		FROM analytics_daily
		WHERE date >= $1 AND date <= $2 AND surface = $3
		ORDER BY date DESC
	`, startDate, endDate, surface)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}
	defer rows.Close()

	var metrics []domain.AnalyticsMetrics
	for rows.Next() {
		var m domain.AnalyticsMetrics
		err := rows.Scan(&m.Date, &m.Surface, &m.CTR, &m.PrecisionK, &m.RecallK, &m.MAPK, &m.NDCGK,
			&m.Coverage, &m.Diversity, &m.Novelty, &m.Impressions, &m.Clicks)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metrics: %w", err)
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// StoreDailyMetrics stores computed metrics.
func (r *AnalyticsRepo) StoreDailyMetrics(ctx context.Context, metrics *domain.AnalyticsMetrics) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO analytics_daily (date, surface, ctr, precision_k, recall_k, map_k, ndcg_k,
		                             coverage, diversity, novelty, impressions, clicks, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (date, surface) DO UPDATE SET
			ctr = $3, precision_k = $4, recall_k = $5, map_k = $6, ndcg_k = $7,
			coverage = $8, diversity = $9, novelty = $10, impressions = $11, clicks = $12,
			created_at = $13
	`, metrics.Date, metrics.Surface, metrics.CTR, metrics.PrecisionK, metrics.RecallK,
		metrics.MAPK, metrics.NDCGK, metrics.Coverage, metrics.Diversity, metrics.Novelty,
		metrics.Impressions, metrics.Clicks, time.Now())
	if err != nil {
		return fmt.Errorf("failed to store metrics: %w", err)
	}
	return nil
}
