package usecase

import (
	"context"
	"math"
	"math/rand"
	"sort"

	"github.com/merkulovlad/nuhach/internal/domain"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RecommendationUseCase handles recommendation logic.
type RecommendationUseCase struct {
	perfumeRepo       domain.PerfumeRepository
	userRepo          domain.UserRepository
	userEmbeddingRepo domain.UserEmbeddingRepository
	eventRepo         domain.EventRepository
	logger            *zap.Logger

	// Algorithm parameters
	bayesianM       float64
	explorationRate float64
	embeddingDim    int
	candidateLimit  int
	maxPerBrand     int
}

// NewRecommendationUseCase creates a new RecommendationUseCase.
func NewRecommendationUseCase(
	perfumeRepo domain.PerfumeRepository,
	userRepo domain.UserRepository,
	userEmbeddingRepo domain.UserEmbeddingRepository,
	eventRepo domain.EventRepository,
	logger *zap.Logger,
	bayesianM float64,
	explorationRate float64,
	embeddingDim int,
	candidateLimit int,
	maxPerBrand int,
) *RecommendationUseCase {
	return &RecommendationUseCase{
		perfumeRepo:       perfumeRepo,
		userRepo:          userRepo,
		userEmbeddingRepo: userEmbeddingRepo,
		eventRepo:         eventRepo,
		logger:            logger,
		bayesianM:         bayesianM,
		explorationRate:   explorationRate,
		embeddingDim:      embeddingDim,
		candidateLimit:    candidateLimit,
		maxPerBrand:       maxPerBrand,
	}
}

// scoredCandidate holds a candidate with its computed score.
type scoredCandidate struct {
	perfume        domain.PerfumeWithEmbedding
	similarity     float64
	weightedRating float64
	finalScore     float64
	isExploration  bool
}

// GetRecommendations generates personalized recommendations for a user.
func (uc *RecommendationUseCase) GetRecommendations(ctx context.Context, tgID int64, limit int) (*domain.RecommendationResult, error) {
	// Get or create user
	user, err := uc.userRepo.GetOrCreate(ctx, tgID)
	if err != nil {
		return nil, err
	}

	// Get user's interacted perfumes to exclude
	excludeIDs, err := uc.eventRepo.GetUserInteractedPerfumes(ctx, user.ID, []domain.EventType{
		domain.EventLike,
		domain.EventDislike,
		domain.EventSave,
	})
	if err != nil {
		uc.logger.Warn("failed to get interacted perfumes", zap.Error(err))
	}

	// Get user embedding
	userEmb, err := uc.userEmbeddingRepo.Get(ctx, user.ID)
	if err != nil {
		uc.logger.Warn("failed to get user embedding", zap.Error(err))
	}

	var candidates []domain.PerfumeWithEmbedding

	if userEmb != nil && len(userEmb.Embedding) > 0 {
		// Use user embedding for kNN candidate generation
		candidates, err = uc.perfumeRepo.GetCandidatesForUser(ctx, userEmb.Embedding, uc.candidateLimit, excludeIDs)
		if err != nil {
			return nil, err
		}
	} else {
		// Cold start: get popular perfumes
		uc.logger.Info("Cold start recommendation for user", zap.Int64("tg_id", tgID))
		candidates, err = uc.getColdStartCandidates(ctx, limit*2, excludeIDs)
		if err != nil {
			return nil, err
		}
	}

	if len(candidates) == 0 {
		return &domain.RecommendationResult{
			Items:     []domain.PerfumeCard{},
			RequestID: uuid.New().String(),
		}, nil
	}

	// Get global stats for Bayesian weighted rating
	globalMean, _, err := uc.perfumeRepo.GetGlobalStats(ctx)
	if err != nil {
		globalMean = 3.5 // Default if stats unavailable
		uc.logger.Warn("using default global mean", zap.Error(err))
	}

	// Score and rank candidates
	scored := uc.scoreAndRank(candidates, userEmb, globalMean)

	// Diversify by brand before exploration so the bubble doesn't survive
	// into the final list. With maxPerBrand=0 this is a no-op.
	scored = applyBrandCap(scored, uc.maxPerBrand)

	// Apply exploration
	result := uc.applyExploration(scored, limit)

	// Convert to cards and track exploration items
	cards := make([]domain.PerfumeCard, len(result))
	ids := make([]int64, len(result))
	ranks := make([]int, len(result))
	var explorationIDs []int64

	for i, sc := range result {
		cards[i] = perfumeToCard(sc.perfume.Perfume)
		ids[i] = sc.perfume.ID
		ranks[i] = i + 1
		if sc.isExploration {
			explorationIDs = append(explorationIDs, sc.perfume.ID)
		}
	}

	// Generate request ID and log impressions
	requestID := uuid.New().String()
	err = uc.eventRepo.CreateImpressionLog(ctx, &domain.ImpressionLog{
		UserID:     user.ID,
		RequestID:  requestID,
		Surface:    "recommendations",
		PerfumeIDs: ids,
		Ranks:      ranks,
	})
	if err != nil {
		uc.logger.Warn("failed to create impression log", zap.Error(err))
	}

	return &domain.RecommendationResult{
		Items:          cards,
		RequestID:      requestID,
		ExplorationIDs: explorationIDs,
	}, nil
}

// scoreAndRank scores candidates and sorts by final score.
func (uc *RecommendationUseCase) scoreAndRank(candidates []domain.PerfumeWithEmbedding, userEmb *domain.UserEmbedding, globalMean float64) []scoredCandidate {
	scored := make([]scoredCandidate, len(candidates))

	for i, c := range candidates {
		sc := scoredCandidate{
			perfume: c,
		}

		// Compute similarity (already computed during kNN, using placeholder)
		if userEmb != nil && len(c.Embedding) > 0 {
			sc.similarity = cosineSimilarity(userEmb.Embedding, c.Embedding)
		} else {
			sc.similarity = 0.5 // Neutral for cold start
		}

		// Compute Bayesian weighted rating
		sc.weightedRating = uc.computeWeightedRating(c.RatingValue, c.RatingCount, globalMean)

		// Combine scores: primary is similarity, secondary is weighted rating
		// Formula: 0.7 * similarity + 0.3 * normalized_weighted_rating
		normalizedRating := (sc.weightedRating - 1) / 4 // Normalize 1-5 to 0-1
		if normalizedRating < 0 {
			normalizedRating = 0
		}
		if normalizedRating > 1 {
			normalizedRating = 1
		}

		sc.finalScore = 0.7*sc.similarity + 0.3*normalizedRating

		scored[i] = sc
	}

	// Sort by final score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].finalScore > scored[j].finalScore
	})

	return scored
}

