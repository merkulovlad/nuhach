package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/merkulovlad/nuhach/internal/domain"

	"github.com/opensearch-project/opensearch-go/v2"
	"go.uber.org/zap"
)

// SearchRepo implements domain.SearchRepository using OpenSearch.
type SearchRepo struct {
	client    *opensearch.Client
	indexName string
	logger    *zap.Logger
}

// NewSearchRepo creates a new SearchRepo.
func NewSearchRepo(client *opensearch.Client, indexName string, logger *zap.Logger) *SearchRepo {
	return &SearchRepo{
		client:    client,
		indexName: indexName,
		logger:    logger,
	}
}

// searchHit represents an OpenSearch search hit.
type searchHit struct {
	ID     string                 `json:"_id"`
	Score  float64                `json:"_score"`
	Source map[string]interface{} `json:"_source"`
}

// searchResponse represents an OpenSearch search response.
type searchResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []searchHit `json:"hits"`
	} `json:"hits"`
}

// BuildSearchQuery constructs the OpenSearch multi_match query with field boosts.
// Exported for testing.
func BuildSearchQuery(query string, limit, offset int) ([]byte, error) {
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"function_score": map[string]interface{}{
				"query": map[string]interface{}{
					"multi_match": map[string]interface{}{
						"query": query,
						"type":  "best_fields",
						"fields": []string{
							"name^5",

							"brand_en^4",
							"accords_ru^3",
							"accords_en^2.5",
							"notes_ru^2",
							"notes_en^1.5",
							"perfumers_en^1.2",
						},
						"fuzziness": "AUTO",
					},
				},
				// Light popularity boost using rating_value and log1p(rating_count)
				"functions": []map[string]interface{}{
					{
						"field_value_factor": map[string]interface{}{
							"field":    "rating_value",
							"factor":   0.1,
							"modifier": "sqrt",
							"missing":  3.5, // Neutral if missing
						},
					},
					{
						"field_value_factor": map[string]interface{}{
							"field":    "rating_count",
							"factor":   0.01,
							"modifier": "log1p",
							"missing":  1, // Neutral if missing
						},
					},
				},
				"score_mode": "multiply",
				"boost_mode": "multiply",
			},
		},
		"from": offset,
		"size": limit,
	}
	return json.Marshal(searchQuery)
}

// Search performs a BM25 multi_match search with field boosts.
func (r *SearchRepo) Search(ctx context.Context, query string, limit, offset int) ([]domain.PerfumeCard, int64, error) {
	// Build multi_match query with field boosts based on opensearch_index.md schema
	body, err := BuildSearchQuery(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal search query: %w", err)
	}

	r.logger.Debug("OpenSearch query", zap.String("query", string(body)))

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.indexName),
		r.client.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("search request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("search error: %s", res.String())
	}

	var response searchResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	cards := make([]domain.PerfumeCard, 0, len(response.Hits.Hits))
	for _, hit := range response.Hits.Hits {
		card := mapHitToCard(hit.Source)
		cards = append(cards, card)
	}

	return cards, response.Hits.Total.Value, nil
}

// mapHitToCard converts an OpenSearch hit to a PerfumeCard.
func mapHitToCard(source map[string]interface{}) domain.PerfumeCard {
	card := domain.PerfumeCard{}

	if v, ok := source["id"].(float64); ok {
		card.ID = int64(v)
	}
	if v, ok := source["name"].(string); ok {
		card.Name = v
	}
	if v, ok := source["brand_en"].(string); ok {
		card.Brand = v
	}
	if v, ok := source["rating_value"].(float64); ok {
		card.RatingValue = &v
	}
	if v, ok := source["rating_count"].(float64); ok {
		i := int(v)
		card.RatingCount = &i
	}
	if v, ok := source["year"].(float64); ok {
		i := int(v)
		card.Year = &i
	}
	// Prefer Russian notes/accords
	if v, ok := source["notes_ru"].(string); ok && v != "" {
		card.Notes = truncateString(v, 100)
	} else if v, ok := source["notes_en"].(string); ok {
		card.Notes = truncateString(v, 100)
	}
	if v, ok := source["accords_ru"].(string); ok && v != "" {
		card.Accords = truncateString(v, 80)
	} else if v, ok := source["accords_en"].(string); ok {
		card.Accords = truncateString(v, 80)
	}

	return card
}

// truncateString truncates a string to maxLen characters.
func truncateString(s string, maxLen int) string {
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

// VectorSearch performs semantic search using embeddings (placeholder).
// For actual vector search, we need query embeddings from an embedding service.
// This implementation returns empty results - the use case layer handles fallback.
func (r *SearchRepo) VectorSearch(ctx context.Context, queryEmbedding []float32, limit, offset int) ([]domain.PerfumeCard, int64, error) {
	// This would require query embedding generation.
	// For now, this is a placeholder that the use case can call with pre-computed embeddings.
	r.logger.Debug("VectorSearch called", zap.Int("embeddingLen", len(queryEmbedding)))
	return nil, 0, nil
}
