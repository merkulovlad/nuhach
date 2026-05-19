.PHONY: help docker-up docker-down docker-logs embeddings embeddings-venv

EMBEDDINGS_VENV ?= .venv-embeddings
EMBEDDINGS_PYTHON := $(EMBEDDINGS_VENV)/bin/python

# Default target
help:
	@echo "Nuhach Perfume Bot - Docker Commands"
	@echo ""
	@echo "Available commands:"
	@echo "  make up       - Start all services (Telegram bot + API + databases)"
	@echo "  make down     - Stop all services"
	@echo "  make logs     - View all logs"
	@echo "  make restart  - Restart all services"
	@echo "  make embeddings - Generate embeddings parquet"
	@echo ""
	@echo "Individual service logs:"
	@echo "  docker compose logs -f bot"
	@echo "  docker compose logs -f api"
	@echo ""
	@echo "Note: All services run in Docker. Use 'docker compose up -d' directly."

# Start all services
up:
	@echo "Starting all services..."
	docker compose up -d

# Stop all services  
down:
	@echo "Stopping all services..."
	docker compose down

# View logs
logs:
	@echo "Viewing logs (Ctrl+C to exit)..."
	docker compose logs -f

# Restart services
restart:
	@echo "Restarting all services..."
	docker compose restart

# Install isolated Python dependencies for embeddings generation
embeddings-venv: $(EMBEDDINGS_VENV)/.installed

$(EMBEDDINGS_VENV)/.installed: requirements-embeddings.txt
	python3 -m venv $(EMBEDDINGS_VENV)
	$(EMBEDDINGS_PYTHON) -m pip install --upgrade pip
	$(EMBEDDINGS_PYTHON) -m pip install -r requirements-embeddings.txt
	touch $(EMBEDDINGS_VENV)/.installed

# Generate embeddings parquet for ingest-embeddings
embeddings: embeddings-venv
	$(EMBEDDINGS_PYTHON) scripts/05_create_embeddings.py --force
