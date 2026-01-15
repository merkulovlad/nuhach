package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nuhach/internal/infra/config"
	"nuhach/internal/infra/db"
	"nuhach/internal/infra/logger"
	"nuhach/internal/infra/opensearch"
	"nuhach/internal/repository"
	transporthttp "nuhach/internal/transport/http"
	"nuhach/internal/usecase"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	log, err := logger.New(os.Getenv("GO_ENV") != "production")
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

	log.Info("Starting API server",
		zap.Int("port", cfg.ServerPort),
		zap.String("db_host", cfg.DBHost),
		zap.String("opensearch_host", cfg.OpenSearchHost),
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

	// Initialize repositories
	perfumeRepo := repository.NewPerfumeRepo(database, log)
	searchRepo := repository.NewSearchRepo(osClient, cfg.OpenSearchIndex, log)
	userRepo := repository.NewUserRepo(database, log)
	userEmbeddingRepo := repository.NewUserEmbeddingRepo(database, log, cfg.EmbeddingDim)
	eventRepo := repository.NewEventRepo(database, log)

	// Initialize use cases
	searchUC := usecase.NewSearchUseCase(searchRepo, perfumeRepo, eventRepo, userRepo, log)
	recsUC := usecase.NewRecommendationUseCase(
		perfumeRepo, userRepo, userEmbeddingRepo, eventRepo, log,
		cfg.BayesianM, cfg.ExplorationRate, cfg.EmbeddingDim, cfg.RecCandidateLimit,
	)
	eventUC := usecase.NewEventUseCase(userRepo, userEmbeddingRepo, eventRepo, perfumeRepo, log, cfg.EmbeddingDim)

	// Initialize HTTP handler
	handler := transporthttp.NewHandler(searchUC, recsUC, eventUC, log)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Nuhach Perfume API",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	})

	// Middleware
	app.Use(cors.New())
	app.Use(transporthttp.RecoveryMiddleware(log))
	app.Use(transporthttp.LoggingMiddleware(log))

	// Register routes
	handler.RegisterRoutes(app)

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Info("Shutting down server...")
		cancel()
		app.Shutdown()
	}()

	// Start server
	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	if err := app.Listen(addr); err != nil {
		log.Error("Server error", zap.Error(err))
	}

	<-ctx.Done()
	log.Info("Server stopped")
}
