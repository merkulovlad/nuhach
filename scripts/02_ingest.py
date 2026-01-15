#!/usr/bin/env python3
import os
import json
import pandas as pd
import psycopg
from psycopg.rows import dict_row
from dotenv import load_dotenv
from opensearchpy import OpenSearch, helpers
from typing import List, Dict, Any

load_dotenv()

# === CONFIGURATION ===
DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://localhost:5432/nuhach")
CSV_PATH = os.getenv("CSV_PATH", "../data/processed/dataset_final.csv")
PARQUET_PATH = os.getenv("PARQUET_PATH", "../data/processed/perfumes_with_embeddings.parquet")
OPENSEARCH_HOST = os.getenv("OPENSEARCH_HOST", "localhost")
OPENSEARCH_PORT = int(os.getenv("OPENSEARCH_PORT", "9200"))
OPENSEARCH_INDEX = os.getenv("OPENSEARCH_INDEX", "perfumes")
BATCH_SIZE = 500

# === CSV COLUMN MAPPING ===
# Note: CSV has _en (English) and _ru (Russian) columns from notebook merge
# Numeric columns and perfumer names have no suffix (not translated)
COLUMN_MAPPING = {
    # English columns
    "Perfume": "perfume_name",
    "Brand_en": "brand",
    "Country_en": "country",
    "Gender_en": "gender",
    "Top_en": "top_notes",
    "Middle_en": "middle_notes",
    "Base_en": "base_notes",
    "mainaccord1_en": "main_accord1",
    "mainaccord2_en": "main_accord2",
    "mainaccord3_en": "main_accord3",
    "mainaccord4_en": "main_accord4",
    "mainaccord5_en": "main_accord5",
    "Perfumer1": "perfumer1",
    "Perfumer2": "perfumer2",
    "url": "url",
    "Year": "year",
    "Rating Value": "rating_value",
    "Rating Count": "rating_count",
    # Russian translations
    "Brand_ru": "brand_ru",
    "Country_ru": "country_ru",
    "Gender_ru": "gender_ru",
    "Top_ru": "top_notes_ru",
    "Middle_ru": "middle_notes_ru",
    "Base_ru": "base_notes_ru",
    "mainaccord1_ru": "main_accord1_ru",
    "mainaccord2_ru": "main_accord2_ru",
    "mainaccord3_ru": "main_accord3_ru",
    "mainaccord4_ru": "main_accord4_ru",
    "mainaccord5_ru": "main_accord5_ru",
}

DB_COLUMNS = [
    "url",
    "perfume_name",
    "brand",
    "country",
    "gender",
    "rating_value",
    "rating_count",
    "year",
    "top_notes",
    "middle_notes",
    "base_notes",
    "perfumer1",
    "perfumer2",
    "main_accord1",
    "main_accord2",
    "main_accord3",
    "main_accord4",
    "main_accord5",
]


def parse_text_value(value):
    """Convert value to string or None."""
    if pd.isna(value):
        return None
    s = str(value).strip()
    return s if s else None


def parse_decimal(value):
    """Convert value to decimal/float or None."""
    if pd.isna(value):
        return None
    try:
        return float(value)
    except (ValueError, TypeError):
        return None


def parse_int(value):
    """Convert value to int or None."""
    if pd.isna(value):
        return None
    try:
        return int(value)
    except (ValueError, TypeError):
        return None


def parse_rating_value(value):
    """Convert European format rating (e.g., '1,42') to float."""
    if pd.isna(value) or value is None:
        return None
    try:
        # Handle European format with comma as decimal separator
        if isinstance(value, str):
            value = value.replace(',', '.')
        return float(value)
    except (ValueError, TypeError):
        return None


def combine_notes(top, middle, base):
    """Combine top, middle, and base notes into a single string."""
    notes = []
    if pd.notna(top) and str(top).strip():
        notes.append(str(top).strip())
    if pd.notna(middle) and str(middle).strip():
        notes.append(str(middle).strip())
    if pd.notna(base) and str(base).strip():
        notes.append(str(base).strip())
    return ', '.join(notes) if notes else None


def combine_accords(row, lang='en'):
    """Combine accords from mainaccord1-5 columns."""
    suffix = f'_{lang}'
    accords = []
    for i in range(1, 6):
        col_name = f'mainaccord{i}{suffix}'
        if col_name in row and pd.notna(row[col_name]) and str(row[col_name]).strip():
            accords.append(str(row[col_name]).strip())
    return ', '.join(accords) if accords else None


def combine_perfumers(row):
    """Combine perfumer names."""
    perfumers = []
    for col in ['Perfumer1', 'Perfumer2']:
        if col in row and pd.notna(row[col]) and str(row[col]).strip() and str(row[col]).strip().lower() != 'unknown':
            perfumers.append(str(row[col]).strip())
    return ', '.join(perfumers) if perfumers else None


