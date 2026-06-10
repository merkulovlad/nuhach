#!/usr/bin/env python3
"""
Ingest perfume data into normalized PostgreSQL tables.

This script reads from dataset_final.csv and populates:
- brands + brand_translations
- notes + note_translations
- accords + accord_translations
- perfumers
- perfumes_normalized
- perfume_top_notes, perfume_middle_notes, perfume_base_notes
- perfume_accords
- perfume_perfumers
"""

import os

import pandas as pd
import psycopg
from dotenv import load_dotenv

load_dotenv()

# === CONFIGURATION ===
DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://admin:admin@localhost:5432/nuhach")
CSV_PATH = os.getenv("CSV_PATH", "../data/processed/dataset_final.csv")
BATCH_SIZE = 500


def parse_text(value) -> str | None:
    """Convert value to cleaned string or None."""
    if pd.isna(value):
        return None
    s = str(value).strip()
    return s if s and s.lower() != "unknown" else None


def parse_int(value) -> int | None:
    """Convert value to int or None."""
    if pd.isna(value):
        return None
    try:
        return int(float(value))
    except (ValueError, TypeError):
        return None


def parse_rating(value) -> float | None:
    """Convert European format rating (e.g., '1,42') to float."""
    if pd.isna(value) or value is None:
        return None
    try:
        if isinstance(value, str):
            value = value.replace(",", ".")
        return float(value)
    except (ValueError, TypeError):
        return None


def split_notes(notes_str: str) -> list[str]:
    """Split comma-separated notes into a list of individual notes."""
    if not notes_str or pd.isna(notes_str):
        return []
    # Split by comma and clean each note
    notes = [n.strip() for n in str(notes_str).split(",")]
    return [n for n in notes if n and n.lower() != "unknown"]


def extract_unique_values(df: pd.DataFrame, columns: list[str]) -> set[str]:
    """Extract unique non-null values from multiple columns."""
    values = set()
    for col in columns:
        if col in df.columns:
            for val in df[col].dropna().unique():
                cleaned = parse_text(val)
                if cleaned:
                    values.add(cleaned)
    return values


def extract_unique_notes(df: pd.DataFrame, columns: list[str]) -> set[str]:
    """Extract unique notes from comma-separated note columns."""
    notes = set()
    for col in columns:
        if col in df.columns:
            for val in df[col].dropna():
                for note in split_notes(val):
                    if note:
                        notes.add(note)
    return notes


