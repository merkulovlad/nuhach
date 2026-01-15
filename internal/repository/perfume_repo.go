// Package repository provides concrete implementations of domain repositories.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"nuhach/internal/domain"

	"github.com/lib/pq"
	"go.uber.org/zap"
)

// parseVector parses a pgvector string format "[0.1,0.2,...]" to []float32
func parseVector(s string) ([]float32, error) {
	if s == "" {
		return nil, nil
	}
	// Remove brackets
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	result := make([]float32, len(parts))
	for i, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return nil, err
		}
		result[i] = float32(f)
	}
	return result, nil
}

// vectorToString converts []float32 to pgvector string format
func vectorToString(v []float32) string {
	if len(v) == 0 {
		return ""
	}
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = strconv.FormatFloat(float64(f), 'f', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// PerfumeRepo implements domain.PerfumeRepository using PostgreSQL.
type PerfumeRepo struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewPerfumeRepo creates a new PerfumeRepo.
func NewPerfumeRepo(db *sql.DB, logger *zap.Logger) *PerfumeRepo {
	return &PerfumeRepo{
		db:     db,
		logger: logger,
	}
}

// GetByID retrieves a perfume by its ID.
func (r *PerfumeRepo) GetByID(ctx context.Context, id int64) (*domain.Perfume, error) {
	query := `
		SELECT 
			p.id, p.url, p.perfume_name, 
			b.name as brand_en,
			p.gender,
			p.rating_value, p.rating_count, p.year
		FROM perfumes_normalized p
		LEFT JOIN brands b ON p.brand_id = b.id
		WHERE p.id = $1
	`

	perfume := &domain.Perfume{}
	var url, brandEN, gender sql.NullString
	var ratingValue sql.NullFloat64
	var ratingCount, year sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&perfume.ID, &url, &perfume.Name,
		&brandEN,
		&gender,
		&ratingValue, &ratingCount, &year,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get perfume: %w", err)
	}

	perfume.URL = url.String
	perfume.Brand = brandEN.String
	perfume.Gender = gender.String
	if ratingValue.Valid {
		perfume.RatingValue = &ratingValue.Float64
	}
	if ratingCount.Valid {
		rc := int(ratingCount.Int64)
		perfume.RatingCount = &rc
	}
	if year.Valid {
		y := int(year.Int64)
		perfume.Year = &y
	}

	// Get notes from all three note tables (top, middle, base)
	notesQuery := `
		SELECT n.name as note_en, nt.translation_ru as note_ru, 'top' as note_type
		FROM perfume_top_notes ptn
		JOIN notes n ON ptn.note_id = n.id
		LEFT JOIN note_translations nt ON nt.note_id = n.id
		WHERE ptn.perfume_id = $1
		UNION ALL
		SELECT n.name as note_en, nt.translation_ru as note_ru, 'middle' as note_type
		FROM perfume_middle_notes pmn
		JOIN notes n ON pmn.note_id = n.id
		LEFT JOIN note_translations nt ON nt.note_id = n.id
		WHERE pmn.perfume_id = $1
		UNION ALL
		SELECT n.name as note_en, nt.translation_ru as note_ru, 'base' as note_type
		FROM perfume_base_notes pbn
		JOIN notes n ON pbn.note_id = n.id
		LEFT JOIN note_translations nt ON nt.note_id = n.id
		WHERE pbn.perfume_id = $1
	`

	rows, err := r.db.QueryContext(ctx, notesQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get notes: %w", err)
	}
	defer rows.Close()

	var notesEN, notesRU []string
	for rows.Next() {
		var noteEN, noteRU sql.NullString
		var noteType string
		if err := rows.Scan(&noteEN, &noteRU, &noteType); err != nil {
			continue
		}
		if noteEN.Valid {
			notesEN = append(notesEN, noteEN.String)
		}
		if noteRU.Valid {
			notesRU = append(notesRU, noteRU.String)
		}
	}
	perfume.NotesEN = strings.Join(notesEN, ", ")
	perfume.NotesRU = strings.Join(notesRU, ", ")

	// Get accords
	accordsQuery := `
		SELECT 
			a.name as accord_en, at.translation_ru as accord_ru
		FROM perfume_accords pa
		JOIN accords a ON pa.accord_id = a.id
		LEFT JOIN accord_translations at ON at.accord_id = a.id
		WHERE pa.perfume_id = $1
	`

	rows, err = r.db.QueryContext(ctx, accordsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get accords: %w", err)
	}
	defer rows.Close()

	var accordsEN, accordsRU []string
	for rows.Next() {
		var accordEN, accordRU sql.NullString
		if err := rows.Scan(&accordEN, &accordRU); err != nil {
			continue
		}
		if accordEN.Valid {
			accordsEN = append(accordsEN, accordEN.String)
		}
		if accordRU.Valid {
			accordsRU = append(accordsRU, accordRU.String)
		}
	}
	perfume.AccordsEN = strings.Join(accordsEN, ", ")
	perfume.AccordsRU = strings.Join(accordsRU, ", ")

	// Get perfumers
	perfumersQuery := `
		SELECT pf.name
		FROM perfume_perfumers pp
		JOIN perfumers pf ON pp.perfumer_id = pf.id
		WHERE pp.perfume_id = $1
	`

	rows, err = r.db.QueryContext(ctx, perfumersQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get perfumers: %w", err)
	}
	defer rows.Close()

	var perfumers []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		perfumers = append(perfumers, name)
	}
	perfume.Perfumers = strings.Join(perfumers, ", ")

	return perfume, nil
}

