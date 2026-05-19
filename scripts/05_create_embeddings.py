#!/usr/bin/env python3
"""
Create search and recommendation embeddings parquet from dataset_final.csv.

The output file is consumed by scripts/04_ingest_embeddings.py and the
ingest-embeddings Docker Compose service.
"""

import argparse
import os
from pathlib import Path
from typing import Iterable, Optional

import pandas as pd


DEFAULT_INPUT = Path("data/processed/dataset_final.csv")
DEFAULT_OUTPUT = Path("data/processed/perfumes_with_embeddings.parquet")
DEFAULT_SEARCH_MODEL = "intfloat/multilingual-e5-base"
DEFAULT_REC_MODEL = "sentence-transformers/paraphrase-multilingual-mpnet-base-v2"


def present(value) -> bool:
    return value is not None and pd.notna(value) and str(value).strip() != ""


def create_search_text(row: pd.Series) -> str:
    """Create a rich document text for query-to-document search embeddings."""
    parts: list[str] = []

    if present(row.get("Perfume")):
        parts.append(f"Perfume: {row['Perfume']}")
    if present(row.get("Brand_en")):
        parts.append(f"Brand: {row['Brand_en']}")
    if present(row.get("Gender_en")):
        parts.append(f"Gender: {row['Gender_en']}")

    if present(row.get("Top_en")):
        parts.append(f"Top notes: {row['Top_en']}")
    if present(row.get("Middle_en")):
        parts.append(f"Middle notes: {row['Middle_en']}")
    if present(row.get("Base_en")):
        parts.append(f"Base notes: {row['Base_en']}")

    accords = collect_accords(row)
    if accords:
        parts.append(f"Accords: {', '.join(accords)}")

    return ". ".join(parts)


def create_rec_text(row: pd.Series) -> str:
    """Create notes-and-accords text for item-to-item recommendation embeddings."""
    parts: list[str] = []

    if present(row.get("Top_en")):
        parts.append(f"Top notes: {row['Top_en']}")
    if present(row.get("Middle_en")):
        parts.append(f"Middle notes: {row['Middle_en']}")
    if present(row.get("Base_en")):
        parts.append(f"Base notes: {row['Base_en']}")

    accords = collect_accords(row)
    if accords:
        parts.append(f"Accords: {', '.join(accords)}")

    return ". ".join(parts)


def collect_accords(row: pd.Series) -> list[str]:
    accords: list[str] = []
    for i in range(1, 6):
        value = row.get(f"mainaccord{i}_en")
        if present(value):
            accords.append(str(value))
    return accords


def choose_device(requested: str) -> str:
    if requested != "auto":
        return requested

    try:
        import torch

        if torch.cuda.is_available():
            return "cuda"
        if getattr(torch.backends, "mps", None) and torch.backends.mps.is_available():
            return "mps"
    except Exception:
        pass

    return "cpu"


def encode_texts(
    texts: Iterable[str],
    *,
    model_name: str,
    device: str,
    batch_size: int,
    prefix: Optional[str] = None,
) -> list[list[float]]:
    from sentence_transformers import SentenceTransformer

    model = SentenceTransformer(model_name, device=device)
    print(f"Loaded {model_name} on {device}, dim={model.get_sentence_embedding_dimension()}")

    prepared = [f"{prefix}{text}" if prefix else text for text in texts]
    embeddings = model.encode(
        prepared,
        batch_size=batch_size,
        show_progress_bar=True,
        convert_to_numpy=True,
        normalize_embeddings=True,
    )

    return embeddings.tolist()


def build_embeddings(
    input_path: Path,
    output_path: Path,
    *,
    batch_size: int,
    device: str,
    limit: Optional[int],
    force: bool,
    search_model: str,
    rec_model: str,
) -> None:
    if not input_path.exists():
        raise FileNotFoundError(f"Input CSV not found: {input_path}")
    if output_path.exists() and not force:
        raise FileExistsError(f"Output already exists: {output_path}. Pass --force to overwrite.")

    print(f"Loading CSV: {input_path}")
    df = pd.read_csv(input_path)
    if limit is not None:
        df = df.head(limit).copy()
    print(f"Loaded {len(df):,} rows")

    print("Creating text fields...")
    df["search_text"] = df.apply(create_search_text, axis=1)
    df["rec_text"] = df.apply(create_rec_text, axis=1)

    resolved_device = choose_device(device)

    print(f"Generating search embeddings with {search_model}...")
    df["search_embedding"] = encode_texts(
        df["search_text"].tolist(),
        model_name=search_model,
        device=resolved_device,
        batch_size=batch_size,
        prefix="passage: ",
    )

    print(f"Generating recommendation embeddings with {rec_model}...")
    df["rec_embedding"] = encode_texts(
        df["rec_text"].tolist(),
        model_name=rec_model,
        device=resolved_device,
        batch_size=batch_size,
    )

    output_path.parent.mkdir(parents=True, exist_ok=True)
    temp_path = output_path.with_suffix(output_path.suffix + ".tmp")

    print(f"Saving parquet: {output_path}")
    df.to_parquet(temp_path, index=False, engine="pyarrow")
    os.replace(temp_path, output_path)

    file_size = output_path.stat().st_size / 1024 / 1024
    print("Done")
    print(f"Rows: {len(df):,}")
    print(f"Output: {output_path}")
    print(f"Size: {file_size:.2f} MB")
    print(f"Search dim: {len(df['search_embedding'].iloc[0])}")
    print(f"Rec dim: {len(df['rec_embedding'].iloc[0])}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Create embeddings parquet for ingest-embeddings.")
    parser.add_argument("--csv", type=Path, default=DEFAULT_INPUT, help="Input dataset CSV path.")
    parser.add_argument("--output", type=Path, default=DEFAULT_OUTPUT, help="Output parquet path.")
    parser.add_argument("--batch-size", type=int, default=32, help="SentenceTransformer batch size.")
    parser.add_argument("--device", default="auto", help="Device: auto, cpu, cuda, mps.")
    parser.add_argument("--limit", type=int, default=None, help="Limit rows for a smoke test.")
    parser.add_argument("--force", action="store_true", help="Overwrite existing output file.")
    parser.add_argument("--search-model", default=DEFAULT_SEARCH_MODEL, help="Search embedding model.")
    parser.add_argument("--rec-model", default=DEFAULT_REC_MODEL, help="Recommendation embedding model.")
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    build_embeddings(
        args.csv,
        args.output,
        batch_size=args.batch_size,
        device=args.device,
        limit=args.limit,
        force=args.force,
        search_model=args.search_model,
        rec_model=args.rec_model,
    )


if __name__ == "__main__":
    main()
