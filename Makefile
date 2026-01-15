.PHONY: all build test lint migrate index analytics run clean help

# Default target
all: build

# Build all binaries
build:
	@echo "Building binaries..."
	go build -o bin/api ./cmd/api
	go build -o bin/indexer ./cmd/indexer
	go build -o bin/analytics ./cmd/analytics

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run ./...

# Run database migrations (requires running PostgreSQL)
migrate:
	@echo "Running migrations..."
	goose -dir migrations postgres "$$DATABASE_URL" up

# Rollback last migration
migrate-down:
	@echo "Rolling back last migration..."
	goose -dir migrations postgres "$$DATABASE_URL" down

# Migration status
migrate-status:
	@echo "Migration status..."
	goose -dir migrations postgres "$$DATABASE_URL" status

# Run OpenSearch indexer
index:
	@echo "Running OpenSearch indexer..."
	go run ./cmd/indexer

# Run analytics computation
analytics:
	@echo "Running analytics computation..."
	go run ./cmd/analytics

# Run API server locally (for development)
run:
	@echo "Starting API server..."
	go run ./cmd/api

# Docker compose commands
docker-up:
	@echo "Starting all services..."
	docker compose up -d

docker-down:
	@echo "Stopping all services..."
	docker compose down

docker-logs:
	@echo "Showing logs..."
	docker compose logs -f

docker-build:
	@echo "Building Docker images..."
	docker compose build

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install development tools
tools:
	@echo "Installing tools..."
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Generate go.sum
deps:
	@echo "Tidying dependencies..."
	go mod tidy

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build all binaries"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Lint code"
	@echo "  migrate        - Run database migrations"
	@echo "  migrate-down   - Rollback last migration"
	@echo "  migrate-status - Show migration status"
	@echo "  index          - Run OpenSearch indexer"
	@echo "  analytics      - Run analytics computation"
	@echo "  run            - Run API server locally"
	@echo "  docker-up      - Start all services with docker compose"
	@echo "  docker-down    - Stop all services"
	@echo "  docker-logs    - Show docker compose logs"
	@echo "  docker-build   - Build Docker images"
	@echo "  clean          - Clean build artifacts"
	@echo "  tools          - Install development tools"
	@echo "  deps           - Tidy dependencies"
	@echo "  help           - Show this help"
