// Package usecase contains application business logic.
package usecase

import (
	"context"

	"nuhach/internal/domain"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SearchUseCase handles search operations.
type SearchUseCase struct {
	searchRepo  domain.SearchRepository
	perfumeRepo domain.PerfumeRepository
	eventRepo   domain.EventRepository
	userRepo    domain.UserRepository
	logger      *zap.Logger
}

// NewSearchUseCase creates a new SearchUseCase.
func NewSearchUseCase(
	searchRepo domain.SearchRepository,
	perfumeRepo domain.PerfumeRepository,
	eventRepo domain.EventRepository,
	userRepo domain.UserRepository,
	logger *zap.Logger,
) *SearchUseCase {
	return &SearchUseCase{
		searchRepo:  searchRepo,
		perfumeRepo: perfumeRepo,
		eventRepo:   eventRepo,
		userRepo:    userRepo,
		logger:      logger,
	}
}

// Search performs a search and logs impressions.
// Uses OpenSearch BM25 as primary, falls back to PostgreSQL full-text search if no results.
func (uc *SearchUseCase) Search(ctx context.Context, query string, limit, offset int, tgID *int64) (*domain.SearchResult, error) {
	// Perform search using OpenSearch
	cards, total, err := uc.searchRepo.Search(ctx, query, limit, offset)
	if err != nil {
		uc.logger.Warn("OpenSearch failed, falling back to PostgreSQL full-text search",
			zap.String("query", query),
			zap.Error(err))
		// Fallback to PostgreSQL full-text search
		cards, total, err = uc.perfumeRepo.FullTextSearch(ctx, query, limit, offset)
		if err != nil {
			return nil, err
		}
	}

	// If OpenSearch returned no results, try PostgreSQL full-text search
	if len(cards) == 0 {
		uc.logger.Info("OpenSearch returned no results, trying PostgreSQL full-text search",
			zap.String("query", query))
		cards, total, err = uc.perfumeRepo.FullTextSearch(ctx, query, limit, offset)
		if err != nil {
			uc.logger.Warn("PostgreSQL full-text search also failed", zap.Error(err))
		}
	}

	// Generate request ID
	requestID := uuid.New().String()

	// Log impressions if user is provided
	if tgID != nil && len(cards) > 0 {
		user, err := uc.userRepo.GetOrCreate(ctx, *tgID)
		if err != nil {
			uc.logger.Warn("failed to get/create user for impression logging", zap.Error(err))
		} else {
			ids := make([]int64, len(cards))
			ranks := make([]int, len(cards))
			for i, card := range cards {
				ids[i] = card.ID
				ranks[i] = offset + i + 1
			}
			err = uc.eventRepo.CreateImpressionLog(ctx, &domain.ImpressionLog{
				UserID:     user.ID,
				RequestID:  requestID,
				Surface:    "search",
				PerfumeIDs: ids,
				Ranks:      ranks,
			})
			if err != nil {
				uc.logger.Warn("failed to create impression log", zap.Error(err))
			}
		}
	}

	return &domain.SearchResult{
		Items:     cards,
		RequestID: requestID,
		Total:     total,
	}, nil
}

// GetPerfumeByID retrieves a perfume by ID.
func (uc *SearchUseCase) GetPerfumeByID(ctx context.Context, id int64) (*domain.Perfume, error) {
	return uc.perfumeRepo.GetByID(ctx, id)
}

// GetSimilarPerfumes retrieves similar perfumes using pgvector.
func (uc *SearchUseCase) GetSimilarPerfumes(ctx context.Context, perfumeID int64, limit int, tgID *int64) (*domain.SearchResult, error) {
	var excludeIDs []int64
	var userID int64

	// Get excluded IDs if user is provided
	if tgID != nil {
		user, err := uc.userRepo.GetOrCreate(ctx, *tgID)
		if err == nil {
			userID = user.ID
			excludeIDs, _ = uc.eventRepo.GetUserInteractedPerfumes(ctx, user.ID, []domain.EventType{
				domain.EventLike,
				domain.EventDislike,
			})
		}
	}

	// Exclude the source perfume
	excludeIDs = append(excludeIDs, perfumeID)

	// Get similar perfumes
	perfumes, err := uc.perfumeRepo.GetSimilar(ctx, perfumeID, limit, excludeIDs)
	if err != nil {
		return nil, err
	}

	// Convert to cards
	cards := make([]domain.PerfumeCard, len(perfumes))
	ids := make([]int64, len(perfumes))
	ranks := make([]int, len(perfumes))
	for i, p := range perfumes {
		cards[i] = perfumeToCard(p.Perfume)
		ids[i] = p.ID
		ranks[i] = i + 1
	}

	// Generate request ID and log impressions
	requestID := uuid.New().String()
	if userID > 0 && len(ids) > 0 {
		err = uc.eventRepo.CreateImpressionLog(ctx, &domain.ImpressionLog{
			UserID:     userID,
			RequestID:  requestID,
			Surface:    "similar",
			PerfumeIDs: ids,
			Ranks:      ranks,
		})
		if err != nil {
			uc.logger.Warn("failed to create impression log", zap.Error(err))
		}
	}

	return &domain.SearchResult{
		Items:     cards,
		RequestID: requestID,
		Total:     int64(len(cards)),
	}, nil
}

// perfumeToCard converts a Perfume to a PerfumeCard.
func perfumeToCard(p domain.Perfume) domain.PerfumeCard {
	card := domain.PerfumeCard{
		ID:          p.ID,
		Name:        p.Name,
		RatingValue: p.RatingValue,
		RatingCount: p.RatingCount,
		Year:        p.Year,
	}
	card.Brand = p.Brand
	// Prefer Russian content for notes/accords
	if p.NotesRU != "" {
		card.Notes = p.NotesRU
	} else {
		card.Notes = p.NotesEN
	}
	if p.AccordsRU != "" {
		card.Accords = p.AccordsRU
	} else {
		card.Accords = p.AccordsEN
	}
	return card
}

// VectorSearch performs hybrid search combining semantic and BM25 results.
// Uses Reciprocal Rank Fusion (RRF) to merge rankings from both approaches.
func (uc *SearchUseCase) VectorSearch(ctx context.Context, query string, embedding []float32, limit, offset int, tgID *int64) (*domain.SearchResult, error) {
	// Fetch more candidates from each source for better fusion
	candidateLimit := limit * 3

	// 1. Get vector search results
	vectorCards, _, _ := uc.perfumeRepo.VectorSearchByEmbedding(ctx, embedding, candidateLimit, 0)

	// 2. Get BM25/text search results
	textCards, _, _ := uc.searchRepo.Search(ctx, query, candidateLimit, 0)

	// 3. If BM25 fails, try PostgreSQL full-text
	if len(textCards) == 0 {
		textCards, _, _ = uc.perfumeRepo.FullTextSearch(ctx, query, candidateLimit, 0)
	}

	// 4. Merge results using RRF
	cards := uc.mergeWithRRF(vectorCards, textCards, limit, offset)

	// If still no results, return empty
	if len(cards) == 0 {
		return &domain.SearchResult{
			Items:     []domain.PerfumeCard{},
			RequestID: uuid.New().String(),
			Total:     0,
		}, nil
	}

	// Generate request ID
	requestID := uuid.New().String()

	// Log impressions if user is provided
	if tgID != nil && len(cards) > 0 {
		user, err := uc.userRepo.GetOrCreate(ctx, *tgID)
		if err != nil {
			uc.logger.Warn("failed to get/create user for impression logging", zap.Error(err))
		} else {
			ids := make([]int64, len(cards))
			ranks := make([]int, len(cards))
			for i, card := range cards {
				ids[i] = card.ID
				ranks[i] = offset + i + 1
			}
			err = uc.eventRepo.CreateImpressionLog(ctx, &domain.ImpressionLog{
				UserID:     user.ID,
				RequestID:  requestID,
				Surface:    "hybrid_search",
				PerfumeIDs: ids,
				Ranks:      ranks,
			})
			if err != nil {
				uc.logger.Warn("failed to create impression log", zap.Error(err))
			}
		}
	}

	return &domain.SearchResult{
		Items:     cards,
		RequestID: requestID,
		Total:     int64(len(cards)),
	}, nil
}

// mergeWithRRF combines results using Reciprocal Rank Fusion.
// RRF score = sum(1 / (k + rank)) where k=60 is standard constant.
func (uc *SearchUseCase) mergeWithRRF(vectorCards, textCards []domain.PerfumeCard, limit, offset int) []domain.PerfumeCard {
	const k = 60.0

	// Calculate RRF scores
	scores := make(map[int64]float64)
	cardMap := make(map[int64]domain.PerfumeCard)

	// Add vector search scores
	for rank, card := range vectorCards {
		scores[card.ID] += 1.0 / (k + float64(rank+1))
		cardMap[card.ID] = card
	}

	// Add text search scores
	for rank, card := range textCards {
		scores[card.ID] += 1.0 / (k + float64(rank+1))
		if _, exists := cardMap[card.ID]; !exists {
			cardMap[card.ID] = card
		}
	}

	// Sort by RRF score
	type scoredCard struct {
		id    int64
		score float64
	}
	sorted := make([]scoredCard, 0, len(scores))
	for id, score := range scores {
		sorted = append(sorted, scoredCard{id: id, score: score})
	}

	// Sort descending by score
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].score > sorted[i].score {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Apply offset and limit
	start := offset
	if start >= len(sorted) {
		return []domain.PerfumeCard{}
	}
	end := start + limit
	if end > len(sorted) {
		end = len(sorted)
	}

	// Build result
	result := make([]domain.PerfumeCard, 0, end-start)
	for i := start; i < end; i++ {
		result = append(result, cardMap[sorted[i].id])
	}

	return result
}
