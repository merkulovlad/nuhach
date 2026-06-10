package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/merkulovlad/nuhach/internal/domain"
)

// ErrPerfumeNotFound indicates that an offer search target does not exist.
var ErrPerfumeNotFound = errors.New("perfume not found")

// OfferUseCase coordinates cached results and demand-driven scraper jobs.
type OfferUseCase struct {
	perfumeRepo domain.PerfumeRepository
	offerRepo   domain.OfferRepository
	now         func() time.Time
	negativeTTL time.Duration
	failureTTL  time.Duration
}

// NewOfferUseCase creates an offer search use case.
func NewOfferUseCase(perfumeRepo domain.PerfumeRepository, offerRepo domain.OfferRepository) *OfferUseCase {
	return &OfferUseCase{
		perfumeRepo: perfumeRepo,
		offerRepo:   offerRepo,
		now:         time.Now,
		negativeTTL: 2 * time.Hour,
		failureTTL:  15 * time.Minute,
	}
}

// Get returns cached offers and the latest search status.
func (uc *OfferUseCase) Get(ctx context.Context, perfumeID int64) (*domain.OfferSearchResult, error) {
	if err := uc.ensurePerfume(ctx, perfumeID); err != nil {
		return nil, err
	}

	return uc.buildResult(ctx, perfumeID)
}

// Search queues an offer refresh when needed and returns the current result.
func (uc *OfferUseCase) Search(ctx context.Context, perfumeID int64, force bool) (*domain.OfferSearchResult, error) {
	if err := uc.ensurePerfume(ctx, perfumeID); err != nil {
		return nil, err
	}

	result, err := uc.buildResult(ctx, perfumeID)
	if err != nil {
		return nil, err
	}

	if !force && (result.Status == "ready" || result.Status == "failed") {
		return result, nil
	}

	job, err := uc.offerRepo.CreateSearchJob(ctx, perfumeID)
	if err != nil {
		return nil, fmt.Errorf("queue offer search: %w", err)
	}

	result.JobID = &job.ID
	if len(result.Offers) > 0 {
		result.Status = "refreshing"
	} else {
		result.Status = "searching"
	}

	result.Error = ""

	return result, nil
}

func (uc *OfferUseCase) ensurePerfume(ctx context.Context, perfumeID int64) error {
	perfume, err := uc.perfumeRepo.GetByID(ctx, perfumeID)
	if err != nil {
		return fmt.Errorf("get perfume: %w", err)
	}

	if perfume == nil {
		return ErrPerfumeNotFound
	}

	return nil
}

func (uc *OfferUseCase) buildResult(ctx context.Context, perfumeID int64) (*domain.OfferSearchResult, error) {
	offers, err := uc.offerRepo.GetOffers(ctx, perfumeID)
	if err != nil {
		return nil, err
	}

	job, err := uc.offerRepo.GetLatestJob(ctx, perfumeID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	result := &domain.OfferSearchResult{PerfumeID: perfumeID, Status: "empty", Offers: offers}
	if len(offers) > 0 {
		latestChecked := offers[0].CheckedAt
		latestExpiry := offers[0].ExpiresAt
		fresh := false

		for _, offer := range offers {
			if offer.CheckedAt.After(latestChecked) {
				latestChecked = offer.CheckedAt
			}

			if offer.ExpiresAt.After(latestExpiry) {
				latestExpiry = offer.ExpiresAt
			}

			if offer.ExpiresAt.After(uc.now()) {
				fresh = true
			}
		}

		result.RefreshedAt = &latestChecked

		result.ExpiresAt = &latestExpiry
		if fresh {
			result.Status = "ready"
		} else {
			result.Status = "stale"
		}
	}

	if job != nil {
		result.JobID = &job.ID
		switch job.Status {
		case "queued", "running":
			if len(offers) > 0 {
				result.Status = "refreshing"
			} else {
				result.Status = "searching"
			}
		case "completed":
			if len(offers) == 0 && job.CompletedAt != nil {
				expiresAt := job.CompletedAt.Add(uc.negativeTTL)
				if expiresAt.After(uc.now()) {
					result.Status = "ready"
					result.RefreshedAt = job.CompletedAt
					result.ExpiresAt = &expiresAt
				}
			}
		case "failed":
			if len(offers) == 0 && job.CompletedAt != nil {
				expiresAt := job.CompletedAt.Add(uc.failureTTL)
				if expiresAt.After(uc.now()) {
					result.Status = "failed"
					result.Error = job.Error
					result.RefreshedAt = job.CompletedAt
					result.ExpiresAt = &expiresAt
				}
			}
		}
	}

	return result, nil
}