// GetByIDs retrieves multiple perfumes by their IDs.
func (r *PerfumeRepo) GetByIDs(ctx context.Context, ids []int64) ([]domain.Perfume, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := `
		SELECT 
			p.id, p.url, p.perfume_name, 
			b.name as brand_en,
			p.gender,
			p.rating_value, p.rating_count, p.year
		FROM perfumes_normalized p
		LEFT JOIN brands b ON p.brand_id = b.id
		WHERE p.id = ANY($1)
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to get perfumes: %w", err)
	}
	defer rows.Close()

	var perfumes []domain.Perfume
	for rows.Next() {
		var p domain.Perfume
		var url, brandEN, gender sql.NullString
		var ratingValue sql.NullFloat64
		var ratingCount, year sql.NullInt64

		if err := rows.Scan(
			&p.ID, &url, &p.Name,
			&brandEN,
			&gender,
			&ratingValue, &ratingCount, &year,
		); err != nil {
			continue
		}

		p.URL = url.String
		p.Brand = brandEN.String
		p.Gender = gender.String
		if ratingValue.Valid {
			p.RatingValue = &ratingValue.Float64
		}
		if ratingCount.Valid {
			rc := int(ratingCount.Int64)
			p.RatingCount = &rc
		}
		if year.Valid {
			y := int(year.Int64)
			p.Year = &y
		}

		perfumes = append(perfumes, p)
	}

	return perfumes, nil
}

// GetSimilar retrieves similar perfumes using pgvector kNN.
// Returns Russian translations when available, falls back to English.
func (r *PerfumeRepo) GetSimilar(ctx context.Context, perfumeID int64, limit int, excludeIDs []int64) ([]domain.PerfumeWithEmbedding, error) {
	// Get rec_embedding for the target perfume (used for item-to-item similarity)
	var embeddingStr string
	err := r.db.QueryRowContext(ctx, `
		SELECT rec_embedding::text 
		FROM perfume_search 
		WHERE perfume_id = $1 AND rec_embedding IS NOT NULL
	`, perfumeID).Scan(&embeddingStr)
	if err != nil {
		if err == sql.ErrNoRows {
			// Perfume doesn't have embeddings yet - return empty result
			r.logger.Warn("Perfume has no embedding for similar search", zap.Int64("perfumeID", perfumeID))
			return []domain.PerfumeWithEmbedding{}, nil
		}
		return nil, fmt.Errorf("failed to get perfume rec_embedding: %w", err)
	}

	embedding, err := parseVector(embeddingStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse embedding: %w", err)
	}

	// Build exclusion list
	excludeList := append([]int64{perfumeID}, excludeIDs...)

	// kNN search using pgvector with rec_embedding for item similarity
	query := `
		SELECT 
			p.id, p.url, p.perfume_name, 
			b.name as brand, 
			p.rating_value, p.rating_count, p.year,
			ps.rec_embedding::text
		FROM perfume_search ps
		JOIN perfumes_normalized p ON ps.perfume_id = p.id
		LEFT JOIN brands b ON p.brand_id = b.id
		WHERE ps.perfume_id != ALL($1) AND ps.rec_embedding IS NOT NULL
		ORDER BY ps.rec_embedding <-> $2::vector
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(excludeList), vectorToString(embedding), limit)
	if err != nil {
		return nil, fmt.Errorf("kNN search failed: %w", err)
	}
	defer rows.Close()

	var results []domain.PerfumeWithEmbedding
	var perfumeIDs []int64
	for rows.Next() {
		var p domain.PerfumeWithEmbedding
		var url, brand sql.NullString
		var ratingValue sql.NullFloat64
		var ratingCount, year sql.NullInt64
		var embStr string

		if err := rows.Scan(
			&p.ID, &url, &p.Name,
			&brand,
			&ratingValue, &ratingCount, &year,
			&embStr,
		); err != nil {
			continue
		}

		p.URL = url.String
		p.Brand = brand.String
		if ratingValue.Valid {
			p.RatingValue = &ratingValue.Float64
		}
		if ratingCount.Valid {
			rc := int(ratingCount.Int64)
			p.RatingCount = &rc
		}
		if year.Valid {
			y := int(year.Int64)
			p.Year = &y
		}
		p.Embedding, _ = parseVector(embStr)

		results = append(results, p)
		perfumeIDs = append(perfumeIDs, p.ID)
	}

	// Enrich with notes and accords (Russian preferred)
	if len(results) > 0 {
		r.enrichPerfumesWithNotesAndAccords(ctx, results, perfumeIDs)
	}

	return results, nil
}

