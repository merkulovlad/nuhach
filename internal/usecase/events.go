package usecase

import (
	"context"
	"math"

	"nuhach/internal/domain"

	"go.uber.org/zap"
)

// EventUseCase handles user event operations.
type EventUseCase struct {
	userRepo          domain.UserRepository
	userEmbeddingRepo domain.UserEmbeddingRepository
	eventRepo         domain.EventRepository
	perfumeRepo       domain.PerfumeRepository
	logger            *zap.Logger
	embeddingDim      int
}

// NewEventUseCase creates a new EventUseCase.
func NewEventUseCase(
	userRepo domain.UserRepository,
	userEmbeddingRepo domain.UserEmbeddingRepository,
	eventRepo domain.EventRepository,
	perfumeRepo domain.PerfumeRepository,
	logger *zap.Logger,
	embeddingDim int,
) *EventUseCase {
	return &EventUseCase{
		userRepo:          userRepo,
		userEmbeddingRepo: userEmbeddingRepo,
		eventRepo:         eventRepo,
		perfumeRepo:       perfumeRepo,
		logger:            logger,
		embeddingDim:      embeddingDim,
	}
}

// CreateEventRequest represents the request to create an event.
type CreateEventRequest struct {
	TgID      int64
	PerfumeID int64
	EventType domain.EventType
	Rating    *int
	RequestID string
}

// CreateEvent creates a new user event and updates user embedding if needed.
func (uc *EventUseCase) CreateEvent(ctx context.Context, req CreateEventRequest) error {
	// Get or create user
	user, err := uc.userRepo.GetOrCreate(ctx, req.TgID)
	if err != nil {
		return err
	}

	// Create event
	event := &domain.UserEvent{
		UserID:    user.ID,
		TgID:      req.TgID,
		PerfumeID: req.PerfumeID,
		EventType: req.EventType,
		Rating:    req.Rating,
		RequestID: req.RequestID,
	}

	if err := uc.eventRepo.Create(ctx, event); err != nil {
		return err
	}

	// Update user embedding on positive feedback
	if req.EventType == domain.EventLike || req.EventType == domain.EventSave {
		if err := uc.updateUserEmbedding(ctx, user.ID, req.PerfumeID, req.Rating); err != nil {
			uc.logger.Warn("failed to update user embedding", zap.Error(err))
		}
	}

	return nil
}

// updateUserEmbedding updates the user's embedding incrementally.
func (uc *EventUseCase) updateUserEmbedding(ctx context.Context, userID, perfumeID int64, rating *int) error {
	// Get perfume embedding
	perfumeEmb, err := uc.perfumeRepo.GetEmbeddingByPerfumeID(ctx, perfumeID)
	if err != nil {
		return err
	}
	if perfumeEmb == nil {
		uc.logger.Warn("no embedding found for perfume", zap.Int64("perfume_id", perfumeID))
		return nil
	}

	// Get current user embedding
	userEmb, err := uc.userEmbeddingRepo.Get(ctx, userID)
	if err != nil {
		return err
	}

	// Compute weight based on rating (default weight = 1.0 for likes)
	weight := 1.0
	if rating != nil {
		weight = float64(*rating) / 5.0 // Normalize rating to 0-1 weight
	}

	if userEmb == nil || len(userEmb.Embedding) == 0 {
		// Initialize user embedding with perfume embedding
		newEmb := make([]float32, len(perfumeEmb))
		copy(newEmb, perfumeEmb)
		normalizeEmbedding(newEmb)

		return uc.userEmbeddingRepo.Upsert(ctx, &domain.UserEmbedding{
			UserID:        userID,
			Embedding:     newEmb,
			NInteractions: 1,
			Version:       1,
		})
	}

	// Incremental update with weighted running mean
	// new_emb = (old_emb * n + perfume_emb * weight) / (n + weight)
	// This gives more weight to items with higher ratings
	n := float64(userEmb.NInteractions)
	newEmb := make([]float32, len(userEmb.Embedding))
	for i := 0; i < len(newEmb); i++ {
		newEmb[i] = float32((float64(userEmb.Embedding[i])*n + float64(perfumeEmb[i])*weight) / (n + weight))
	}

	// Normalize the embedding
	normalizeEmbedding(newEmb)

	return uc.userEmbeddingRepo.Upsert(ctx, &domain.UserEmbedding{
		UserID:        userID,
		Embedding:     newEmb,
		NInteractions: userEmb.NInteractions + 1,
		Version:       userEmb.Version + 1,
	})
}

// normalizeEmbedding normalizes an embedding vector to unit length.
func normalizeEmbedding(emb []float32) {
	var norm float64
	for _, v := range emb {
		norm += float64(v) * float64(v)
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range emb {
			emb[i] = float32(float64(emb[i]) / norm)
		}
	}
}

// NormalizeEmbedding exports the normalization function for testing.
func NormalizeEmbedding(emb []float32) {
	normalizeEmbedding(emb)
}

// GetUserSaves retrieves saved perfumes for a user.
func (uc *EventUseCase) GetUserSaves(ctx context.Context, tgID int64) ([]domain.PerfumeCard, error) {
	user, err := uc.userRepo.GetByTgID(ctx, tgID)
	if err != nil || user == nil {
		return []domain.PerfumeCard{}, nil
	}

	ids, err := uc.eventRepo.GetUserSaves(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return []domain.PerfumeCard{}, nil
	}

	perfumes, err := uc.perfumeRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	cards := make([]domain.PerfumeCard, len(perfumes))
	for i, p := range perfumes {
		cards[i] = perfumeToCard(p)
	}

	return cards, nil
}
