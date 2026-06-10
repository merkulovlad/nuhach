"""
Telegram bot command and callback handlers.
"""
import asyncio
import logging
from typing import Optional, List

from aiogram import Bot, Dispatcher, Router, F
from aiogram.filters import Command, CommandStart
from aiogram.types import Message, CallbackQuery
from aiogram.enums import ParseMode

from .api_client import APIClient, APIError, PerfumeItem
from .state import state_manager
from .formatters import (
    format_perfume_list,
    format_perfume_details,
    format_perfume_card,
    format_error_message,
    format_store_offers,
)
from .keyboards import (
    build_perfume_keyboard,
    build_perfume_list_keyboard,
    build_detail_keyboard,
    build_offers_keyboard,
    parse_callback_data,
)

logger = logging.getLogger(__name__)

# Create router
router = Router()

# API client will be set during bot initialization
api_client: Optional[APIClient] = None

# Vector search flag
_enable_vector_search: bool = False


def set_api_client(client: APIClient) -> None:
    """Set the API client instance."""
    global api_client
    api_client = client


def set_vector_search_enabled(enabled: bool) -> None:
    """Enable or disable vector search."""
    global _enable_vector_search
    _enable_vector_search = enabled
    if enabled:
        logger.info("Vector search enabled")


def _get_query_embedding(query: str) -> Optional[List[float]]:
    """Get embedding for search query if vector search is enabled."""
    if not _enable_vector_search:
        return None
    try:
        from .embedding import embed_query
        return embed_query(query)
    except Exception as e:
        logger.warning("Failed to get query embedding: %s", e)
        return None


def get_tg_id(message_or_callback) -> int:
    """Extract Telegram user ID."""
    if hasattr(message_or_callback, "from_user"):
        return message_or_callback.from_user.id
    return 0


# ============================================================================
# Command Handlers
# ============================================================================

@router.message(CommandStart())
async def cmd_start(message: Message) -> None:
    """Handle /start command."""
    welcome_text = """
<b>Добро пожаловать в Nuhach Perfume Bot 👃</b>

Я не сомелье и не парфюмерный бог.
Я — нюхач.
Понюхал, прикинул, сказал как есть.

Могу подсказать аромат,
могу отговорить от херни,
могу просто поныть, что раньше трава была пахучее.

<b>Чё тут есть:</b>

/search &lt;запрос&gt; — ищем духи, если ты хоть что-то помнишь

/recommend — скажу, что взять, если сам не знаешь чего хочешь

/saves — твои отложенные пузырьки

/help — если всё пошло по пизде и ты забыл команды

Гарантий ноль.
Вкусы — субъективные.
Советы — как от знакомого во дворе.

Ну чё, давай нюхать 👃
"""
    await message.answer(welcome_text, parse_mode=ParseMode.HTML)


@router.message(Command("help"))
async def cmd_help(message: Message) -> None:
    """Handle /help command."""
    await cmd_start(message)


@router.message(Command("search"))
async def cmd_search(message: Message) -> None:
    """Handle /search <query> command."""
    if api_client is None:
        await message.answer(format_error_message("api_unavailable"))
        return
    
    # Extract query from message
    parts = message.text.split(maxsplit=1)
    if len(parts) < 2:
        await message.answer(format_error_message("no_query"), parse_mode=ParseMode.HTML)
        return
    
    query = parts[1].strip()
    if not query:
        await message.answer(format_error_message("no_query"), parse_mode=ParseMode.HTML)
        return
    
    tg_id = get_tg_id(message)
    
    # Send "searching" message
    status_msg = await message.answer("🔍 Searching...")
    
    try:
        # Try to get query embedding for semantic search
        embedding = _get_query_embedding(query)
        
        result = await api_client.search(
            query=query, 
            limit=10, 
            tg_id=tg_id,
            embedding=embedding
        )
        
        if not result.items:
            await status_msg.edit_text(format_error_message("no_results"))
            return
        
        # Save state for event logging
        perfume_ids = [p.id for p in result.items]
        state_manager.set_last_request(tg_id, result.request_id, perfume_ids)
        
        # Log impressions
        await api_client.log_impression(tg_id, perfume_ids, result.request_id)
        
        # Format and send results
        text = format_perfume_list(result.items, title=f"🔍 Search: {query}")
        keyboard = build_perfume_list_keyboard(result.items)
        
        await status_msg.edit_text(text, parse_mode=ParseMode.HTML, reply_markup=keyboard)
        
    except APIError as e:
        logger.error("Search API error: %s", e)
        await status_msg.edit_text(format_error_message("api_unavailable"))
    except Exception as e:
        logger.error("Search error: %s", e)
        await status_msg.edit_text(format_error_message())


