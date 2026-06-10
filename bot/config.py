"""
Configuration loader for the Telegram bot.

Uses ONLY existing environment variables from .env.
"""

import os
from dataclasses import dataclass

from dotenv import load_dotenv

load_dotenv()


@dataclass
class Config:
    """Bot configuration loaded from environment variables."""

    # Telegram bot token (existing in .env as BOT_API_KEY)
    bot_token: str

    # API base URL (default to localhost:8080 as per docker-compose)
    api_base_url: str

    # Request timeout in seconds
    request_timeout: float

    # Retry settings
    max_retries: int
    retry_delay: float

    # Enable vector search (requires sentence-transformers)
    enable_vector_search: bool

    @classmethod
    def from_env(cls) -> "Config":
        """Load configuration from environment variables."""
        bot_token = os.getenv("BOT_API_KEY")
        if not bot_token:
            raise ValueError(
                "BOT_API_KEY environment variable is required. Please set it in your .env file."
            )

        # API URL: use API_BASE_URL if set, otherwise default to localhost:8080
        # (matches the docker-compose port mapping for the api service)
        api_base_url = os.getenv("API_BASE_URL", "http://localhost:8080")

        # Vector search: disable by default since it requires extra deps
        enable_vector_search = os.getenv("ENABLE_VECTOR_SEARCH", "false").lower() in (
            "true",
            "1",
            "yes",
        )

        return cls(
            bot_token=bot_token,
            api_base_url=api_base_url.rstrip("/"),
            request_timeout=float(os.getenv("BOT_REQUEST_TIMEOUT", "10.0")),
            max_retries=int(os.getenv("BOT_MAX_RETRIES", "3")),
            retry_delay=float(os.getenv("BOT_RETRY_DELAY", "0.5")),
            enable_vector_search=enable_vector_search,
        )


# Global config instance
config = Config.from_env()
