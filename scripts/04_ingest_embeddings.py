#!/usr/bin/env python3
"""
Ingest embeddings from parquet file into perfume_search table.

This script reads embeddings from the parquet file and inserts them into
the perfume_search table in PostgreSQL with both search and recommendation embeddings.
"""
import os
import json
import pandas as pd
import psycopg
from psycopg.rows import dict_row
from dotenv import load_dotenv
from typing import Optional, List
import argparse

load_dotenv()

# === CONFIGURATION ===
DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://admin:securepassword123@localhost:5432/nuhach")
PARQUET_PATH = os.getenv("PARQUET_PATH", "data/processed/perfumes_with_embeddings.parquet")
BATCH_SIZE = 100


def parse_embedding(value) -> Optional[List[float]]:
    """Parse embedding from string or list."""
    if pd.isna(value) or value is None:
        return None
    if isinstance(value, str):
        try:
            return json.loads(value)
        except json.JSONDecodeError:
            return None
    if isinstance(value, (list, tuple)):
        return list(value)
    return None


def create_search_text(row: pd.Series) -> str:
    """Create Russian search text from row data."""
    parts = []
    
    # Name
    if pd.notna(row.get('Perfume')):
        parts.append(str(row['Perfume']))
    
    # Brand (Russian if available)
    if pd.notna(row.get('Brand_ru')):
        parts.append(str(row['Brand_ru']))
    elif pd.notna(row.get('Brand_en')):
        parts.append(str(row['Brand_en']))
    
    # Notes (Russian if available)
    for note_col in ['Top_ru', 'Middle_ru', 'Base_ru']:
        if pd.notna(row.get(note_col)):
            parts.append(str(row[note_col]))
    
    # Accords (Russian if available)
    for i in range(1, 6):
        col = f'mainaccord{i}_ru'
        if pd.notna(row.get(col)):
            parts.append(str(row[col]))
    
    return ' '.join(parts)


def create_search_text_en(row: pd.Series) -> str:
    """Create English search text from row data."""
    parts = []
    
    # Name
    if pd.notna(row.get('Perfume')):
        parts.append(str(row['Perfume']))
    
    # Brand
    if pd.notna(row.get('Brand_en')):
        parts.append(str(row['Brand_en']))
    
    # Notes
    for note_col in ['Top_en', 'Middle_en', 'Base_en']:
        if pd.notna(row.get(note_col)):
            parts.append(str(row[note_col]))
    
    # Accords
    for i in range(1, 6):
        col = f'mainaccord{i}_en'
        if pd.notna(row.get(col)):
            parts.append(str(row[col]))
    
    # Perfumers
    for perfumer_col in ['Perfumer1', 'Perfumer2']:
        if pd.notna(row.get(perfumer_col)):
            parts.append(str(row[perfumer_col]))
    
    return ' '.join(parts)


