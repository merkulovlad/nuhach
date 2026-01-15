package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"nuhach/internal/infra/config"
	"nuhach/internal/infra/db"
	"nuhach/internal/infra/logger"
	"nuhach/internal/repository"

	"go.uber.org/zap"
)

func main() {
	// Flags
	date := flag.String("date", time.Now().AddDate(0, 0, -1).Format("2006-01-02"), "Date to compute metrics for (YYYY-MM-DD)")
	surface := flag.String("surface", "", "Surface to compute metrics for (search, recommendations, or empty for all)")
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

	log.Info("Computing analytics",
		zap.String("date", *date),
		zap.String("surface", *surface),
	)

	// Connect to PostgreSQL
	database, err := db.Connect(cfg.DatabaseURL(), log)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close()

	ctx := context.Background()

	// Initialize repository
	analyticsRepo := repository.NewAnalyticsRepo(database, log)

	// Compute metrics for specified surface(s)
	surfaces := []string{"search", "recommendations"}
	if *surface != "" {
		surfaces = []string{*surface}
	}

	for _, s := range surfaces {
		log.Info("Computing metrics", zap.String("surface", s))

		metrics, err := analyticsRepo.ComputeDailyMetrics(ctx, *date, s)
		if err != nil {
			log.Error("Failed to compute metrics", zap.String("surface", s), zap.Error(err))
			continue
		}

		// Store metrics
		if err := analyticsRepo.StoreDailyMetrics(ctx, metrics); err != nil {
			log.Error("Failed to store metrics", zap.Error(err))
			continue
		}

		log.Info("Metrics computed and stored",
			zap.String("date", metrics.Date),
			zap.String("surface", metrics.Surface),
			zap.Float64("ctr", metrics.CTR),
			zap.Float64("precision_k", metrics.PrecisionK),
			zap.Float64("coverage", metrics.Coverage),
			zap.Float64("novelty", metrics.Novelty),
			zap.Int64("impressions", metrics.Impressions),
			zap.Int64("clicks", metrics.Clicks),
		)
	}

	// Print summary
	printMetricsSummary(ctx, database, *date)
}

func printMetricsSummary(ctx context.Context, database *sql.DB, date string) {
	fmt.Println("\n=== Analytics Summary ===")
	fmt.Printf("Date: %s\n\n", date)

	rows, err := database.QueryContext(ctx, `
		SELECT surface, ctr, precision_k, coverage, novelty, impressions, clicks
		FROM analytics_daily
		WHERE date = $1
		ORDER BY surface
	`, date)
	if err != nil {
		fmt.Printf("Error fetching summary: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Printf("%-15s %8s %12s %10s %10s %12s %8s\n",
		"Surface", "CTR", "Precision@K", "Coverage", "Novelty", "Impressions", "Clicks")
	fmt.Println(strings.Repeat("-", 80))

	for rows.Next() {
		var surface string
		var ctr, precisionK, coverage, novelty float64
		var impressions, clicks int64

		if err := rows.Scan(&surface, &ctr, &precisionK, &coverage, &novelty, &impressions, &clicks); err != nil {
			continue
		}

		fmt.Printf("%-15s %8.4f %12.4f %10.4f %10.4f %12d %8d\n",
			surface, ctr, precisionK, coverage, novelty, impressions, clicks)
	}
}