// computeWeightedRating computes Bayesian weighted rating.
// Formula: wr = (v/(v+m))*R + (m/(v+m))*C
// Where: v = rating_count, R = rating_value, m = threshold, C = global mean
func (uc *RecommendationUseCase) computeWeightedRating(ratingValue *float64, ratingCount *int, globalMean float64) float64 {
	// If missing data, return global mean (neutral)
	if ratingValue == nil || ratingCount == nil {
		return globalMean
	}

	v := float64(*ratingCount)
	R := *ratingValue
	m := uc.bayesianM
	C := globalMean

	// Weighted rating formula
	return (v/(v+m))*R + (m/(v+m))*C
}

// ComputeWeightedRatingExported exports the weighted rating computation for testing.
func ComputeWeightedRating(ratingValue *float64, ratingCount *int, globalMean, bayesianM float64) float64 {
	if ratingValue == nil || ratingCount == nil {
		return globalMean
	}
	v := float64(*ratingCount)
	R := *ratingValue
	m := bayesianM
	C := globalMean
	return (v/(v+m))*R + (m/(v+m))*C
}

// brandCapOrder returns the indices of items in input order, with items
// from over-represented brands shoved to the tail. The number of items
// kept from any one brand is capped at maxPerBrand. Empty brand strings
// count as their own bucket per-item (we don't want unknown-brand items
// to collapse together). maxPerBrand <= 0 disables the cap.
func brandCapOrder(brands []string, maxPerBrand int) []int {
	if maxPerBrand <= 0 {
		out := make([]int, len(brands))
		for i := range brands {
			out[i] = i
		}
		return out
	}
	head := make([]int, 0, len(brands))
	tail := make([]int, 0)
	counts := make(map[string]int, len(brands))
	for i, b := range brands {
		if b == "" {
			head = append(head, i)
			continue
		}
		if counts[b] >= maxPerBrand {
			tail = append(tail, i)
			continue
		}
		counts[b]++
		head = append(head, i)
	}
	return append(head, tail...)
}

// BrandCapOrder exports brandCapOrder for testing.
func BrandCapOrder(brands []string, maxPerBrand int) []int {
	return brandCapOrder(brands, maxPerBrand)
}

// applyBrandCap reorders scored candidates so at most maxPerBrand items
// from any single brand appear in the head of the slice; overflow items
// follow at the tail. Final truncation to `limit` happens downstream in
// applyExploration.
func applyBrandCap(scored []scoredCandidate, maxPerBrand int) []scoredCandidate {
	if maxPerBrand <= 0 || len(scored) == 0 {
		return scored
	}
	brands := make([]string, len(scored))
	for i, c := range scored {
		brands[i] = c.perfume.Brand
	}
	order := brandCapOrder(brands, maxPerBrand)
	out := make([]scoredCandidate, len(scored))
	for i, idx := range order {
		out[i] = scored[idx]
	}
	return out
}

// applyExploration applies epsilon-greedy exploration.
func (uc *RecommendationUseCase) applyExploration(scored []scoredCandidate, limit int) []scoredCandidate {
	if len(scored) <= limit {
		return scored
	}

	// Take top (1-epsilon)*limit items
	exploitCount := int(float64(limit) * (1 - uc.explorationRate))
	if exploitCount < 1 {
		exploitCount = 1
	}

	result := make([]scoredCandidate, 0, limit)
	result = append(result, scored[:exploitCount]...)

	// Fill remaining with random exploration from the rest
	remaining := scored[exploitCount:]
	exploreCount := limit - exploitCount

	if len(remaining) > 0 && exploreCount > 0 {
		// Shuffle remaining
		perm := rand.Perm(len(remaining))
		for i := 0; i < exploreCount && i < len(perm); i++ {
			candidate := remaining[perm[i]]
			candidate.isExploration = true
			result = append(result, candidate)
		}
	}

	return result
}

// getColdStartCandidates returns popular perfumes for users without interaction history.
func (uc *RecommendationUseCase) getColdStartCandidates(ctx context.Context, limit int, excludeIDs []int64) ([]domain.PerfumeWithEmbedding, error) {
	// For cold start, we'll use a zero vector to get diverse results
	// or ideally get top-rated perfumes
	zeroEmb := make([]float32, uc.embeddingDim)
	for i := range zeroEmb {
		zeroEmb[i] = 0.0
	}

	return uc.perfumeRepo.GetCandidatesForUser(ctx, zeroEmb, limit, excludeIDs)
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// CosineSimilarity exports cosine similarity for testing.
func CosineSimilarity(a, b []float32) float64 {
	return cosineSimilarity(a, b)
}