def prepare_opensearch_doc(row: pd.Series, doc_id: int) -> Dict[str, Any]:
    """Transform a parquet row into an OpenSearch document according to the schema."""
    doc = {
        'id': doc_id,
        'url': str(row['url']) if pd.notna(row['url']) else None,
        'name': str(row['Perfume']) if pd.notna(row['Perfume']) else None,
        
        # Brand (both languages)
        'brand_en': str(row['Brand_en']) if pd.notna(row['Brand_en']) else None,
        'brand_ru': str(row['Brand_ru']) if pd.notna(row['Brand_ru']) else None,
        
        # Gender (both languages)
        'gender_en': str(row['Gender_en']) if pd.notna(row['Gender_en']) else None,
        'gender_ru': str(row['Gender_ru']) if pd.notna(row['Gender_ru']) else None,
        
        # Year
        'year': parse_int(row['Year']),
        
        # Ratings
        'rating_value': parse_rating_value(row['Rating Value']),
        'rating_count': parse_int(row['Rating Count']),
        
        # Notes - combine top, middle, base for each language
        'notes_en': combine_notes(row.get('Top_en'), row.get('Middle_en'), row.get('Base_en')),
        'notes_ru': combine_notes(row.get('Top_ru'), row.get('Middle_ru'), row.get('Base_ru')),
        
        # Accords - combine mainaccord1-5 for each language
        'accords_en': combine_accords(row, 'en'),
        'accords_ru': combine_accords(row, 'ru'),
        
        # Perfumers
        'perfumers_en': combine_perfumers(row),
    }
    
    # Remove None values to keep the document clean
    return {k: v for k, v in doc.items() if v is not None}


def create_opensearch_client() -> OpenSearch:
    """Create and return an OpenSearch client."""
    client = OpenSearch(
        hosts=[{'host': OPENSEARCH_HOST, 'port': OPENSEARCH_PORT}],
        http_compress=True,
        use_ssl=False,
        verify_certs=False,
        ssl_assert_hostname=False,
        ssl_show_warn=False
    )
    return client


def ingest_to_opensearch():
    """Load parquet data and ingest into OpenSearch."""
    print(f"Loading parquet file from: {PARQUET_PATH}")
    df = pd.read_parquet(PARQUET_PATH)
    print(f"Loaded {len(df)} rows from parquet file")
    
    # Create OpenSearch client
    print(f"Connecting to OpenSearch at {OPENSEARCH_HOST}:{OPENSEARCH_PORT}")
    client = create_opensearch_client()
    
    # Check if index exists
    if client.indices.exists(index=OPENSEARCH_INDEX):
        print(f"Index '{OPENSEARCH_INDEX}' already exists")
        # Optionally delete and recreate
        # client.indices.delete(index=OPENSEARCH_INDEX)
        # print(f"Deleted existing index '{OPENSEARCH_INDEX}'")
    else:
        print(f"Index '{OPENSEARCH_INDEX}' does not exist yet")
    
    # Prepare documents for bulk indexing
    print("Preparing documents for OpenSearch...")
    actions = []
    for idx, row in df.iterrows():
        doc = prepare_opensearch_doc(row, idx + 1)
        action = {
            '_index': OPENSEARCH_INDEX,
            '_id': idx + 1,
            '_source': doc
        }
        actions.append(action)
    
    # Bulk index documents
    print(f"Starting bulk indexing of {len(actions)} documents...")
    success_count = 0
    error_count = 0
    
    # Use helpers.bulk for efficient bulk indexing
    try:
        for ok, response in helpers.streaming_bulk(
            client,
            actions,
            chunk_size=BATCH_SIZE,
            raise_on_error=False
        ):
            if ok:
                success_count += 1
            else:
                error_count += 1
                print(f"Error indexing document: {response}")
            
            if (success_count + error_count) % 1000 == 0:
                print(f"Progress: {success_count} successful, {error_count} errors")
        
        print(f"\nIndexing complete!")
        print(f"Successfully indexed: {success_count} documents")
        print(f"Errors: {error_count} documents")
        
        # Refresh the index to make documents searchable immediately
        client.indices.refresh(index=OPENSEARCH_INDEX)
        print(f"Index '{OPENSEARCH_INDEX}' refreshed")
        
    except Exception as e:
        print(f"Error during bulk indexing: {e}")
        raise
    finally:
        client.close()


def prepare_row(row_dict):
    """Map CSV row to DB columns."""
    db_row = {col: None for col in DB_COLUMNS}

    for csv_col, value in row_dict.items():
        db_col = COLUMN_MAPPING.get(csv_col)
        
        if db_col is None:
            # Unmapped column, skip
            continue

        # Map to DB column with appropriate type conversion
        if db_col in ["top_notes", "middle_notes", "base_notes", "perfume_name", 
                      "brand", "country", "gender", "perfumer1", "perfumer2",
                      "main_accord1", "main_accord2", "main_accord3", 
                      "main_accord4", "main_accord5", "url"]:
            db_row[db_col] = parse_text_value(value)
        elif db_col == "year" or db_col == "rating_count":
            db_row[db_col] = parse_int(value)
        elif db_col == "rating_value":
            db_row[db_col] = parse_decimal(value)
        else:
            db_row[db_col] = value

    return db_row


