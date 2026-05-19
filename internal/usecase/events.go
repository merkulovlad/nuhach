package usecase

import (
	"context"
	"math"

	"github.com/merkulovlad/nuhach/internal/domain"

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
	embeddingDecay    float64
}

// NewEventUseCase creates a new EventUseCase. embeddingDecay is the EMA
// weight applied to the prior user embedding on each update; values in
// (0,1) favor recent events, 1.0 keeps a pure running mean (no decay).
func NewEventUseCase(
	userRepo domain.UserRepository,
	userEmbeddingRepo domain.UserEmbeddingRepository,
	eventRepo domain.EventRepository,
	perfumeRepo domain.PerfumeRepository,
	logger *zap.Logger,
	embeddingDim int,
	embeddingDecay float64,
) *EventUseCase {
	if embeddingDecay <= 0 || embeddingDecay > 1 {
		embeddingDecay = 0.95
	}
	return &EventUseCase{
		userRepo:          userRepo,
		userEmbeddingRepo: userEmbeddingRepo,
		eventRepo:         eventRepo,
		perfumeRepo:       perfumeRepo,
		logger:            logger,
		embeddingDim:      embeddingDim,
		embeddingDecay:    embeddingDecay,
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

	// Move the user embedding by a signed weight per event type.
	// Likes/saves pull the centroid toward the item; dislikes push it away;
	// clicks are a weak positive signal (intent without commitment).
	if w := eventEmbeddingWeight(req.EventType, req.Rating); w != 0 {
		if err := uc.updateUserEmbedding(ctx, user.ID, req.PerfumeID, w); err != nil {
			uc.logger.Warn("failed to update user embedding", zap.Error(err))
		}
	}

	return nil
}

// eventEmbeddingWeight returns the signed weight applied to the perfume
// embedding when folding it into the user centroid. Returns 0 when the event
// type should not move the embedding (impression, my_saves, ...).
func eventEmbeddingWeight(t domain.EventType, rating *int) float64 {
	switch t {
	case domain.EventLike:
		if rating != nil {
			return float64(*rating) / 5.0
		}
		return 1.0
	case domain.EventSave:
		return 1.0
	case domain.EventClick:
		return 0.15
	case domain.EventDislike:
		return -0.4
	default:
		return 0
	}
}

// updateUserEmbedding folds a perfume embedding into the user centroid
// with a signed weight. Positive weights pull the centroid toward the
// item (like/save/click); negative weights push it away (dislike).
func (uc *EventUseCase) updateUserEmbedding(ctx context.Context, userID, perfumeID int64, weight float64) error {
	perfumeEmb, err := uc.perfumeRepo.GetEmbeddingByPerfumeID(ctx, perfumeID)
	if err != nil {
		return err
	}
	if perfumeEmb == nil {
		uc.logger.Warn("no embedding found for perfume", zap.Int64("perfume_id", perfumeID))
		return nil
	}

	userEmb, err := uc.userEmbeddingRepo.Get(ctx, userID)
	if err != nil {
		return err
	}

	// Cold start: only seed from a positive signal. A first interaction
	// being a dislike would orient the user vector backwards from a
	// disliked item, which is worse than no vector at all.
	if userEmb == nil || len(userEmb.Embedding) == 0 {
		if weight <= 0 {
			return nil
		}
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

	newEmb := mergeEmbedding(userEmb.Embedding, perfumeEmb, weight, uc.embeddingDecay)

	return uc.userEmbeddingRepo.Upsert(ctx, &domain.UserEmbedding{
		UserID:        userID,
		Embedding:     newEmb,
		NInteractions: userEmb.NInteractions + 1,
		Version:       userEmb.Version + 1,
	})
}

// mergeEmbedding folds a perfume vector into a user centroid as an
// exponential moving average: out = normalize(decay·user + (1-decay)·weight·perfume).
// Positive weight pulls toward the item, negative pushes away, decay in
// (0,1) controls how fast old preferences fade (lower = faster decay).
// decay=1 reduces to "always trust history" — no movement — so the
// caller should pass <1 in production.
func mergeEmbedding(user, perfume []float32, weight, decay float64) []float32 {
	out := make([]float32, len(user))
	novelty := 1 - decay
	for i := range out {
		out[i] = float32(decay*float64(user[i]) + novelty*weight*float64(perfume[i]))
	}
	normalizeEmbedding(out)
	return out
}

// MergeEmbedding exports mergeEmbedding for testing.
func MergeEmbedding(user, perfume []float32, weight, decay float64) []float32 {
	return mergeEmbedding(user, perfume, weight, decay)
}

// EventEmbeddingWeight exports eventEmbeddingWeight for testing.
func EventEmbeddingWeight(t domain.EventType, rating *int) float64 {
	return eventEmbeddingWeight(t, rating)
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
