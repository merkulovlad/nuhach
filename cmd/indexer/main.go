package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/merkulovlad/nuhach/internal/infra/config"
	"github.com/merkulovlad/nuhach/internal/infra/db"
	"github.com/merkulovlad/nuhach/internal/infra/logger"
	"github.com/merkulovlad/nuhach/internal/infra/opensearch"
	"github.com/merkulovlad/nuhach/internal/repository"

	"go.uber.org/zap"
)

func main() {
	// Flags
	recreate := flag.Bool("recreate", false, "Recreate the index (delete and create)")

	flag.Parse()

	// Initialize logger
	log, err := logger.New(true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration", zap.Error(err))
	}

	log.Info("Starting indexer",
		zap.String("index", cfg.OpenSearchIndex),
		zap.Bool("recreate", *recreate),
	)

	// Connect to PostgreSQL
	database, err := db.Connect(cfg.DatabaseURL(), log)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close()

	// Connect to OpenSearch
	osClient, err := opensearch.NewClient(cfg.OpenSearchURL(), log)
	if err != nil {
		log.Fatal("Failed to connect to OpenSearch", zap.Error(err))
	}

	ctx := context.Background()

	// Initialize repositories
	perfumeRepo := repository.NewPerfumeRepo(database, log)
	indexerRepo := repository.NewIndexerRepo(osClient, cfg.OpenSearchIndex, log)

	// Recreate index if requested
	if *recreate {
		log.Info("Deleting existing index...")

		if err := indexerRepo.DeleteIndex(ctx); err != nil {
			log.Warn("Failed to delete index (may not exist)", zap.Error(err))
		}

		log.Info("Creating index with mapping...")

		if err := indexerRepo.CreateIndex(ctx); err != nil {
			log.Fatal("Failed to create index", zap.Error(err))
		}
	}

	// Load all perfumes from PostgreSQL
	log.Info("Loading perfumes from database...")

	perfumes, err := perfumeRepo.GetAll(ctx)
	if err != nil {
		log.Fatal("Failed to load perfumes", zap.Error(err))
	}

	log.Info("Loaded perfumes", zap.Int("count", len(perfumes)))

	// Index to OpenSearch
	log.Info("Indexing perfumes to OpenSearch...")

	if err := indexerRepo.IndexPerfumes(ctx, perfumes); err != nil {
		log.Fatal("Failed to index perfumes", zap.Error(err))
	}

	log.Info("Indexing complete!")
}
