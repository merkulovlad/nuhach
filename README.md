# Nuhach Perfume API

Go 1.24 backend for perfume search and recommendations using Clean Architecture.

## Features

- **Search**: BM25 multi-match search with field boosts using OpenSearch
- **Recommendations**: Personalized recommendations with Bayesian weighted rating and epsilon-greedy exploration
- **Similar Perfumes**: pgvector kNN similarity search
- **User Events**: Track interactions (impressions, clicks, likes, dislikes, saves)
- **Analytics**: Daily metrics computation (CTR, Precision@K, Coverage, Novelty)

## Architecture

```
internal/
├── domain/          # Entities and repository interfaces
├── usecase/         # Business logic
├── repository/      # Data access implementations
├── transport/http/  # Fiber HTTP handlers
└── infra/           # Config, DB, OpenSearch clients
```

## Quick Start

```bash
# Start everything with one command (postgres, opensearch, api, ingest, indexer, telegram bot)
docker compose up -d

# View logs
docker compose logs -f

# View specific service logs
docker compose logs -f bot
docker compose logs -f api
```

### Startup Order

The services start automatically in the correct order:

1. **postgres** + **opensearch** - Start and wait for health checks
2. **migrate** - Run database migrations
3. **api** - Start the API server (waits for health check)
4. **ingest** - Load data from CSV into PostgreSQL
5. **indexer** - Build OpenSearch index from PostgreSQL data
6. **bot** - Start Telegram bot (connects to API)

### Stop Everything

```bash
docker compose down        # Stop containers
docker compose down -v     # Stop and remove volumes (fresh start)
```

## API Endpoints

### Health Check
```http
GET /api/health
```

### Search
```http
GET /api/search?q=роза&limit=20&offset=0&tg_id=123456
```

Query parameters:
- `q` (required): Search query
- `limit` (optional, default: 20, max: 100): Results per page
- `offset` (optional, default: 0): Pagination offset
- `tg_id` (optional): Telegram user ID for impression logging

Response:
```json
{
  "items": [
    {
      "id": 1,
      "name": "Perfume Name",
      "brand": "Brand",
      "rating_value": 4.5,
      "rating_count": 100,
      "year": 2020,
      "notes": "rose, jasmine",
      "accords": "floral, fresh"
    }
  ],
  "request_id": "uuid",
  "total": 150
}
```

### Get Perfume Details
```http
GET /api/perfumes/:id
```

### Get Similar Perfumes
```http
GET /api/perfumes/:id/similar?limit=10&tg_id=123456
```

### Get Recommendations
```http
GET /api/users/:tg_id/recommendations?limit=20
```

Response includes `exploration_ids` array indicating which items were from exploration (epsilon-greedy).

### Create Event
```http
POST /api/users/:tg_id/events
Content-Type: application/json

{
  "perfume_id": 123,
  "event_type": "like",
  "rating": 5,
  "request_id": "uuid-from-search"
}
```

Event types: `impression`, `click`, `like`, `dislike`, `save`, `my_saves`

### Get User Saves
```http
GET /api/users/:tg_id/saves
```

## Configuration

Uses existing environment variables from `.env`:

| Variable | Description | Default |
|----------|-------------|---------|
| DB_HOST | PostgreSQL host | localhost |
| DB_PORT | PostgreSQL port | 5432 |
| DB_NAME | Database name | nuhach |
| DB_USER | Database user | admin |
| DB_PASSWORD | Database password | - |
| OPENSEARCH_HOST | OpenSearch host | localhost |
| OPENSEARCH_PORT | OpenSearch port | 9200 |
| OPENSEARCH_INDEX | Index name | perfumes |
| SERVER_PORT | API server port | 8080 |

## Recommendation Algorithm

### Bayesian Weighted Rating
$$wr = \frac{v}{v+m} \cdot R + \frac{m}{v+m} \cdot C$$

Where:
- $v$ = rating_count
- $R$ = rating_value
- $m$ = threshold (default: 10)
- $C$ = global mean rating

### Scoring Formula
```
final_score = 0.7 * similarity + 0.3 * normalized_weighted_rating
```

### Exploration
Epsilon-greedy with 5% exploration rate.

## Analytics Metrics

- **CTR**: Click-through rate
- **Precision@K**: Fraction of shown items that were liked
- **Coverage**: Unique items shown / total catalog
- **Novelty**: Inverse log popularity of shown items

---

## Telegram Bot

A Python Telegram bot is available in the `bot/` directory.

### Bot Features

- **Search**: `/search <query>` - Search perfumes by name, notes, or brand
- **Recommendations**: `/recommend` - Get personalized recommendations
- **Saves**: `/saves` - View saved perfumes
- **Inline buttons**: Details, Similar, Like, Dislike, Save

### Bot Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `BOT_API_KEY` | Telegram bot token | **Yes** | - |
| `API_BASE_URL` | Backend API URL | No | `http://localhost:8080` |
| `BOT_REQUEST_TIMEOUT` | API request timeout (seconds) | No | `10.0` |
| `BOT_MAX_RETRIES` | Max API retry attempts | No | `3` |
| `BOT_RETRY_DELAY` | Delay between retries (seconds) | No | `0.5` |

### Running the Bot

```bash
# Install Python dependencies
pip install -r requirements.txt

# Ensure .env has BOT_API_KEY set
# Ensure the API service is running (docker compose up -d api)

# Run the bot
python -m bot.main
```

### Bot Sample Commands

```
/start          - Show welcome message and commands
/help           - Show help message
/search rose    - Search for perfumes with "rose"
/search vanilla noir - Search for "vanilla noir"
/recommend      - Get personalized recommendations
/saves          - View your saved perfumes
```

### Bot Testing

```bash
# Run bot tests
pytest bot/tests/ -v
```

---

## Development

```bash
# Install dependencies
go mod tidy

# Run tests
go test ./...

# Run API locally
go run ./cmd/api

# Run indexer
go run ./cmd/indexer -recreate

# Run analytics
go run ./cmd/analytics
```

## Make Targets

```bash
make build          # Build all binaries
make test           # Run tests
make test-coverage  # Run tests with coverage
make docker-up      # Start all services
make docker-down    # Stop all services
make migrate        # Run database migrations
make index          # Run OpenSearch indexer
make analytics      # Run analytics computation
```

