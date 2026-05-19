package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/merkulovlad/nuhach/internal/domain"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// EventRepo implements domain.EventRepository using PostgreSQL.
type EventRepo struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewEventRepo creates a new EventRepo.
func NewEventRepo(db *sql.DB, logger *zap.Logger) *EventRepo {
	return &EventRepo{db: db, logger: logger}
}

// Create stores a new user event.
func (r *EventRepo) Create(ctx context.Context, event *domain.UserEvent) error {
	now := time.Now()

	// Map rating to interaction_score (1-5 -> 0.2-1.0)
	var score float64 = 1.0
	if event.Rating != nil {
		score = float64(*event.Rating) / 5.0
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_events (user_id, tg_id, perfume_id, event_type, rating, request_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, event.UserID, event.TgID, event.PerfumeID, string(event.EventType), event.Rating, event.RequestID, now)
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	// Also insert into user_interactions for compatibility with existing schema
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO user_interactions (user_id, perfume_id, interaction_type, interaction_score, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, perfume_id, interaction_type) DO UPDATE SET
			interaction_score = $4,
			created_at = $5
	`, event.UserID, event.PerfumeID, string(event.EventType), score, now)
	if err != nil {
		r.logger.Warn("failed to create user_interaction record", zap.Error(err))
	}

	return nil
}

// GetUserInteractedPerfumes retrieves perfume IDs the user has interacted with.
func (r *EventRepo) GetUserInteractedPerfumes(ctx context.Context, userID int64, eventTypes []domain.EventType) ([]int64, error) {
	if len(eventTypes) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(eventTypes))
	args := make([]interface{}, len(eventTypes)+1)
	args[0] = userID
	for i, et := range eventTypes {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = string(et)
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT perfume_id
		FROM user_events
		WHERE user_id = $1 AND event_type IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get interacted perfumes: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan perfume id: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// GetUserLikedPerfumes retrieves perfumes the user has liked (for embedding updates).
func (r *EventRepo) GetUserLikedPerfumes(ctx context.Context, userID int64, limit int) ([]domain.UserEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, tg_id, perfume_id, event_type, rating, request_id, created_at
		FROM user_events
		WHERE user_id = $1 AND event_type IN ('like', 'save')
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get liked perfumes: %w", err)
	}
	defer rows.Close()

	var events []domain.UserEvent
	for rows.Next() {
		var e domain.UserEvent
		var eventType string
		var rating sql.NullInt64
		var requestID sql.NullString

		if err := rows.Scan(&e.ID, &e.UserID, &e.TgID, &e.PerfumeID, &eventType, &rating, &requestID, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		e.EventType = domain.EventType(eventType)
		if rating.Valid {
			r := int(rating.Int64)
			e.Rating = &r
		}
		if requestID.Valid {
			e.RequestID = requestID.String
		}

		events = append(events, e)
	}

	return events, nil
}

// CreateImpressionLog stores an impression log for metrics.
func (r *EventRepo) CreateImpressionLog(ctx context.Context, log *domain.ImpressionLog) error {
	now := time.Now()

	// Convert slices to PostgreSQL arrays
	perfumeIDsStr := fmt.Sprintf("{%s}", int64SliceToString(log.PerfumeIDs))
	ranksStr := fmt.Sprintf("{%s}", intSliceToString(log.Ranks))

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO impression_logs (user_id, request_id, surface, perfume_ids, ranks, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, log.UserID, log.RequestID, log.Surface, perfumeIDsStr, ranksStr, now)
	if err != nil {
		return fmt.Errorf("failed to create impression log: %w", err)
	}

	return nil
}

// GetUserSaves retrieves saved perfumes for a user.
func (r *EventRepo) GetUserSaves(ctx context.Context, userID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT perfume_id
		FROM user_events
		WHERE user_id = $1 AND event_type = 'save'
		ORDER BY perfume_id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user saves: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan perfume id: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func int64SliceToString(s []int64) string {
	parts := make([]string, len(s))
	for i, v := range s {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, ",")
}

func intSliceToString(s []int) string {
	parts := make([]string, len(s))
	for i, v := range s {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, ",")
}

// Ensure pgtype is used (for future use)
var _ pgtype.Text