// GetCandidatesForUser retrieves recommendation candidates using user embedding.
// Uses rec_embedding for personalized recommendations.
// Returns Russian translations when available, falls back to English.
func (r *PerfumeRepo) GetCandidatesForUser(ctx context.Context, userEmbedding []float32, limit int, excludeIDs []int64) ([]domain.PerfumeWithEmbedding, error) {
	query := `
		SELECT 
			p.id, p.url, p.perfume_name, 
			b.name as brand,
			p.rating_value, p.rating_count, p.year,
			ps.rec_embedding::text
		FROM perfume_search ps
		JOIN perfumes_normalized p ON ps.perfume_id = p.id
		LEFT JOIN brands b ON p.brand_id = b.id
		WHERE ($1::bigint[] IS NULL OR ps.perfume_id != ALL($1))
		  AND ps.rec_embedding IS NOT NULL
		ORDER BY ps.rec_embedding <-> $2::vector
		LIMIT $3
	`

	var excludeParam interface{} = nil
	if len(excludeIDs) > 0 {
		excludeParam = pq.Array(excludeIDs)
	}

	rows, err := r.db.QueryContext(ctx, query, excludeParam, vectorToString(userEmbedding), limit)
	if err != nil {
		return nil, fmt.Errorf("candidate search failed: %w", err)
	}
	defer rows.Close()

	var results []domain.PerfumeWithEmbedding
	var perfumeIDs []int64
	for rows.Next() {
		var p domain.PerfumeWithEmbedding
		var url, brand sql.NullString
		var ratingValue sql.NullFloat64
		var ratingCount, year sql.NullInt64
		var embStr string

		if err := rows.Scan(
			&p.ID, &url, &p.Name,
			&brand,
			&ratingValue, &ratingCount, &year,
			&embStr,
		); err != nil {
			continue
		}

		p.URL = url.String
		p.Brand = brand.String
		if ratingValue.Valid {
			p.RatingValue = &ratingValue.Float64
		}
		if ratingCount.Valid {
			rc := int(ratingCount.Int64)
			p.RatingCount = &rc
		}
		if year.Valid {
			y := int(year.Int64)
			p.Year = &y
		}
		p.Embedding, _ = parseVector(embStr)

		results = append(results, p)
		perfumeIDs = append(perfumeIDs, p.ID)
	}

	// Enrich with notes and accords (Russian preferred)
	if len(results) > 0 {
		r.enrichPerfumesWithNotesAndAccords(ctx, results, perfumeIDs)
	}

	return results, nil
}

// GetEmbeddingByPerfumeID retrieves the rec_embedding for a specific perfume.
// Used for building user embeddings from liked perfumes.
func (r *PerfumeRepo) GetEmbeddingByPerfumeID(ctx context.Context, perfumeID int64) ([]float32, error) {
	var embStr string
	err := r.db.QueryRowContext(ctx, `
		SELECT rec_embedding::text 
		FROM perfume_search 
		WHERE perfume_id = $1 AND rec_embedding IS NOT NULL
	`, perfumeID).Scan(&embStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}
	return parseVector(embStr)
}

