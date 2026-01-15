// Package config handles application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration.
type Config struct {
	// Database configuration
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string

	// OpenSearch configuration
	OpenSearchHost  string
	OpenSearchPort  int
	OpenSearchIndex string

	// Server configuration
	ServerPort int

	// Algorithm configuration
	BayesianM         float64 // Threshold for Bayesian weighted rating
	ExplorationRate   float64 // Epsilon for exploration
	EmbeddingDim      int     // Embedding dimension (384 based on migrations)
	RecCandidateLimit int     // Number of candidates for recommendations
}

// Load reads configuration from environment variables using existing patterns.
func Load() (*Config, error) {
	cfg := &Config{
		// Database - use existing env vars from .env
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 5432),
		DBName:     getEnv("DB_NAME", "nuhach"),
		DBUser:     getEnv("DB_USER", "admin"),
		DBPassword: getEnv("DB_PASSWORD", "securepassword123"),

		// OpenSearch - use existing patterns from 02_ingest.py
		OpenSearchHost:  getEnv("OPENSEARCH_HOST", "localhost"),
		OpenSearchPort:  getEnvInt("OPENSEARCH_PORT", 9200),
		OpenSearchIndex: getEnv("OPENSEARCH_INDEX", "perfumes"),

		// Server
		ServerPort: getEnvInt("SERVER_PORT", 8080),

		// Algorithm defaults - using sensible defaults, not inventing new env vars
		BayesianM:         10.0, // Threshold for weighted rating
		ExplorationRate:   0.05, // 5% exploration
		EmbeddingDim:      768,  // From multilingual-e5-base model
		RecCandidateLimit: 200,  // Top-N candidates for recs
	}

	return cfg, nil
}

// DatabaseURL returns the PostgreSQL connection string.
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

// OpenSearchURL returns the OpenSearch URL.
func (c *Config) OpenSearchURL() string {
	return fmt.Sprintf("http://%s:%d", c.OpenSearchHost, c.OpenSearchPort)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
