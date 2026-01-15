# Nuhach - Telegram Perfume Bot 👃

Telegram bot for perfume recommendations powered by hybrid search (BM25 + embeddings) and personalized recommendations.

## Features

- **Hybrid Search**: Combines OpenSearch BM25 with pgvector semantic search (multilingual-e5-base embeddings)
- **Personalized Recommendations**: Bayesian weighted ratings + epsilon-greedy exploration
- **Similar Perfumes**: Item-to-item similarity using rec_embedding (paraphrase-multilingual-mpnet)
- **User Tracking**: Impressions, clicks, likes, dislikes, saves
- **24,063 Perfumes** with Russian translations for notes/accords

## Quick Start

```bash
# 1. Set up environment
cp .env.example .env
# Add your BOT_API_KEY to .env

# 2. Start everything with one command
docker compose up -d

# 3. View logs
docker compose logs -f bot
docker compose logs -f api
```

**⏱️ First Startup**: Takes 3-5 minutes (embeddings ingestion + model download)  
**🚀 Subsequent Restarts**: <30 seconds

That's it! The bot is now running and connected to Telegram.

### What Happens on Startup

Services start automatically in order:

1. **PostgreSQL** + **OpenSearch** - Databases start and health checks pass
2. **Migrate** - Database migrations run
3. **API** - Go backend starts (Fiber HTTP server)
4. **Ingest** - Loads 24,063 perfumes from CSV into PostgreSQL
5. **Ingest Embeddings** - Loads search + rec embeddings (768-dim vectors) into perfume_search table
6. **Indexer** - Builds OpenSearch BM25 index from PostgreSQL data
7. **Bot** - Telegram bot starts (downloads multilingual-e5-base model on first run)

### Why First Startup is Slow

**Embedding Ingestion** (~2-3 minutes):
- Loads 584MB parquet file with 24,063 perfumes
- Each perfume has 2 embeddings × 768 dimensions = ~1.5M floats
- Inserts into PostgreSQL with pgvector indexing

**Model Download** (~1-2 minutes):
- Bot downloads `intfloat/multilingual-e5-base` (~500MB) from HuggingFace
- Cached in container after first download
- Pre-loads model before accepting requests (no hanging on first search)

**Subsequent Restarts**: Only services restart, data persists in Docker volumes.

## Bot Commands

```
/start      - Welcome message
/search <query> - Search perfumes (e.g., /search роза ваниль)
/recommend  - Get personalized recommendations
/saves      - View saved perfumes
/help       - Show commands
```

### Search Examples

```
/search версаче версенс   - Finds Versace perfumes (hybrid Russian/English search)
/search tom ford oud      - Searches by brand + note
/search цветочный сладкий - Searches by accords in Russian
```

## Architecture

```
├── bot/                    # Python Telegram bot (aiogram v3)
│   ├── main.py            # Bot entry point
│   ├── handlers.py        # Command & callback handlers
│   ├── api_client.py      # HTTP client for backend API
│   ├── embedding.py       # multilingual-e5-base for query embeddings
│   ├── formatters.py      # Message formatting
│   └── keyboards.py       # Inline keyboards
│
├── cmd/
│   ├── api/               # Go HTTP API server (Fiber)
│   ├── indexer/           # OpenSearch indexer
│   └── analytics/         # Daily metrics computation
│
├── internal/
│   ├── domain/            # Entities & repository interfaces
│   ├── usecase/           # Business logic
│   │   ├── search.go      # Hybrid search with RRF fusion
│   │   └── recommendations.go  # Personalized recs
│   ├── repository/        # PostgreSQL + OpenSearch + pgvector
│   └── transport/http/    # Fiber HTTP handlers
│
├── migrations/            # PostgreSQL migrations (pgvector, tables, indexes)
├── scripts/               # Data ingestion scripts
│   ├── 03_ingest_normalized.py   # Load perfumes from CSV
│   └── 04_ingest_embeddings.py  # Load embeddings from parquet
│
└── data/
    └── processed/
        ├── dataset_final.csv                   # Perfume data with Russian translations
        └── perfumes_with_embeddings.parquet    # 768-dim embeddings (search + rec)
```

## Hybrid Search

### How It Works

1. **User sends query**: "версаче версенс" (Russian misspelling)
2. **Bot generates embedding**: multilingual-e5-base encodes "query: версаче версенс" → 768-dim vector
3. **API performs hybrid search**:
   - **Vector search**: pgvector finds 30 nearest neighbors using `embedding <-> query_vector`
   - **BM25 search**: OpenSearch multi-match on `name^5, brand_en^4, accords_ru^3, notes_ru^2`