// GetGlobalStats retrieves global statistics for rating calculations.
func (r *PerfumeRepo) GetGlobalStats(ctx context.Context) (meanRating float64, totalPerfumes int64, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(AVG(rating_value), 3.5) as mean_rating,
			COUNT(*) as total
		FROM perfumes_normalized
		WHERE rating_value IS NOT NULL
	`).Scan(&meanRating, &totalPerfumes)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get global stats: %w", err)
	}
	return meanRating, totalPerfumes, nil
}

// GetAll retrieves all perfumes for indexing.
func (r *PerfumeRepo) GetAll(ctx context.Context) ([]domain.Perfume, error) {
	query := `
		SELECT 
			p.id, p.url, p.perfume_name, 
			b.name as brand_en,
			p.rating_value, p.rating_count, p.year
		FROM perfumes_normalized p
		LEFT JOIN brands b ON p.brand_id = b.id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all perfumes: %w", err)
	}
	defer rows.Close()

	var perfumes []domain.Perfume
	for rows.Next() {
		var p domain.Perfume
		var url, brandEN sql.NullString
		var ratingValue sql.NullFloat64
		var ratingCount, year sql.NullInt64

		if err := rows.Scan(
			&p.ID, &url, &p.Name,
			&brandEN,
			&ratingValue, &ratingCount, &year,
		); err != nil {
			continue
		}

		p.URL = url.String
		p.Brand = brandEN.String
		if ratingValue.Valid {
			p.RatingValue = &ratingValue.Float64
		}
		if ratingCount.Valid {
			rc := int(ratingCount.Int64)
			p.RatingCount = &rc
		}
		if year.Valid {
			y := int(year.Int64)
			p.Year = &y
		}

		perfumes = append(perfumes, p)
	}

	return perfumes, nil
}

