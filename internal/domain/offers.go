package domain

import "time"

// StoreOffer is a normalized product offer collected from an allowed store.
type StoreOffer struct {
	ID              int64     `json:"id"`
	PerfumeID       int64     `json:"perfume_id"`
	SearchJobID     *int64    `json:"search_job_id,omitempty"`
	Store           string    `json:"store"`
	Seller          string    `json:"seller,omitempty"`
	Title           string    `json:"title"`
	Price           float64   `json:"price"`
	OldPrice        *float64  `json:"old_price,omitempty"`
	Currency        string    `json:"currency"`
	VolumeML        *int      `json:"volume_ml,omitempty"`
	Concentration   string    `json:"concentration,omitempty"`
	ProductType     string    `json:"product_type,omitempty"`
	InStock         bool      `json:"in_stock"`
	URL             string    `json:"url"`
	Rating          *float64  `json:"rating,omitempty"`
	ReviewsCount    *int      `json:"reviews_count,omitempty"`
	MatchConfidence float64   `json:"match_confidence"`
	RiskLevel       string    `json:"risk_level"`
	RiskScore       float64   `json:"risk_score"`
	Comment         string    `json:"comment,omitempty"`
	CheckedAt       time.Time `json:"checked_at"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// OfferSearchJob tracks an on-demand store search.
type OfferSearchJob struct {
	ID          int64      `json:"id"`
	PerfumeID   int64      `json:"perfume_id"`
	Status      string     `json:"status"`
	RequestedAt time.Time  `json:"requested_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// OfferSearchResult is returned to clients while a search is cached or running.
type OfferSearchResult struct {
	PerfumeID   int64        `json:"perfume_id"`
	Status      string       `json:"status"`
	JobID       *int64       `json:"job_id,omitempty"`
	Offers      []StoreOffer `json:"offers"`
	RefreshedAt *time.Time   `json:"refreshed_at,omitempty"`
	ExpiresAt   *time.Time   `json:"expires_at,omitempty"`
	Error       string       `json:"error,omitempty"`
}
