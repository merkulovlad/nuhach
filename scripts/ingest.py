#!/usr/bin/env python3
import os
import json
import pandas as pd
import psycopg
from psycopg.rows import dict_row
from dotenv import load_dotenv

load_dotenv()

# === CONFIGURATION ===
DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://localhost:5432/nuhach")
CSV_PATH = os.getenv("CSV_PATH", "../data/processed/dataset_final.csv")
BATCH_SIZE = 500

# === CSV COLUMN MAPPING ===
# Note: CSV has _x (original) and _y (translated) columns from notebook merge
# We use _x columns (original English data)
COLUMN_MAPPING = {
    "Perfume": "perfume_name",
    "Brand_x": "brand",
    "Country_x": "country",
    "Gender_x": "gender",
    "Top_x": "top_notes",
    "Middle_x": "middle_notes",
    "Base_x": "base_notes",
    "mainaccord1_x": "main_accord1",
    "mainaccord2_x": "main_accord2",
    "mainaccord3_x": "main_accord3",
    "mainaccord4_x": "main_accord4",
    "mainaccord5_x": "main_accord5",
    "Perfumer1_x": "perfumer1",
    "Perfumer2_x": "perfumer2",
    "url": "url",
    "Year_x": "year",
    "Rating Value_x": "rating_value",
    "Rating Count_x": "rating_count",
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


def main():
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


if __name__ == "__main__":
    main()