// FullTextSearch performs PostgreSQL full-text search as a fallback.
// Uses the perfume_search table with GIN indexes for Russian and English text.
// Returns Russian translations when available, falls back to English.
func (r *PerfumeRepo) FullTextSearch(ctx context.Context, query string, limit, offset int) ([]domain.PerfumeCard, int64, error) {
	// Search both Russian and English text columns with ILIKE fallback
	// Returns Russian notes/accords with fallback to English
	searchQuery := `
		WITH search_results AS (
			SELECT DISTINCT ON (ps.perfume_id)
				ps.perfume_id,
				p.perfume_name,
				b.name as brand,
				p.rating_value,
				p.rating_count,
				p.year,
				COALESCE(
					ts_rank(to_tsvector('russian', ps.search_text), plainto_tsquery('russian', $1)) +
					ts_rank(to_tsvector('english', ps.search_text_en), plainto_tsquery('english', $1)),
					0
				) as rank
			FROM perfume_search ps
			JOIN perfumes_normalized p ON ps.perfume_id = p.id
			LEFT JOIN brands b ON p.brand_id = b.id
			WHERE 
				to_tsvector('russian', ps.search_text) @@ plainto_tsquery('russian', $1)
				OR to_tsvector('english', ps.search_text_en) @@ plainto_tsquery('english', $1)
				OR ps.search_text ILIKE '%' || $1 || '%'
				OR ps.search_text_en ILIKE '%' || $1 || '%'
				OR p.perfume_name ILIKE '%' || $1 || '%'
			ORDER BY ps.perfume_id, rank DESC
		)
		SELECT perfume_id, perfume_name, brand, rating_value, rating_count, year
		FROM search_results
		ORDER BY rank DESC, rating_count DESC NULLS LAST
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, searchQuery, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("full-text search failed: %w", err)
	}
	defer rows.Close()

	var cards []domain.PerfumeCard
	var perfumeIDs []int64
	for rows.Next() {
		var card domain.PerfumeCard
		var brand sql.NullString
		var ratingValue sql.NullFloat64
		var ratingCount, year sql.NullInt64

		if err := rows.Scan(
			&card.ID, &card.Name,
			&brand,
			&ratingValue, &ratingCount, &year,
		); err != nil {
			r.logger.Warn("failed to scan full-text search result", zap.Error(err))
			continue
		}

		card.Brand = brand.String
		if ratingValue.Valid {
			card.RatingValue = &ratingValue.Float64
		}
		if ratingCount.Valid {
			rc := int(ratingCount.Int64)
			card.RatingCount = &rc
		}
		if year.Valid {
			y := int(year.Int64)
			card.Year = &y
		}

		cards = append(cards, card)
		perfumeIDs = append(perfumeIDs, card.ID)
	}

	// Enrich cards with notes and accords (Russian preferred)
	if len(cards) > 0 {
		r.enrichCardsWithNotesAndAccords(ctx, cards, perfumeIDs)
	}

	// Get total count
	countQuery := `
		SELECT COUNT(DISTINCT ps.perfume_id)
		FROM perfume_search ps
		JOIN perfumes_normalized p ON ps.perfume_id = p.id
		WHERE 
			to_tsvector('russian', ps.search_text) @@ plainto_tsquery('russian', $1)
			OR to_tsvector('english', ps.search_text_en) @@ plainto_tsquery('english', $1)
			OR ps.search_text ILIKE '%' || $1 || '%'
			OR ps.search_text_en ILIKE '%' || $1 || '%'
			OR p.perfume_name ILIKE '%' || $1 || '%'
	`
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, query).Scan(&total); err != nil {
		r.logger.Warn("failed to get full-text search count", zap.Error(err))
		total = int64(len(cards))
	}

	return cards, total, nil
}

// VectorSearchByEmbedding performs semantic search using query embedding.
// Uses the search_embedding column (768-dim from multilingual-e5-base).
// Returns Russian translations when available, falls back to English.
func (r *PerfumeRepo) VectorSearchByEmbedding(ctx context.Context, embedding []float32, limit, offset int) ([]domain.PerfumeCard, int64, error) {
	// kNN search using pgvector with search_embedding (for query→doc search)
	query := `
		SELECT 
			p.id, p.perfume_name,
			b.name as brand,
			p.rating_value, p.rating_count, p.year
		FROM perfume_search ps
		JOIN perfumes_normalized p ON ps.perfume_id = p.id
		LEFT JOIN brands b ON p.brand_id = b.id
		WHERE ps.embedding IS NOT NULL
		ORDER BY ps.embedding <-> $1::vector
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, vectorToString(embedding), limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("vector search failed: %w", err)
	}
	defer rows.Close()

	var cards []domain.PerfumeCard
	var perfumeIDs []int64
	for rows.Next() {
		var card domain.PerfumeCard
		var brand sql.NullString
		var ratingValue sql.NullFloat64
		var ratingCount, year sql.NullInt64

		if err := rows.Scan(
			&card.ID, &card.Name,
			&brand,
			&ratingValue, &ratingCount, &year,
		); err != nil {
			r.logger.Warn("failed to scan vector search result", zap.Error(err))
			continue
		}

		card.Brand = brand.String
		if ratingValue.Valid {
			card.RatingValue = &ratingValue.Float64
		}
		if ratingCount.Valid {
			rc := int(ratingCount.Int64)
			card.RatingCount = &rc
		}
		if year.Valid {
			y := int(year.Int64)
			card.Year = &y
		}

		cards = append(cards, card)
		perfumeIDs = append(perfumeIDs, card.ID)
	}

	// Enrich with notes and accords (Russian preferred)
	if len(cards) > 0 {
		r.enrichCardsWithNotesAndAccords(ctx, cards, perfumeIDs)
	}

	// Get total count (approximate - just return what we have for now)
	total := int64(len(cards))
	if limit > 0 && len(cards) == limit {
		// There might be more results
		var countResult int64
		err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM perfume_search WHERE embedding IS NOT NULL`).Scan(&countResult)
		if err == nil {
			total = countResult
		}
	}

	return cards, total, nil
}

// enrichCardsWithNotesAndAccords adds notes and accords to PerfumeCards.
// Prefers Russian translations, falls back to English.
func (r *PerfumeRepo) enrichCardsWithNotesAndAccords(ctx context.Context, cards []domain.PerfumeCard, perfumeIDs []int64) {
	// Build ID to index map
	idToIdx := make(map[int64]int)
	for i, id := range perfumeIDs {
		idToIdx[id] = i
	}

	// Get notes (Russian preferred)
	notesQuery := `
		SELECT 
			ptn.perfume_id,
			COALESCE(nt.translation_ru, n.name) as note
		FROM (
			SELECT perfume_id, note_id FROM perfume_top_notes WHERE perfume_id = ANY($1)
			UNION ALL
			SELECT perfume_id, note_id FROM perfume_middle_notes WHERE perfume_id = ANY($1)
			UNION ALL
			SELECT perfume_id, note_id FROM perfume_base_notes WHERE perfume_id = ANY($1)
		) ptn
		JOIN notes n ON ptn.note_id = n.id
		LEFT JOIN note_translations nt ON nt.note_id = n.id
	`

	rows, err := r.db.QueryContext(ctx, notesQuery, pq.Array(perfumeIDs))
	if err == nil {
		defer rows.Close()
		notesMap := make(map[int64][]string)
		for rows.Next() {
			var perfumeID int64
			var note string
			if err := rows.Scan(&perfumeID, &note); err == nil {
				notesMap[perfumeID] = append(notesMap[perfumeID], note)
			}
		}
		for id, notes := range notesMap {
			if idx, ok := idToIdx[id]; ok {
				cards[idx].Notes = truncateNotesString(strings.Join(notes, ", "), 100)
			}
		}
	}

	// Get accords (Russian preferred)
	accordsQuery := `
		SELECT 
			pa.perfume_id,
			COALESCE(at.translation_ru, a.name) as accord
		FROM perfume_accords pa
		JOIN accords a ON pa.accord_id = a.id
		LEFT JOIN accord_translations at ON at.accord_id = a.id
		WHERE pa.perfume_id = ANY($1)
	`

	rows, err = r.db.QueryContext(ctx, accordsQuery, pq.Array(perfumeIDs))
	if err == nil {
		defer rows.Close()
		accordsMap := make(map[int64][]string)
		for rows.Next() {
			var perfumeID int64
			var accord string
			if err := rows.Scan(&perfumeID, &accord); err == nil {
				accordsMap[perfumeID] = append(accordsMap[perfumeID], accord)
			}
		}
		for id, accords := range accordsMap {
			if idx, ok := idToIdx[id]; ok {
				cards[idx].Accords = truncateNotesString(strings.Join(accords, ", "), 80)
			}
		}
	}
}

// enrichPerfumesWithNotesAndAccords adds notes and accords to PerfumeWithEmbedding.
// Prefers Russian translations, falls back to English.
func (r *PerfumeRepo) enrichPerfumesWithNotesAndAccords(ctx context.Context, perfumes []domain.PerfumeWithEmbedding, perfumeIDs []int64) {
	// Build ID to index map
	idToIdx := make(map[int64]int)
	for i, id := range perfumeIDs {
		idToIdx[id] = i
	}

	// Get notes (Russian preferred)
	notesQuery := `
		SELECT 
			ptn.perfume_id,
			COALESCE(nt.translation_ru, n.name) as note
		FROM (
			SELECT perfume_id, note_id FROM perfume_top_notes WHERE perfume_id = ANY($1)
			UNION ALL
			SELECT perfume_id, note_id FROM perfume_middle_notes WHERE perfume_id = ANY($1)
			UNION ALL
			SELECT perfume_id, note_id FROM perfume_base_notes WHERE perfume_id = ANY($1)
		) ptn
		JOIN notes n ON ptn.note_id = n.id
		LEFT JOIN note_translations nt ON nt.note_id = n.id
	`

	rows, err := r.db.QueryContext(ctx, notesQuery, pq.Array(perfumeIDs))
	if err == nil {
		defer rows.Close()
		notesMap := make(map[int64][]string)
		for rows.Next() {
			var perfumeID int64
			var note string
			if err := rows.Scan(&perfumeID, &note); err == nil {
				notesMap[perfumeID] = append(notesMap[perfumeID], note)
			}
		}
		for id, notes := range notesMap {
			if idx, ok := idToIdx[id]; ok {
				perfumes[idx].NotesRU = strings.Join(notes, ", ")
			}
		}
	}

	// Get accords (Russian preferred)
	accordsQuery := `
		SELECT 
			pa.perfume_id,
			COALESCE(at.translation_ru, a.name) as accord
		FROM perfume_accords pa
		JOIN accords a ON pa.accord_id = a.id
		LEFT JOIN accord_translations at ON at.accord_id = a.id
		WHERE pa.perfume_id = ANY($1)
	`

	rows, err = r.db.QueryContext(ctx, accordsQuery, pq.Array(perfumeIDs))
	if err == nil {
		defer rows.Close()
		accordsMap := make(map[int64][]string)
		for rows.Next() {
			var perfumeID int64
			var accord string
			if err := rows.Scan(&perfumeID, &accord); err == nil {
				accordsMap[perfumeID] = append(accordsMap[perfumeID], accord)
			}
		}
		for id, accords := range accordsMap {
			if idx, ok := idToIdx[id]; ok {
				perfumes[idx].AccordsRU = strings.Join(accords, ", ")
			}
		}
	}
}

// truncateNotesString truncates a string to maxLen characters.
func truncateNotesString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// Try to cut at a comma
	truncated := s[:maxLen]
	if idx := strings.LastIndex(truncated, ","); idx > maxLen/2 {
		return truncated[:idx]
	}
	return truncated + "..."
}