@router.message(Command("recommend"))
async def cmd_recommend(message: Message) -> None:
    """Handle /recommend command."""
    if api_client is None:
        await message.answer(format_error_message("api_unavailable"))
        return
    
    tg_id = get_tg_id(message)
    
    status_msg = await message.answer("🎯 Getting recommendations...")
    
    try:
        result = await api_client.get_recommendations(tg_id=tg_id, limit=10)
        
        if not result.items:
            await status_msg.edit_text(format_error_message("no_recommendations"))
            return
        
        # Save state for event logging
        perfume_ids = [p.id for p in result.items]
        state_manager.set_last_request(tg_id, result.request_id, perfume_ids)
        
        # Log impressions
        await api_client.log_impression(tg_id, perfume_ids, result.request_id)
        
        # Format results
        title = "🎯 Recommendations for you"
        if result.exploration_ids:
            title += f" (🔮 exploring: {len(result.exploration_ids)})"
        
        text = format_perfume_list(result.items, title=title)
        keyboard = build_perfume_list_keyboard(result.items)
        
        await status_msg.edit_text(text, parse_mode=ParseMode.HTML, reply_markup=keyboard)
        
    except APIError as e:
        logger.error("Recommendations API error: %s", e)
        await status_msg.edit_text(format_error_message("api_unavailable"))
    except Exception as e:
        logger.error("Recommendations error: %s", e)
        await status_msg.edit_text(format_error_message())


@router.message(Command("saves"))
async def cmd_saves(message: Message) -> None:
    """Handle /saves command."""
    if api_client is None:
        await message.answer(format_error_message("api_unavailable"))
        return
    
    tg_id = get_tg_id(message)
    
    status_msg = await message.answer("💾 Loading saves...")
    
    try:
        saves = await api_client.get_saves(tg_id)
        
        if not saves:
            await status_msg.edit_text(format_error_message("no_saves"))
            return
        
        text = format_perfume_list(saves, title="💾 Your saved perfumes")
        keyboard = build_perfume_list_keyboard(saves)
        
        await status_msg.edit_text(text, parse_mode=ParseMode.HTML, reply_markup=keyboard)
        
    except APIError as e:
        logger.error("Saves API error: %s", e)
        await status_msg.edit_text(format_error_message("api_unavailable"))
    except Exception as e:
        logger.error("Saves error: %s", e)
        await status_msg.edit_text(format_error_message())


# ============================================================================
# Callback Handlers
# ============================================================================

@router.callback_query(F.data.startswith("select:"))
async def callback_select(callback: CallbackQuery) -> None:
    """Handle perfume selection from list."""
    await callback_details(callback)


@router.callback_query(F.data.startswith("details:"))
async def callback_details(callback: CallbackQuery) -> None:
    """Handle Details button click."""
    if api_client is None:
        await callback.answer("Service unavailable", show_alert=True)
        return
    
    _, perfume_id = parse_callback_data(callback.data)
    tg_id = get_tg_id(callback)
    request_id = state_manager.get_last_request_id(tg_id)
    
    # Log click event
    await api_client.create_event(
        tg_id=tg_id,
        perfume_id=perfume_id,
        event_type="click",
        request_id=request_id,
    )
    
    try:
        perfume = await api_client.get_perfume(perfume_id)
        
        if perfume is None:
            await callback.answer("Perfume not found", show_alert=True)
            return
        
        text = format_perfume_details(perfume)
        keyboard = build_detail_keyboard(perfume_id)
        
        await callback.message.edit_text(
            text, parse_mode=ParseMode.HTML, reply_markup=keyboard
        )
        await callback.answer()
        
    except Exception as e:
        logger.error("Details error: %s", e)
        await callback.answer("Failed to load details", show_alert=True)


@router.callback_query(F.data.startswith("similar:"))
async def callback_similar(callback: CallbackQuery) -> None:
    """Handle Similar button click."""
    if api_client is None:
        await callback.answer("Service unavailable", show_alert=True)
        return
    
    _, perfume_id = parse_callback_data(callback.data)
    tg_id = get_tg_id(callback)
    
    try:
        result = await api_client.get_similar(perfume_id=perfume_id, limit=10, tg_id=tg_id)
        
        if not result.items:
            await callback.answer("No similar perfumes found", show_alert=True)
            return
        
        # Save state for event logging
        perfume_ids = [p.id for p in result.items]
        state_manager.set_last_request(tg_id, result.request_id, perfume_ids)
        
        # Log impressions
        await api_client.log_impression(tg_id, perfume_ids, result.request_id)
        
        text = format_perfume_list(result.items, title="🔄 Similar perfumes")
        keyboard = build_perfume_list_keyboard(result.items)
        
        await callback.message.edit_text(
            text, parse_mode=ParseMode.HTML, reply_markup=keyboard
        )
        await callback.answer()
        
    except Exception as e:
        logger.error("Similar error: %s", e)
        await callback.answer("Failed to load similar perfumes", show_alert=True)


