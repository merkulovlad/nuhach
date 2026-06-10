// Package domain contains repository interfaces for the application.
package domain

import "context"

// OfferRepository stores demand-driven store search jobs and cached offers.
type OfferRepository interface {
	GetOffers(ctx context.Context, perfumeID int64) ([]StoreOffer, error)
	GetLatestJob(ctx context.Context, perfumeID int64) (*OfferSearchJob, error)
	CreateSearchJob(ctx context.Context, perfumeID int64) (*OfferSearchJob, error)
}

// PerfumeRepository defines operations for perfume data access.
type PerfumeRepository interface {
	// GetByID retrieves a perfume by its ID.
	GetByID(ctx context.Context, id int64) (*Perfume, error)

	// GetByIDs retrieves multiple perfumes by their IDs.
	GetByIDs(ctx context.Context, ids []int64) ([]Perfume, error)

	// GetSimilar retrieves similar perfumes using pgvector.
	GetSimilar(ctx context.Context, perfumeID int64, limit int, excludeIDs []int64) ([]PerfumeWithEmbedding, error)

	// GetCandidatesForUser retrieves recommendation candidates using user embedding.
	GetCandidatesForUser(
		ctx context.Context,
		userEmbedding []float32,
		limit int,
		excludeIDs []int64,
	) ([]PerfumeWithEmbedding, error)

	// GetEmbeddingByPerfumeID retrieves the embedding for a specific perfume.
	GetEmbeddingByPerfumeID(ctx context.Context, perfumeID int64) ([]float32, error)

	// GetGlobalStats retrieves global statistics for rating calculations.
	GetGlobalStats(ctx context.Context) (meanRating float64, totalPerfumes int64, err error)

	// GetAll retrieves all perfumes for indexing.
	GetAll(ctx context.Context) ([]Perfume, error)

	// FullTextSearch performs PostgreSQL full-text search as a fallback.
	FullTextSearch(ctx context.Context, query string, limit, offset int) ([]PerfumeCard, int64, error)

	// VectorSearchByEmbedding performs semantic search using query embedding.
	VectorSearchByEmbedding(ctx context.Context, embedding []float32, limit, offset int) ([]PerfumeCard, int64, error)
}

// SearchRepository defines operations for search functionality.
type SearchRepository interface {
	// Search performs a BM25 search with field boosts.
	Search(ctx context.Context, query string, limit, offset int) ([]PerfumeCard, int64, error)

	// VectorSearch performs semantic search using embeddings.
	VectorSearch(ctx context.Context, queryEmbedding []float32, limit, offset int) ([]PerfumeCard, int64, error)
}

// UserRepository defines operations for user data access.
type UserRepository interface {
	// GetOrCreate retrieves or creates a user by Telegram ID.
	GetOrCreate(ctx context.Context, tgID int64) (*User, error)

	// GetByTgID retrieves a user by Telegram ID.
	GetByTgID(ctx context.Context, tgID int64) (*User, error)
}

// UserEmbeddingRepository defines operations for user embeddings.
type UserEmbeddingRepository interface {
	// Get retrieves the user embedding.
	Get(ctx context.Context, userID int64) (*UserEmbedding, error)

	// Upsert creates or updates the user embedding.
	Upsert(ctx context.Context, emb *UserEmbedding) error
}

// EventRepository defines operations for event tracking.
type EventRepository interface {
	// Create stores a new user event.
	Create(ctx context.Context, event *UserEvent) error

	// GetUserInteractedPerfumes retrieves perfume IDs the user has interacted with.
	GetUserInteractedPerfumes(ctx context.Context, userID int64, eventTypes []EventType) ([]int64, error)

	// GetUserLikedPerfumes retrieves perfume IDs the user has liked (for embedding updates).
	GetUserLikedPerfumes(ctx context.Context, userID int64, limit int) ([]UserEvent, error)

	// CreateImpressionLog stores an impression log for metrics.
	CreateImpressionLog(ctx context.Context, log *ImpressionLog) error

	// GetUserSaves retrieves saved perfumes for a user.
	GetUserSaves(ctx context.Context, userID int64) ([]int64, error)
}

// AnalyticsRepository defines operations for analytics computation.
type AnalyticsRepository interface {
	// ComputeDailyMetrics computes and stores daily metrics.
	ComputeDailyMetrics(ctx context.Context, date, surface string) (*AnalyticsMetrics, error)

	// GetMetrics retrieves metrics for a date range.
	GetMetrics(ctx context.Context, startDate, endDate, surface string) ([]AnalyticsMetrics, error)

	// StoreDailyMetrics stores computed metrics.
	StoreDailyMetrics(ctx context.Context, metrics *AnalyticsMetrics) error
}

// IndexerRepository defines operations for OpenSearch indexing.
type IndexerRepository interface {
	// IndexPerfumes bulk indexes perfumes to OpenSearch.
	IndexPerfumes(ctx context.Context, perfumes []Perfume) error

	// DeleteIndex deletes the OpenSearch index.
	DeleteIndex(ctx context.Context) error

	// CreateIndex creates the OpenSearch index with proper mapping.
	CreateIndex(ctx context.Context) error
}
