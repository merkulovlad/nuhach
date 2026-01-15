# OpenSearch Ingestion

This document describes how to use the extended `02_ingest.py` script to populate the OpenSearch perfumes index.

## Prerequisites

1. **OpenSearch running**: Make sure OpenSearch is running (typically via Docker Compose)
2. **Index created**: Create the perfumes index using the schema in `opensearch_index.md`
3. **Dependencies installed**: Install required Python packages

## Setup

### 1. Install Dependencies

```bash
pip install -r requirements.txt
```

### 2. Create OpenSearch Index

First, create the index with the proper schema. You can use curl or any HTTP client:

```bash
curl -X PUT "http://localhost:9200/perfumes" \
  -H 'Content-Type: application/json' \
  -d @opensearch_index.md
```

Or use the OpenSearch Dashboard (usually at http://localhost:5601).

### 3. Configure Environment Variables

Create a `.env` file in the project root or set environment variables:

```bash
# PostgreSQL (optional, for --postgres mode)
DATABASE_URL=postgresql://user:password@localhost:5432/nuhach

# OpenSearch
OPENSEARCH_HOST=localhost
OPENSEARCH_PORT=9200
OPENSEARCH_INDEX=perfumes

# Data paths
PARQUET_PATH=data/processed/perfumes_with_embeddings.parquet
CSV_PATH=data/processed/dataset_final.csv

# Batch size
BATCH_SIZE=500
```

## Usage

The script now supports multiple ingestion targets:

### Ingest to OpenSearch only

```bash
python scripts/02_ingest.py --opensearch
```

### Ingest to PostgreSQL only

```bash
python scripts/02_ingest.py --postgres
```

### Ingest to both systems

```bash
python scripts/02_ingest.py --all
```

## Data Mapping

The script transforms parquet data to match the OpenSearch schema:

| Parquet Column | OpenSearch Field | Notes |
|---------------|------------------|-------|
| `url` | `url` | Perfume URL |
| `Perfume` | `name` | Perfume name |
| `Brand_en` | `brand_en` | Brand name (English) |
| `Brand_ru` | `brand_ru` | Brand name (Russian) |
| `Gender_en` | `gender_en` | Gender (English) |
| `Gender_ru` | `gender_ru` | Gender (Russian) |
| `Year` | `year` | Release year |
| `Rating Value` | `rating_value` | Rating (converted from "1,42" to 1.42) |
| `Rating Count` | `rating_count` | Number of ratings |
| `Top_en`, `Middle_en`, `Base_en` | `notes_en` | Combined notes (English) |
| `Top_ru`, `Middle_ru`, `Base_ru` | `notes_ru` | Combined notes (Russian) |
| `mainaccord1-5_en` | `accords_en` | Combined accords (English) |
| `mainaccord1-5_ru` | `accords_ru` | Combined accords (Russian) |
| `Perfumer1`, `Perfumer2` | `perfumers_en` | Combined perfumer names |

## Features

- ✅ Bulk indexing with configurable batch size
- ✅ Progress tracking during ingestion
- ✅ Error handling and reporting
- ✅ European number format conversion (comma to dot for decimals)
- ✅ Automatic data transformation and cleaning
- ✅ Index refresh after ingestion for immediate searchability

## Troubleshooting

### Connection Error

If you get a connection error:
- Check if OpenSearch is running: `curl http://localhost:9200`
- Verify the host and port in your environment variables

### Index Already Exists

If the index already exists and you want to recreate it:

```bash
# Delete the index
curl -X DELETE "http://localhost:9200/perfumes"

# Recreate it with the schema
curl -X PUT "http://localhost:9200/perfumes" \
  -H 'Content-Type: application/json' \
  -d @opensearch_index.md
```

### Indexing Errors

The script will report the number of successful and failed documents. Check the error messages for details.

## Verifying the Data

After ingestion, verify the data:

```bash
# Check index stats
curl -X GET "http://localhost:9200/perfumes/_stats"

# Count documents
curl -X GET "http://localhost:9200/perfumes/_count"

# Search for a sample document
curl -X GET "http://localhost:9200/perfumes/_search?q=*&size=1&pretty"
```
