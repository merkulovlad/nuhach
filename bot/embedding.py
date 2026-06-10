"""
Query embedding service using sentence-transformers.

Uses intfloat/multilingual-e5-base for encoding user search queries.
Queries are prefixed with "query: " as per E5 model requirements.
"""

import logging

logger = logging.getLogger(__name__)

# Lazy loading to avoid import time overhead
_model = None
_model_name = "intfloat/multilingual-e5-base"


def _get_model():
    """Lazy load the embedding model."""
    global _model
    if _model is None:
        logger.info("Loading embedding model: %s", _model_name)
        try:
            from sentence_transformers import SentenceTransformer

            _model = SentenceTransformer(_model_name)
            logger.info(
                "Model loaded successfully (dim=%d)", _model.get_sentence_embedding_dimension()
            )
        except Exception as e:
            logger.error("Failed to load embedding model: %s", e)
            raise
    return _model


def preload_model() -> bool:
    """Pre-load the embedding model during startup.

    Returns:
        True if model loaded successfully, False otherwise.
    """
    try:
        _get_model()
        return True
    except Exception:
        return False


def embed_query(query: str) -> list[float] | None:
    """
    Embed a search query using multilingual-e5-base.

    E5 models require "query: " prefix for queries (documents use "passage: ").

    Args:
        query: User's search query text

    Returns:
        768-dimensional embedding as list of floats, or None on error
    """
    try:
        model = _get_model()

        # E5 requires "query: " prefix for queries
        prefixed_query = f"query: {query}"

        # Generate embedding
        embedding = model.encode(
            prefixed_query,
            convert_to_numpy=True,
            normalize_embeddings=True,  # Normalize for cosine similarity
        )

        return embedding.tolist()

    except Exception as e:
        logger.error("Failed to embed query '%s': %s", query[:50], e)
        return None


def embed_queries_batch(queries: list[str]) -> list[list[float]] | None:
    """
    Embed multiple search queries in batch.

    Args:
        queries: List of user search query texts

    Returns:
        List of 768-dimensional embeddings, or None on error
    """
    try:
        model = _get_model()

        # E5 requires "query: " prefix for queries
        prefixed_queries = [f"query: {q}" for q in queries]

        # Generate embeddings
        embeddings = model.encode(
            prefixed_queries, batch_size=32, convert_to_numpy=True, normalize_embeddings=True
        )

        return embeddings.tolist()

    except Exception as e:
        logger.error("Failed to embed queries batch: %s", e)
        return None


def is_model_loaded() -> bool:
    """Check if the embedding model is loaded."""
    return _model is not None


def get_embedding_dim() -> int:
    """Get the embedding dimension (768 for multilingual-e5-base)."""
    return 768
