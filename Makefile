.PHONY: help docker-up docker-down docker-logs

# Default target
help:
	@echo "Nuhach Perfume Bot - Docker Commands"
	@echo ""
	@echo "Available commands:"
	@echo "  make up       - Start all services (Telegram bot + API + databases)"
	@echo "  make down     - Stop all services"
	@echo "  make logs     - View all logs"
	@echo "  make restart  - Restart all services"
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
