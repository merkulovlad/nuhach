// Package domain contains core business entities and repository interfaces.
package domain

import "time"

// Perfume represents a perfume entity.
type Perfume struct {
	ID          int64    `json:"id"`
	URL         string   `json:"url,omitempty"`
	Name        string   `json:"name"`
	Brand       string   `json:"brand"`
	Gender      string   `json:"gender,omitempty"`
	GenderRU    string   `json:"gender_ru,omitempty"`
	RatingValue *float64 `json:"rating_value,omitempty"`
	RatingCount *int     `json:"rating_count,omitempty"`
	Year        *int     `json:"year,omitempty"`
	NotesEN     string   `json:"notes_en,omitempty"`
	NotesRU     string   `json:"notes_ru,omitempty"`
	AccordsEN   string   `json:"accords_en,omitempty"`
	AccordsRU   string   `json:"accords_ru,omitempty"`
	Perfumers   string   `json:"perfumers,omitempty"`
}

// PerfumeCard is a compact representation for Telegram-friendly responses.
type PerfumeCard struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Brand       string   `json:"brand"`
	RatingValue *float64 `json:"rating_value,omitempty"`
	RatingCount *int     `json:"rating_count,omitempty"`
	Year        *int     `json:"year,omitempty"`
	Notes       string   `json:"notes,omitempty"`
	Accords     string   `json:"accords,omitempty"`
}

// PerfumeWithEmbedding represents a perfume with its embedding for recommendations.
type PerfumeWithEmbedding struct {
	Perfume
	Embedding []float32 `json:"-"`
}

// User represents a Telegram user.
type User struct {
	ID        int64     `json:"id"`
	TgID      int64     `json:"tg_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserEmbedding stores the user's preference embedding.
type UserEmbedding struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Embedding     []float32 `json:"embedding"`
	NInteractions int       `json:"n_interactions"`
	Version       int       `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// EventType represents types of user events.
type EventType string

const (
	EventImpression EventType = "impression"
	EventClick      EventType = "click"
	EventLike       EventType = "like"
	EventDislike    EventType = "dislike"
	EventSave       EventType = "save"
	EventMySaves    EventType = "my_saves"
)

// UserEvent represents a user interaction event.
type UserEvent struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	TgID      int64     `json:"tg_id"`
	PerfumeID int64     `json:"perfume_id"`
	EventType EventType `json:"event_type"`
	Rating    *int      `json:"rating,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ImpressionLog stores impression lists for metrics computation.
type ImpressionLog struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	RequestID  string    `json:"request_id"`
	Surface    string    `json:"surface"` // "search" or "recommendations"
	PerfumeIDs []int64   `json:"perfume_ids"`
	Ranks      []int     `json:"ranks"`
	CreatedAt  time.Time `json:"created_at"`
}

// SearchResult contains search results with request tracking.
type SearchResult struct {
	Items     []PerfumeCard `json:"items"`
	RequestID string        `json:"request_id"`
	Total     int64         `json:"total"`
}

// RecommendationResult contains recommendation results with request tracking.
type RecommendationResult struct {
	Items          []PerfumeCard `json:"items"`
	RequestID      string        `json:"request_id"`
	ExplorationIDs []int64       `json:"exploration_ids,omitempty"`
}

// AnalyticsMetrics contains computed analytics metrics.
type AnalyticsMetrics struct {
	Date        string  `json:"date"`
	Surface     string  `json:"surface"`
	CTR         float64 `json:"ctr"`
	PrecisionK  float64 `json:"precision_at_k"`
	RecallK     float64 `json:"recall_at_k"`
	MAPK        float64 `json:"map_at_k"`
	NDCGK       float64 `json:"ndcg_at_k"`
	Coverage    float64 `json:"coverage"`
	Diversity   float64 `json:"diversity"`
	Novelty     float64 `json:"novelty"`
	Impressions int64   `json:"impressions"`
	Clicks      int64   `json:"clicks"`
}
