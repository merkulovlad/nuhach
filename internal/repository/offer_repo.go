package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/merkulovlad/nuhach/internal/domain"
	"go.uber.org/zap"
)

// OfferRepo implements cached store offers and on-demand search jobs in PostgreSQL.
type OfferRepo struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewOfferRepo(db *sql.DB, logger *zap.Logger) *OfferRepo {
	return &OfferRepo{db: db, logger: logger}
}

func (r *OfferRepo) GetOffers(ctx context.Context, perfumeID int64) ([]domain.StoreOffer, error) {
	const query = `
		SELECT id, perfume_id, search_job_id, store, seller, title, price, old_price,
		       currency, volume_ml, concentration, product_type, in_stock, url,
		       rating, reviews_count, match_confidence, risk_level, risk_score,
		       comment, checked_at, expires_at
		FROM store_offers
		WHERE perfume_id = $1
		ORDER BY in_stock DESC, price ASC, checked_at DESC`

	rows, err := r.db.QueryContext(ctx, query, perfumeID)
	if err != nil {
		return nil, fmt.Errorf("get store offers: %w", err)
	}
	defer rows.Close()

	offers := make([]domain.StoreOffer, 0)
	for rows.Next() {
		var offer domain.StoreOffer
		var jobID, volume, reviews sql.NullInt64
		var seller, concentration, productType, comment sql.NullString
		var oldPrice, rating sql.NullFloat64

		if err := rows.Scan(
			&offer.ID, &offer.PerfumeID, &jobID, &offer.Store, &seller,
			&offer.Title, &offer.Price, &oldPrice, &offer.Currency, &volume,
			&concentration, &productType, &offer.InStock, &offer.URL, &rating,
			&reviews, &offer.MatchConfidence, &offer.RiskLevel, &offer.RiskScore,
			&comment, &offer.CheckedAt, &offer.ExpiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan store offer: %w", err)
		}

		if jobID.Valid {
			v := jobID.Int64
			offer.SearchJobID = &v
		}
		if volume.Valid {
			v := int(volume.Int64)
			offer.VolumeML = &v
		}
		if reviews.Valid {
			v := int(reviews.Int64)
			offer.ReviewsCount = &v
		}
		if oldPrice.Valid {
			v := oldPrice.Float64
			offer.OldPrice = &v
		}
		if rating.Valid {
			v := rating.Float64
			offer.Rating = &v
		}
		offer.Seller = seller.String
		offer.Concentration = concentration.String
		offer.ProductType = productType.String
		offer.Comment = comment.String
		offers = append(offers, offer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate store offers: %w", err)
	}
	return offers, nil
}

func (r *OfferRepo) GetLatestJob(ctx context.Context, perfumeID int64) (*domain.OfferSearchJob, error) {
	const query = `
		SELECT id, perfume_id, status, requested_at, started_at, completed_at, error
		FROM offer_search_jobs
		WHERE perfume_id = $1
		ORDER BY requested_at DESC
		LIMIT 1`

	return scanOfferSearchJob(r.db.QueryRowContext(ctx, query, perfumeID))
}

func (r *OfferRepo) CreateSearchJob(ctx context.Context, perfumeID int64) (*domain.OfferSearchJob, error) {
	const insert = `
		INSERT INTO offer_search_jobs (perfume_id, status)
		VALUES ($1, 'queued')
		ON CONFLICT (perfume_id) WHERE status IN ('queued', 'running') DO NOTHING
		RETURNING id, perfume_id, status, requested_at, started_at, completed_at, error`

	job, err := scanOfferSearchJob(r.db.QueryRowContext(ctx, insert, perfumeID))
	if err == nil {
		return job, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("create offer search job: %w", err)
	}

	const active = `
		SELECT id, perfume_id, status, requested_at, started_at, completed_at, error
		FROM offer_search_jobs
		WHERE perfume_id = $1 AND status IN ('queued', 'running')
		ORDER BY requested_at DESC
		LIMIT 1`
	job, err = scanOfferSearchJob(r.db.QueryRowContext(ctx, active, perfumeID))
	if err != nil {
		return nil, fmt.Errorf("get active offer search job: %w", err)
	}
	return job, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOfferSearchJob(row rowScanner) (*domain.OfferSearchJob, error) {
	var job domain.OfferSearchJob
	var startedAt, completedAt sql.NullTime
	var jobError sql.NullString
	if err := row.Scan(
		&job.ID, &job.PerfumeID, &job.Status, &job.RequestedAt,
		&startedAt, &completedAt, &jobError,
	); err != nil {
		return nil, err
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	job.Error = jobError.String
	return &job, nil
}