class NormalizedIngestor:
    """Handles ingestion of perfume data into normalized tables."""

    def __init__(self, conn: psycopg.Connection):
        self.conn = conn
        self.brand_cache: dict[str, int] = {}  # name -> id
        self.note_cache: dict[str, int] = {}  # name -> id
        self.accord_cache: dict[str, int] = {}  # name -> id
        self.perfumer_cache: dict[str, int] = {}  # name -> id

    def truncate_tables(self):
        """Truncate all normalized tables for clean re-run."""
        print("Truncating existing data...")
        with self.conn.cursor() as cur:
            # Order matters due to foreign key constraints
            cur.execute("TRUNCATE TABLE perfume_accords CASCADE")
            cur.execute("TRUNCATE TABLE perfume_base_notes CASCADE")
            cur.execute("TRUNCATE TABLE perfume_middle_notes CASCADE")
            cur.execute("TRUNCATE TABLE perfume_top_notes CASCADE")
            cur.execute("TRUNCATE TABLE perfume_perfumers CASCADE")
            cur.execute("TRUNCATE TABLE perfumes_normalized RESTART IDENTITY CASCADE")
            cur.execute("TRUNCATE TABLE brand_translations CASCADE")
            cur.execute("TRUNCATE TABLE note_translations CASCADE")
            cur.execute("TRUNCATE TABLE accord_translations CASCADE")
            cur.execute("TRUNCATE TABLE brands RESTART IDENTITY CASCADE")
            cur.execute("TRUNCATE TABLE notes RESTART IDENTITY CASCADE")
            cur.execute("TRUNCATE TABLE accords RESTART IDENTITY CASCADE")
            cur.execute("TRUNCATE TABLE perfumers RESTART IDENTITY CASCADE")
        self.conn.commit()
        print("Tables truncated successfully")

    def ingest_brands(self, df: pd.DataFrame):
        """Extract and insert unique brands with translations."""
        print("Ingesting brands...")

        # Build brand -> (country, translation) mapping
        brand_data: dict[str, tuple[str | None, str | None]] = {}
        for _, row in df.iterrows():
            brand_en = parse_text(row.get("Brand_en"))
            if not brand_en:
                continue
            country = parse_text(row.get("Country_en"))
            brand_ru = parse_text(row.get("Brand_ru"))
            # Keep first country/translation we see
            if brand_en not in brand_data:
                brand_data[brand_en] = (country, brand_ru)

        with self.conn.cursor() as cur:
            for brand_name, (country, brand_ru) in brand_data.items():
                # Insert brand
                cur.execute(
                    """
                    INSERT INTO brands (name, country)
                    VALUES (%s, %s)
                    ON CONFLICT (name) DO UPDATE SET country = EXCLUDED.country
                    RETURNING id
                """,
                    (brand_name, country),
                )
                brand_id = cur.fetchone()[0]
                self.brand_cache[brand_name] = brand_id

                # Insert translation if available
                if brand_ru:
                    cur.execute(
                        """
                        INSERT INTO brand_translations (brand_id, translation_ru)
                        VALUES (%s, %s)
                        ON CONFLICT (brand_id) DO UPDATE SET translation_ru = EXCLUDED.translation_ru
                    """,
                        (brand_id, brand_ru),
                    )

        self.conn.commit()
        print(f"  Inserted {len(brand_data)} brands")

    def ingest_notes(self, df: pd.DataFrame):
        """Extract and insert unique notes with translations."""
        print("Ingesting notes...")

        # Build note_en -> note_ru mapping from the data
        note_translations: dict[str, str] = {}

        # Process each note column pair
        note_pairs = [
            ("Top_en", "Top_ru"),
            ("Middle_en", "Middle_ru"),
            ("Base_en", "Base_ru"),
        ]

        for _, row in df.iterrows():
            for col_en, col_ru in note_pairs:
                notes_en = split_notes(row.get(col_en))
                notes_ru = split_notes(row.get(col_ru))
                # Match notes by position
                for i, note_en in enumerate(notes_en):
                    note_en_lower = note_en.lower()
                    if i < len(notes_ru) and notes_ru[i] and note_en_lower not in note_translations:
                        note_translations[note_en_lower] = notes_ru[i]

        # Get all unique English notes
        all_notes_en = extract_unique_notes(df, ["Top_en", "Middle_en", "Base_en"])

        with self.conn.cursor() as cur:
            for note_en in all_notes_en:
                # Insert note
                cur.execute(
                    """
                    INSERT INTO notes (name)
                    VALUES (%s)
                    ON CONFLICT (name) DO NOTHING
                    RETURNING id
                """,
                    (note_en,),
                )
                result = cur.fetchone()
                if result:
                    note_id = result[0]
                else:
                    cur.execute("SELECT id FROM notes WHERE name = %s", (note_en,))
                    note_id = cur.fetchone()[0]

                self.note_cache[note_en.lower()] = note_id

                # Insert translation if available
                note_ru = note_translations.get(note_en.lower())
                if note_ru:
                    cur.execute(
                        """
                        INSERT INTO note_translations (note_id, translation_ru)
                        VALUES (%s, %s)
                        ON CONFLICT (note_id) DO UPDATE SET translation_ru = EXCLUDED.translation_ru
                    """,
                        (note_id, note_ru),
                    )

        self.conn.commit()
        print(f"  Inserted {len(all_notes_en)} notes")

    def ingest_accords(self, df: pd.DataFrame):
        """Extract and insert unique accords with translations."""
        print("Ingesting accords...")

        # Build accord_en -> accord_ru mapping
        accord_translations: dict[str, str] = {}

        accord_pairs = [
            ("mainaccord1_en", "mainaccord1_ru"),
            ("mainaccord2_en", "mainaccord2_ru"),
            ("mainaccord3_en", "mainaccord3_ru"),
            ("mainaccord4_en", "mainaccord4_ru"),
            ("mainaccord5_en", "mainaccord5_ru"),
        ]

        for _, row in df.iterrows():
            for col_en, col_ru in accord_pairs:
                accord_en = parse_text(row.get(col_en))
                accord_ru = parse_text(row.get(col_ru))
                if accord_en and accord_ru and accord_en.lower() not in accord_translations:
                    accord_translations[accord_en.lower()] = accord_ru

        # Get all unique English accords
        all_accords_en = extract_unique_values(df, [f"mainaccord{i}_en" for i in range(1, 6)])

        with self.conn.cursor() as cur:
            for accord_en in all_accords_en:
                # Insert accord
                cur.execute(
                    """
                    INSERT INTO accords (name)
                    VALUES (%s)
                    ON CONFLICT (name) DO NOTHING
                    RETURNING id
                """,
                    (accord_en,),
                )
                result = cur.fetchone()
                if result:
                    accord_id = result[0]
                else:
                    cur.execute("SELECT id FROM accords WHERE name = %s", (accord_en,))
                    accord_id = cur.fetchone()[0]

                self.accord_cache[accord_en.lower()] = accord_id

                # Insert translation if available
                accord_ru = accord_translations.get(accord_en.lower())
                if accord_ru:
                    cur.execute(
                        """
                        INSERT INTO accord_translations (accord_id, translation_ru)
                        VALUES (%s, %s)
                        ON CONFLICT (accord_id) DO UPDATE SET translation_ru = EXCLUDED.translation_ru
                    """,
                        (accord_id, accord_ru),
                    )

        self.conn.commit()
        print(f"  Inserted {len(all_accords_en)} accords")

    def ingest_perfumers(self, df: pd.DataFrame):
        """Extract and insert unique perfumers."""
        print("Ingesting perfumers...")

        perfumers = set()
        for col in ["Perfumer1", "Perfumer2"]:
            if col in df.columns:
                for val in df[col].dropna():
                    name = parse_text(val)
                    if name:
                        perfumers.add(name)

        with self.conn.cursor() as cur:
            for perfumer_name in perfumers:
                cur.execute(
                    """
                    INSERT INTO perfumers (name)
                    VALUES (%s)
                    ON CONFLICT (name) DO NOTHING
                    RETURNING id
                """,
                    (perfumer_name,),
                )
                result = cur.fetchone()
                if result:
                    perfumer_id = result[0]
                else:
                    cur.execute("SELECT id FROM perfumers WHERE name = %s", (perfumer_name,))
                    perfumer_id = cur.fetchone()[0]
                self.perfumer_cache[perfumer_name.lower()] = perfumer_id

        self.conn.commit()
        print(f"  Inserted {len(perfumers)} perfumers")

    def ingest_perfumes(self, df: pd.DataFrame):
        """Insert perfumes and all their relationships."""
        print("Ingesting perfumes...")

        total = len(df)
        inserted = 0

        with self.conn.cursor() as cur:
            for _idx, row in df.iterrows():
                url = parse_text(row.get("url"))
                perfume_name = parse_text(row.get("Perfume"))
                brand_en = parse_text(row.get("Brand_en"))

                if not url or not perfume_name:
                    continue

                # Get brand_id
                brand_id = self.brand_cache.get(brand_en) if brand_en else None

                # Insert perfume
                cur.execute(
                    """
                    INSERT INTO perfumes_normalized (
                        url, perfume_name, brand_id, gender, 
                        rating_value, rating_count, year
                    )
                    VALUES (%s, %s, %s, %s, %s, %s, %s)
                    ON CONFLICT (url) DO UPDATE SET
                        perfume_name = EXCLUDED.perfume_name,
                        brand_id = EXCLUDED.brand_id,
                        gender = EXCLUDED.gender,
                        rating_value = EXCLUDED.rating_value,
                        rating_count = EXCLUDED.rating_count,
                        year = EXCLUDED.year,
                        updated_at = CURRENT_TIMESTAMP
                    RETURNING id
                """,
                    (
                        url,
                        perfume_name,
                        brand_id,
                        parse_text(row.get("Gender_en")),
                        parse_rating(row.get("Rating Value")),
                        parse_int(row.get("Rating Count")),
                        parse_int(row.get("Year")),
                    ),
                )
                perfume_id = cur.fetchone()[0]

                # Insert perfumers
                for perfumer_col in ["Perfumer1", "Perfumer2"]:
                    perfumer_name = parse_text(row.get(perfumer_col))
                    if perfumer_name:
                        perfumer_id = self.perfumer_cache.get(perfumer_name.lower())
                        if perfumer_id:
                            cur.execute(
                                """
                                INSERT INTO perfume_perfumers (perfume_id, perfumer_id)
                                VALUES (%s, %s)
                                ON CONFLICT DO NOTHING
                            """,
                                (perfume_id, perfumer_id),
                            )

                # Insert notes (top, middle, base)
                note_tables = [
                    ("Top_en", "perfume_top_notes"),
                    ("Middle_en", "perfume_middle_notes"),
                    ("Base_en", "perfume_base_notes"),
                ]
                for col, table in note_tables:
                    notes = split_notes(row.get(col))
                    for note_name in notes:
                        note_id = self.note_cache.get(note_name.lower())
                        if note_id:
                            cur.execute(
                                f"""
                                INSERT INTO {table} (perfume_id, note_id)
                                VALUES (%s, %s)
                                ON CONFLICT DO NOTHING
                            """,
                                (perfume_id, note_id),
                            )

                # Insert accords with position
                for i in range(1, 6):
                    accord_name = parse_text(row.get(f"mainaccord{i}_en"))
                    if accord_name:
                        accord_id = self.accord_cache.get(accord_name.lower())
                        if accord_id:
                            cur.execute(
                                """
                                INSERT INTO perfume_accords (perfume_id, accord_id, position)
                                VALUES (%s, %s, %s)
                                ON CONFLICT DO NOTHING
                            """,
                                (perfume_id, accord_id, i),
                            )

                inserted += 1
                if inserted % 1000 == 0:
                    self.conn.commit()
                    print(f"  Progress: {inserted}/{total} perfumes")

            self.conn.commit()

        print(f"  Inserted {inserted} perfumes")