4. **RRF Fusion**: Merges both result lists using Reciprocal Rank Fusion with k=60
5. **Returns top 10** with Russian notes/accords + English brands

### Embeddings

- **search_embedding** (multilingual-e5-base, 768-dim): Query→document search
- **rec_embedding** (paraphrase-multilingual-mpnet, 768-dim): Item-to-item similarity

Both stored in `perfume_search` table with pgvector `<->` operator for cosine distance.

## API Endpoints

### Search (Hybrid)
```http
POST /api/search/vector
Content-Type: application/json

{
  "query": "роза ваниль",
  "embedding": [0.023, -0.041, ...],  // 768-dim vector
  "limit": 10,
  "tg_id": 123456
}
```

### Recommendations
```http
GET /api/users/:tg_id/recommendations?limit=20
```

Returns personalized items with `exploration_ids` array (epsilon-greedy).

### Similar Perfumes
```http
GET /api/perfumes/:id/similar?limit=10
```

Uses `rec_embedding` for item-to-item kNN.

### Event Tracking
```http
POST /api/users/:tg_id/events
Content-Type: application/json

{
  "perfume_id": 123,
  "event_type": "like",
  "request_id": "uuid"
}
```

Event types: `impression`, `click`, `like`, `dislike`, `save`

## Configuration

All config in `.env`:

```env
# Telegram Bot
BOT_API_KEY=your_telegram_bot_token

# Database (PostgreSQL)
DB_HOST=postgres
DB_PORT=5432
DB_NAME=nuhach
DB_USER=admin
DB_PASSWORD=securepassword123

# OpenSearch
OPENSEARCH_HOST=opensearch
OPENSEARCH_PORT=9200
OPENSEARCH_INDEX=perfumes

# API
SERVER_PORT=8080
```

## Recommendations Algorithm

### Bayesian Weighted Rating
$$wr = \frac{v}{v+m} \cdot R + \frac{m}{v+m} \cdot C$$

Where:
- $v$ = rating_count (number of votes)
- $R$ = rating_value (average rating)  
- $m$ = threshold (default: 10)
- $C$ = global mean rating (~3.8)

### Scoring
```go
similarity := cosineSimilarity(user_embedding, perfume_embedding)
normalized_rating := bayesian_rating / 5.0
final_score := 0.7 * similarity + 0.3 * normalized_rating
```

### Exploration
Epsilon-greedy with 5% exploration rate - occasionally shows random perfumes to avoid filter bubbles.

## Development

### Prerequisites
- Docker & Docker Compose
- Python 3.12+ (for local bot development)
- Go 1.24+ (for API development)

### Local Development

```bash
# Start only databases
docker compose up -d postgres opensearch

# Run API locally
go run ./cmd/api

# Run bot locally (after pip install -r requirements.txt)
export BOT_API_KEY=your_token
export API_BASE_URL=http://localhost:8080
python -m bot.main
```

### Database Access

```bash
# PostgreSQL
docker compose exec postgres psql -U admin -d nuhach

# Check embeddings
SELECT COUNT(*) FROM perfume_search WHERE embedding IS NOT NULL;

# OpenSearch
curl http://localhost:9200/perfumes/_search?pretty
```

### Useful Commands

```bash
# View specific service logs
docker compose logs -f bot
docker compose logs -f api

# Restart service
docker compose restart bot

# Rebuild after code changes
docker compose build bot && docker compose up -d bot
docker compose build api && docker compose up -d api

# Fresh start (deletes all data)
docker compose down -v && docker compose up -d
```

## Makefile Commands

```bash
make up       # Start all services
make down     # Stop all services  
make logs     # View all logs
make restart  # Restart all services
```

## Tech Stack

- **Bot**: Python 3.12, aiogram 3.4, sentence-transformers, torch
- **API**: Go 1.24, Fiber, pgx (PostgreSQL driver)
- **Search**: OpenSearch 2.12 (BM25), pgvector (cosine similarity)
- **Embeddings**: multilingual-e5-base (search), paraphrase-multilingual-mpnet (recommendations)
- **Database**: PostgreSQL 16 with pgvector extension
- **Deployment**: Docker Compose

## Data

- **24,063 perfumes** from Fragrantica
- Russian translations for notes, accords, brands
- Pre-computed embeddings in `data/processed/perfumes_with_embeddings.parquet` (584MB)


