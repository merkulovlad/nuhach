package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"nuhach/internal/domain"

	"github.com/pgvector/pgvector-go"
	"go.uber.org/zap"
)

// UserEmbeddingRepo implements domain.UserEmbeddingRepository.
type UserEmbeddingRepo struct {
	db           *sql.DB
	logger       *zap.Logger
	embeddingDim int
}

// NewUserEmbeddingRepo creates a new UserEmbeddingRepo.
func NewUserEmbeddingRepo(db *sql.DB, logger *zap.Logger, embeddingDim int) *UserEmbeddingRepo {
	return &UserEmbeddingRepo{db: db, logger: logger, embeddingDim: embeddingDim}
}

// Get retrieves the user embedding.
func (r *UserEmbeddingRepo) Get(ctx context.Context, userID int64) (*domain.UserEmbedding, error) {
	var emb domain.UserEmbedding
	var vec pgvector.Vector

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, embedding, n_interactions, version, created_at, updated_at
		FROM user_embeddings
		WHERE user_id = $1
	`, userID).Scan(&emb.ID, &emb.UserID, &vec, &emb.NInteractions, &emb.Version, &emb.CreatedAt, &emb.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user embedding: %w", err)
	}

	emb.Embedding = vec.Slice()
	if len(emb.Embedding) != r.embeddingDim {
		return nil, fmt.Errorf("user embedding dim mismatch: got %d, want %d", len(emb.Embedding), r.embeddingDim)
	}

	return &emb, nil
}

// Upsert creates or updates the user embedding.
func (r *UserEmbeddingRepo) Upsert(ctx context.Context, emb *domain.UserEmbedding) error {
	if len(emb.Embedding) != r.embeddingDim {
		return fmt.Errorf("upsert embedding dim mismatch: got %d, want %d", len(emb.Embedding), r.embeddingDim)
	}

	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_embeddings (user_id, embedding, n_interactions, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			embedding = EXCLUDED.embedding,
			n_interactions = EXCLUDED.n_interactions,
			version = user_embeddings.version + 1,
			updated_at = EXCLUDED.updated_at
	`, emb.UserID, pgvector.NewVector(emb.Embedding), emb.NInteractions, emb.Version, now)
	if err != nil {
		return fmt.Errorf("failed to upsert user embedding: %w", err)
	}

	return nil
}