def ingest_to_postgres():
    """Ingest data from CSV into PostgreSQL database."""
    print(f"Loading CSV from: {CSV_PATH}")
    df = pd.read_csv(CSV_PATH)
    print(f"Loaded {len(df)} rows from CSV")

    # Get CSV columns that are mapped
    mapped_cols = set(COLUMN_MAPPING.keys())
    actual_cols = set(df.columns)
    unmapped = actual_cols - mapped_cols
    if unmapped:
        print(f"Unmapped CSV columns (will be ignored): {unmapped}")

    # Prepare rows
    print("Preparing data...")
    rows_to_insert = []
    for _, row in df.iterrows():
        db_row = prepare_row(row.to_dict())
        rows_to_insert.append(db_row)

    print(f"Connecting to database: {DATABASE_URL.split('@')[-1] if '@' in DATABASE_URL else DATABASE_URL}")

    with psycopg.connect(DATABASE_URL) as conn:
        with conn.cursor(row_factory=dict_row) as cur:
            # Optional: Truncate table for clean re-run
            # cur.execute("TRUNCATE TABLE perfumes RESTART IDENTITY CASCADE")
            # conn.commit()

            insert_sql = f"""
                INSERT INTO perfumes (
                    url, perfume_name, brand, country, gender,
                    rating_value, rating_count, year,
                    top_notes, middle_notes, base_notes,
                    perfumer1, perfumer2,
                    main_accord1, main_accord2, main_accord3, 
                    main_accord4, main_accord5
                ) VALUES (
                    %(url)s, %(perfume_name)s, %(brand)s, %(country)s, %(gender)s,
                    %(rating_value)s, %(rating_count)s, %(year)s,
                    %(top_notes)s, %(middle_notes)s, %(base_notes)s,
                    %(perfumer1)s, %(perfumer2)s,
                    %(main_accord1)s, %(main_accord2)s, %(main_accord3)s,
                    %(main_accord4)s, %(main_accord5)s
                )
                ON CONFLICT (url) DO UPDATE SET
                    perfume_name = EXCLUDED.perfume_name,
                    brand = EXCLUDED.brand,
                    country = EXCLUDED.country,
                    gender = EXCLUDED.gender,
                    rating_value = EXCLUDED.rating_value,
                    rating_count = EXCLUDED.rating_count,
                    year = EXCLUDED.year,
                    top_notes = EXCLUDED.top_notes,
                    middle_notes = EXCLUDED.middle_notes,
                    base_notes = EXCLUDED.base_notes,
                    perfumer1 = EXCLUDED.perfumer1,
                    perfumer2 = EXCLUDED.perfumer2,
                    main_accord1 = EXCLUDED.main_accord1,
                    main_accord2 = EXCLUDED.main_accord2,
                    main_accord3 = EXCLUDED.main_accord3,
                    main_accord4 = EXCLUDED.main_accord4,
                    main_accord5 = EXCLUDED.main_accord5,
                    updated_at = CURRENT_TIMESTAMP
            """

            total = len(rows_to_insert)
            inserted = 0

            for i in range(0, total, BATCH_SIZE):
                batch = rows_to_insert[i:i + BATCH_SIZE]
                cur.executemany(insert_sql, batch)
                inserted += len(batch)
                print(f"Progress: {inserted}/{total} rows")

            conn.commit()

    print(f"Successfully loaded {inserted} rows into perfumes table")


def main():
    """Main function to orchestrate data ingestion."""
    import argparse
    
    parser = argparse.ArgumentParser(description='Ingest perfume data into PostgreSQL and/or OpenSearch')
    parser.add_argument('--postgres', action='store_true', help='Ingest into PostgreSQL')
    parser.add_argument('--opensearch', action='store_true', help='Ingest into OpenSearch')
    parser.add_argument('--all', action='store_true', help='Ingest into both PostgreSQL and OpenSearch')
    
    args = parser.parse_args()
    
    # If no arguments provided, show help
    if not (args.postgres or args.opensearch or args.all):
        parser.print_help()
        print("\n" + "="*60)
        print("No target specified. Please choose one of the following:")
        print("  --postgres    : Ingest data into PostgreSQL")
        print("  --opensearch  : Ingest data into OpenSearch")
        print("  --all         : Ingest data into both systems")
        print("="*60)
        return
    
    try:
        if args.postgres or args.all:
            print("\n" + "="*60)
            print("INGESTING INTO POSTGRESQL")
            print("="*60)
            ingest_to_postgres()
        
        if args.opensearch or args.all:
            print("\n" + "="*60)
            print("INGESTING INTO OPENSEARCH")
            print("="*60)
            ingest_to_opensearch()
        
        print("\n" + "="*60)
        print("ALL INGESTION TASKS COMPLETED SUCCESSFULLY")
        print("="*60)
    
    except Exception as e:
        print(f"\nError during ingestion: {e}")
        raise


if __name__ == "__main__":
    main()
