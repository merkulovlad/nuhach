package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"nuhach/internal/domain"

	"github.com/jackc/pgx/v5/pgtype"
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
	var embJSON []byte

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, embedding::text, n_interactions, version, created_at, updated_at
		FROM user_embeddings
		WHERE user_id = $1
	`, userID).Scan(&emb.ID, &emb.UserID, &embJSON, &emb.NInteractions, &emb.Version, &emb.CreatedAt, &emb.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user embedding: %w", err)
	}

	// Parse embedding from PostgreSQL vector format
	embStr := strings.Trim(string(embJSON), "[]")
	parts := strings.Split(embStr, ",")
	emb.Embedding = make([]float32, len(parts))
	for i, p := range parts {
		var f float64
		fmt.Sscanf(strings.TrimSpace(p), "%f", &f)
		emb.Embedding[i] = float32(f)
	}

	return &emb, nil
}

// Upsert creates or updates the user embedding.
func (r *UserEmbeddingRepo) Upsert(ctx context.Context, emb *domain.UserEmbedding) error {
	// Build embedding string for PostgreSQL vector
	embParts := make([]string, len(emb.Embedding))
	for i, v := range emb.Embedding {
		embParts[i] = fmt.Sprintf("%f", v)
	}
	embStr := "[" + strings.Join(embParts, ",") + "]"

	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_embeddings (user_id, embedding, n_interactions, version, created_at, updated_at)
		VALUES ($1, $2::vector, $3, $4, $5, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			embedding = $2::vector,
			n_interactions = $3,
			version = user_embeddings.version + 1,
			updated_at = $5
	`, emb.UserID, embStr, emb.NInteractions, emb.Version, now)
	if err != nil {
		return fmt.Errorf("failed to upsert user embedding: %w", err)
	}

	return nil
}

// Ensure pgtype is used (for future use)
var _ pgtype.Text