@router.callback_query(F.data.startswith("offers:"))
@router.callback_query(F.data.startswith("refresh_offers:"))
async def callback_offers(callback: CallbackQuery) -> None:
    """Start an on-demand store search or return fresh cached offers."""
    if api_client is None:
        await callback.answer("Service unavailable", show_alert=True)
        return

    action, perfume_id = parse_callback_data(callback.data)
    await callback.answer()
    await callback.message.edit_text("Ищу наличие в разрешённых магазинах…")

    try:
        result = await api_client.search_offers(
            perfume_id,
            force=action == "refresh_offers",
        )
        for _ in range(10):
            if result.status not in {"searching", "refreshing"}:
                break
            await asyncio.sleep(2)
            result = await api_client.get_offers(perfume_id)

        keyboard = build_offers_keyboard(perfume_id)
        if result.offers:
            await callback.message.edit_text(
                format_store_offers(result.offers),
                parse_mode=ParseMode.HTML,
                reply_markup=keyboard,
                disable_web_page_preview=True,
            )
        elif result.status == "failed":
            await callback.message.edit_text(
                "Автоматический поиск сейчас недоступен: правила источников не разрешают "
                "обход нужных страниц или не удалось безопасно проверить данные.",
                reply_markup=keyboard,
            )
        elif result.status in {"searching", "refreshing"}:
            await callback.message.edit_text(
                "Поиск продолжается. Проверьте предложения немного позже.",
                reply_markup=keyboard,
            )
        else:
            await callback.message.edit_text(
                "В разрешённых источниках предложений не найдено.",
                reply_markup=keyboard,
            )
    except Exception as e:
        logger.error("Offers error: %s", e)
        await callback.message.edit_text("Не удалось получить предложения в магазинах.")


@router.callback_query(F.data.startswith("like:"))
async def callback_like(callback: CallbackQuery) -> None:
    """Handle Like button click."""
    if api_client is None:
        await callback.answer("Service unavailable", show_alert=True)
        return
    
    _, perfume_id = parse_callback_data(callback.data)
    tg_id = get_tg_id(callback)
    request_id = state_manager.get_last_request_id(tg_id)
    
    success = await api_client.create_event(
        tg_id=tg_id,
        perfume_id=perfume_id,
        event_type="like",
        request_id=request_id,
    )
    
    if success:
        await callback.answer("👍 Liked! This will improve your recommendations.")
    else:
        await callback.answer("Failed to save like", show_alert=True)


@router.callback_query(F.data.startswith("dislike:"))
async def callback_dislike(callback: CallbackQuery) -> None:
    """Handle Dislike button click."""
    if api_client is None:
        await callback.answer("Service unavailable", show_alert=True)
        return
    
    _, perfume_id = parse_callback_data(callback.data)
    tg_id = get_tg_id(callback)
    request_id = state_manager.get_last_request_id(tg_id)
    
    success = await api_client.create_event(
        tg_id=tg_id,
        perfume_id=perfume_id,
        event_type="dislike",
        request_id=request_id,
    )
    
    if success:
        await callback.answer("👎 Noted. We'll show you less like this.")
    else:
        await callback.answer("Failed to save dislike", show_alert=True)


@router.callback_query(F.data.startswith("save:"))
async def callback_save(callback: CallbackQuery) -> None:
    """Handle Save button click."""
    if api_client is None:
        await callback.answer("Service unavailable", show_alert=True)
        return
    
    _, perfume_id = parse_callback_data(callback.data)
    tg_id = get_tg_id(callback)
    request_id = state_manager.get_last_request_id(tg_id)
    
    success = await api_client.create_event(
        tg_id=tg_id,
        perfume_id=perfume_id,
        event_type="save",
        request_id=request_id,
    )
    
    if success:
        await callback.answer("💾 Saved! View your saves with /saves")
    else:
        await callback.answer("Failed to save", show_alert=True)


@router.callback_query(F.data == "back")
async def callback_back(callback: CallbackQuery) -> None:
    """Handle Back button - show recommendations."""
    # For simplicity, show recommendations on back
    await callback.answer()
    
    # Create a fake message to reuse cmd_recommend
    if api_client is None:
        await callback.answer("Service unavailable", show_alert=True)
        return
    
    tg_id = get_tg_id(callback)
    
    try:
        result = await api_client.get_recommendations(tg_id=tg_id, limit=10)
        
        if result.items:
            perfume_ids = [p.id for p in result.items]
            state_manager.set_last_request(tg_id, result.request_id, perfume_ids)
            
            text = format_perfume_list(result.items, title="🎯 Recommendations for you")
            keyboard = build_perfume_list_keyboard(result.items)
            
            await callback.message.edit_text(
                text, parse_mode=ParseMode.HTML, reply_markup=keyboard
            )
        else:
            await callback.message.edit_text(
                "Use /search to find perfumes!",
                parse_mode=ParseMode.HTML
            )
    except Exception as e:
        logger.error("Back button error: %s", e)


@router.callback_query(F.data.startswith("page:"))
async def callback_page(callback: CallbackQuery) -> None:
    """Handle pagination - currently not fully implemented."""
    await callback.answer("Pagination coming soon!")


@router.callback_query(F.data == "noop")
async def callback_noop(callback: CallbackQuery) -> None:
    """Handle no-op callbacks (like page counter)."""
    await callback.answer()
