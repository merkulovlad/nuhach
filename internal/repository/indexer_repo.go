package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"nuhach/internal/domain"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchutil"
	"go.uber.org/zap"
)

// IndexerRepo implements domain.IndexerRepository for OpenSearch indexing.
type IndexerRepo struct {
	client    *opensearch.Client
	indexName string
	logger    *zap.Logger
}

// NewIndexerRepo creates a new IndexerRepo.
func NewIndexerRepo(client *opensearch.Client, indexName string, logger *zap.Logger) *IndexerRepo {
	return &IndexerRepo{
		client:    client,
		indexName: indexName,
		logger:    logger,
	}
}

// IndexPerfumes bulk indexes perfumes to OpenSearch.
func (r *IndexerRepo) IndexPerfumes(ctx context.Context, perfumes []domain.Perfume) error {
	indexer, err := opensearchutil.NewBulkIndexer(opensearchutil.BulkIndexerConfig{
		Client:     r.client,
		Index:      r.indexName,
		NumWorkers: 4,
	})
	if err != nil {
		return fmt.Errorf("failed to create bulk indexer: %w", err)
	}

	for _, p := range perfumes {
		doc := perfumeToDoc(p)
		data, err := json.Marshal(doc)
		if err != nil {
			r.logger.Error("failed to marshal perfume", zap.Int64("id", p.ID), zap.Error(err))
			continue
		}

		err = indexer.Add(ctx, opensearchutil.BulkIndexerItem{
			Action:     "index",
			DocumentID: fmt.Sprintf("%d", p.ID),
			Body:       bytes.NewReader(data),
			OnFailure: func(ctx context.Context, item opensearchutil.BulkIndexerItem, res opensearchutil.BulkIndexerResponseItem, err error) {
				r.logger.Error("failed to index document",
					zap.String("doc_id", item.DocumentID),
					zap.String("error", res.Error.Reason),
				)
			},
		})
		if err != nil {
			r.logger.Error("failed to add document to bulk indexer", zap.Error(err))
		}
	}

	if err := indexer.Close(ctx); err != nil {
		return fmt.Errorf("failed to close bulk indexer: %w", err)
	}

	stats := indexer.Stats()
	r.logger.Info("Bulk indexing complete",
		zap.Uint64("indexed", stats.NumFlushed),
		zap.Uint64("failed", stats.NumFailed),
	)

	if stats.NumFailed > 0 {
		return fmt.Errorf("%d documents failed to index", stats.NumFailed)
	}

	return nil
}

// perfumeToDoc converts a Perfume to an OpenSearch document matching the schema.
func perfumeToDoc(p domain.Perfume) map[string]interface{} {
	doc := map[string]interface{}{
		"id":   p.ID,
		"url":  p.URL,
		"name": p.Name,
	}

	if p.Brand != "" {
		doc["brand_en"] = p.Brand
	}
	if p.Gender != "" {
		doc["gender_en"] = p.Gender
	}
	if p.GenderRU != "" {
		doc["gender_ru"] = p.GenderRU
	}
	if p.Year != nil {
		doc["year"] = *p.Year
	}
	if p.RatingValue != nil {
		doc["rating_value"] = *p.RatingValue
	}
	if p.RatingCount != nil {
		doc["rating_count"] = *p.RatingCount
	}
	if p.NotesEN != "" {
		doc["notes_en"] = p.NotesEN
	}
	if p.NotesRU != "" {
		doc["notes_ru"] = p.NotesRU
	}
	if p.AccordsEN != "" {
		doc["accords_en"] = p.AccordsEN
	}
	if p.AccordsRU != "" {
		doc["accords_ru"] = p.AccordsRU
	}
	if p.Perfumers != "" {
		doc["perfumers_en"] = p.Perfumers
	}

	return doc
}

// DeleteIndex deletes the OpenSearch index.
func (r *IndexerRepo) DeleteIndex(ctx context.Context) error {
	res, err := r.client.Indices.Delete([]string{r.indexName})
	if err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() && !strings.Contains(res.String(), "index_not_found") {
		return fmt.Errorf("delete index error: %s", res.String())
	}

	r.logger.Info("Index deleted", zap.String("index", r.indexName))
	return nil
}

// CreateIndex creates the OpenSearch index with proper mapping.
func (r *IndexerRepo) CreateIndex(ctx context.Context) error {
	// Mapping based on opensearch_index.md
	mapping := `{
		"settings": {
			"number_of_shards": 1,
			"number_of_replicas": 0,
			"analysis": {
				"filter": {
					"ru_stop": { "type": "stop", "stopwords": "_russian_" },
					"en_stop": { "type": "stop", "stopwords": "_english_" },
					"ru_stemmer": { "type": "stemmer", "language": "russian" },
					"en_stemmer": { "type": "stemmer", "language": "english" }
				},
				"analyzer": {
					"mix_ru_en": {
						"type": "custom",
						"tokenizer": "standard",
						"filter": ["lowercase", "asciifolding", "ru_stop", "en_stop", "ru_stemmer", "en_stemmer"]
					}
				}
			}
		},
		"mappings": {
			"properties": {
				"id": { "type": "long" },
				"url": { "type": "keyword" },
				"name": {
					"type": "text",
					"analyzer": "mix_ru_en",
					"fields": { "keyword": { "type": "keyword" } }
				},
				"brand_en": { "type": "text", "analyzer": "mix_ru_en", "fields": { "keyword": { "type": "keyword" } } },
				"gender_en": { "type": "keyword" },
				"gender_ru": { "type": "keyword" },
				"year": { "type": "integer" },
				"rating_value": { "type": "float" },
				"rating_count": { "type": "integer" },
				"notes_en": { "type": "text", "analyzer": "mix_ru_en" },
				"notes_ru": { "type": "text", "analyzer": "mix_ru_en" },
				"accords_en": { "type": "text", "analyzer": "mix_ru_en" },
				"accords_ru": { "type": "text", "analyzer": "mix_ru_en" },
				"perfumers_en": { "type": "text", "analyzer": "mix_ru_en", "fields": { "keyword": { "type": "keyword" } } }
			}
		}
	}`

	res, err := r.client.Indices.Create(
		r.indexName,
		r.client.Indices.Create.WithBody(strings.NewReader(mapping)),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("create index error: %s", res.String())
	}

	r.logger.Info("Index created", zap.String("index", r.indexName))
	return nil
}