def ingest_embeddings(parquet_path: str, truncate: bool = False):
    """Ingest embeddings from parquet into perfume_search table."""
    print(f"Loading parquet file: {parquet_path}")
    df = pd.read_parquet(parquet_path)
    print(f"Loaded {len(df)} rows")
    
    # Check for embedding columns
    has_search_emb = 'search_embedding' in df.columns
    has_rec_emb = 'rec_embedding' in df.columns
    
    print(f"Has search_embedding: {has_search_emb}")
    print(f"Has rec_embedding: {has_rec_emb}")
    
    if not has_search_emb and not has_rec_emb:
        print("ERROR: No embedding columns found in parquet file!")
        return
    
    # Validate embedding dimensions
    if has_search_emb:
        sample_emb = parse_embedding(df['search_embedding'].iloc[0])
        if sample_emb:
            print(f"Search embedding dimension: {len(sample_emb)}")
    
    if has_rec_emb:
        sample_emb = parse_embedding(df['rec_embedding'].iloc[0])
        if sample_emb:
            print(f"Rec embedding dimension: {len(sample_emb)}")
    
    print(f"\nConnecting to database...")
    
    with psycopg.connect(DATABASE_URL) as conn:
        with conn.cursor(row_factory=dict_row) as cur:
            # Get mapping of URLs to perfume IDs
            print("Fetching perfume URL -> ID mapping...")
            cur.execute("SELECT id, url FROM perfumes_normalized")
            url_to_id = {row['url']: row['id'] for row in cur.fetchall()}
            print(f"Found {len(url_to_id)} perfumes in database")
            
            if truncate:
                print("Truncating perfume_search table...")
                cur.execute("TRUNCATE TABLE perfume_search RESTART IDENTITY CASCADE")
                conn.commit()
            
            # Prepare insert/upsert SQL
            insert_sql = """
                INSERT INTO perfume_search (
                    perfume_id, doc_id, search_text, search_text_en, 
                    embedding, rec_embedding
                ) VALUES (
                    %(perfume_id)s, %(doc_id)s, %(search_text)s, %(search_text_en)s,
                    %(embedding)s::vector, %(rec_embedding)s::vector
                )
                ON CONFLICT (doc_id) DO UPDATE SET
                    search_text = EXCLUDED.search_text,
                    search_text_en = EXCLUDED.search_text_en,
                    embedding = EXCLUDED.embedding,
                    rec_embedding = EXCLUDED.rec_embedding,
                    updated_at = CURRENT_TIMESTAMP
            """
            
            total = len(df)
            inserted = 0
            skipped = 0
            errors = 0
            
            batch = []
            
            for idx, row in df.iterrows():
                url = row.get('url')
                if url is None or url not in url_to_id:
                    skipped += 1
                    continue
                
                perfume_id = url_to_id[url]
                
                # Parse embeddings
                search_emb = parse_embedding(row.get('search_embedding')) if has_search_emb else None
                rec_emb = parse_embedding(row.get('rec_embedding')) if has_rec_emb else None
                
                # Convert to PostgreSQL array format
                search_emb_str = str(search_emb) if search_emb else None
                rec_emb_str = str(rec_emb) if rec_emb else None
                
                record = {
                    'perfume_id': perfume_id,
                    'doc_id': f"perfume_{perfume_id}",
                    'search_text': create_search_text(row),
                    'search_text_en': create_search_text_en(row),
                    'embedding': search_emb_str,
                    'rec_embedding': rec_emb_str,
                }
                
                batch.append(record)
                
                if len(batch) >= BATCH_SIZE:
                    try:
                        cur.executemany(insert_sql, batch)
                        conn.commit()
                        inserted += len(batch)
                        print(f"Progress: {inserted}/{total} (skipped: {skipped}, errors: {errors})")
                    except Exception as e:
                        print(f"Error inserting batch: {e}")
                        errors += len(batch)
                        conn.rollback()
                    batch = []
            
            # Insert remaining batch
            if batch:
                try:
                    cur.executemany(insert_sql, batch)
                    conn.commit()
                    inserted += len(batch)
                except Exception as e:
                    print(f"Error inserting final batch: {e}")
                    errors += len(batch)
                    conn.rollback()
            
            print(f"\n{'='*60}")
            print(f"EMBEDDING INGESTION COMPLETE")
            print(f"{'='*60}")
            print(f"Total rows in parquet: {total}")
            print(f"Inserted/updated: {inserted}")
            print(f"Skipped (no matching URL): {skipped}")
            print(f"Errors: {errors}")


def main():
    parser = argparse.ArgumentParser(description='Ingest embeddings into perfume_search table')
    parser.add_argument('--parquet', type=str, default=PARQUET_PATH,
                        help='Path to parquet file with embeddings')
    parser.add_argument('--truncate', action='store_true',
                        help='Truncate perfume_search table before inserting')
    
    args = parser.parse_args()
    
    ingest_embeddings(args.parquet, args.truncate)


if __name__ == "__main__":
    main()
