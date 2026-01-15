#!/usr/bin/env python3
"""
Nuhach Telegram Bot - Main entry point.

Usage:
    python -m bot.main

Environment variables (from .env):
    BOT_API_KEY - Telegram bot token (required)
    API_BASE_URL - Backend API URL (default: http://localhost:8080)
"""
import asyncio
import logging
import sys
from pathlib import Path

from aiogram import Bot, Dispatcher
from aiogram.client.default import DefaultBotProperties
from aiogram.enums import ParseMode

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

from bot.config import config
from bot.api_client import APIClient
from bot.handlers import router, set_api_client, set_vector_search_enabled

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    handlers=[
        logging.StreamHandler(sys.stdout),
    ],
)
logger = logging.getLogger(__name__)


async def main() -> None:
    """Main bot entry point."""
    logger.info("Starting Nuhach Telegram Bot...")
    logger.info("API Base URL: %s", config.api_base_url)
    logger.info("Vector search: %s", "enabled" if config.enable_vector_search else "disabled")
    
    # Initialize API client
    api_client = APIClient(
        base_url=config.api_base_url,
        timeout=config.request_timeout,
        max_retries=config.max_retries,
        retry_delay=config.retry_delay,
    )
    
    # Check API health
    if await api_client.health_check():
        logger.info("✅ API health check passed")
    else:
        logger.warning("⚠️ API health check failed - bot will start anyway")
    
    # Set API client for handlers
    set_api_client(api_client)
    
    # Set vector search enabled/disabled
    set_vector_search_enabled(config.enable_vector_search)
    
    # Pre-load embedding model if vector search is enabled
    if config.enable_vector_search:
        logger.info("Pre-loading embedding model (this may take a while on first run)...")
        from bot.embedding import preload_model
        if preload_model():
            logger.info("✅ Embedding model ready")
        else:
            logger.warning("⚠️ Failed to load embedding model - disabling vector search")
            set_vector_search_enabled(False)
    
    # Initialize bot and dispatcher
    bot = Bot(
        token=config.bot_token,
        default=DefaultBotProperties(parse_mode=ParseMode.HTML),
    )
    dp = Dispatcher()
    
    # Register handlers
    dp.include_router(router)
    
    logger.info("Bot initialized, starting polling...")
    
    try:
        # Start polling
        await dp.start_polling(bot)
    finally:
        # Cleanup
        await api_client.close()
        await bot.session.close()
        logger.info("Bot stopped")


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("Bot stopped by user")
    except Exception as e:
        logger.error("Fatal error: %s", e)
        sys.exit(1)