def main():
    """Main function to orchestrate normalized data ingestion."""
    import argparse

    parser = argparse.ArgumentParser(
        description="Ingest perfume data into normalized PostgreSQL tables"
    )
    parser.add_argument(
        "--no-truncate", action="store_true", help="Skip truncating tables (append mode)"
    )
    parser.add_argument("--csv", type=str, default=CSV_PATH, help="Path to CSV file")

    args = parser.parse_args()

    csv_path = args.csv
    if not os.path.isabs(csv_path):
        csv_path = os.path.join(os.path.dirname(__file__), csv_path)

    print(f"Loading CSV from: {csv_path}")
    df = pd.read_csv(csv_path)
    print(f"Loaded {len(df)} rows from CSV")

    db_display = DATABASE_URL.split("@")[-1] if "@" in DATABASE_URL else DATABASE_URL
    print(f"Connecting to database: {db_display}")

    with psycopg.connect(DATABASE_URL) as conn:
        ingestor = NormalizedIngestor(conn)

        if not args.no_truncate:
            ingestor.truncate_tables()

        # Ingest in order (respecting foreign key constraints)
        ingestor.ingest_brands(df)
        ingestor.ingest_notes(df)
        ingestor.ingest_accords(df)
        ingestor.ingest_perfumers(df)
        ingestor.ingest_perfumes(df)

    print("\n" + "=" * 60)
    print("NORMALIZED INGESTION COMPLETED SUCCESSFULLY")
    print("=" * 60)


if __name__ == "__main__":
    main()
