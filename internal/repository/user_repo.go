package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"nuhach/internal/domain"

	"go.uber.org/zap"
)

// UserRepo implements domain.UserRepository using PostgreSQL.
type UserRepo struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(db *sql.DB, logger *zap.Logger) *UserRepo {
	return &UserRepo{db: db, logger: logger}
}

// GetOrCreate retrieves or creates a user by Telegram ID.
func (r *UserRepo) GetOrCreate(ctx context.Context, tgID int64) (*domain.User, error) {
	// Try to get existing user
	user, err := r.GetByTgID(ctx, tgID)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}

	// Create new user
	now := time.Now()
	var id int64
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO users (tg_id, created_at, updated_at)
		VALUES ($1, $2, $2)
		ON CONFLICT (tg_id) DO UPDATE SET updated_at = $2
		RETURNING id
	`, tgID, now).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &domain.User{
		ID:        id,
		TgID:      tgID,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// GetByTgID retrieves a user by Telegram ID.
func (r *UserRepo) GetByTgID(ctx context.Context, tgID int64) (*domain.User, error) {
	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tg_id, created_at, updated_at
		FROM users
		WHERE tg_id = $1
	`, tgID).Scan(&user.ID, &user.TgID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}
