"""
Inline keyboard builders for Telegram messages.
"""

from contextlib import suppress

from aiogram.types import InlineKeyboardButton, InlineKeyboardMarkup

from .api_client import PerfumeItem

# Callback data format: action:perfume_id
# Actions: details, similar, like, dislike, save


def build_perfume_keyboard(
    perfume: PerfumeItem,
    show_details: bool = True,
    show_similar: bool = True,
    show_reactions: bool = True,
) -> InlineKeyboardMarkup:
    """Build inline keyboard for a single perfume."""
    buttons = []

    row1 = []
    if show_details:
        row1.append(InlineKeyboardButton(text="📋 Details", callback_data=f"details:{perfume.id}"))
    if show_similar:
        row1.append(InlineKeyboardButton(text="🔄 Similar", callback_data=f"similar:{perfume.id}"))
    if row1:
        buttons.append(row1)

    if show_reactions:
        row2 = [
            InlineKeyboardButton(text="👍", callback_data=f"like:{perfume.id}"),
            InlineKeyboardButton(text="👎", callback_data=f"dislike:{perfume.id}"),
            InlineKeyboardButton(text="💾 Save", callback_data=f"save:{perfume.id}"),
        ]
        buttons.append(row2)

    return InlineKeyboardMarkup(inline_keyboard=buttons)


def build_perfume_list_keyboard(
    perfumes: list[PerfumeItem],
    page: int = 0,
    page_size: int = 5,
    show_pagination: bool = True,
) -> InlineKeyboardMarkup:
    """Build inline keyboard for a list of perfumes with selection buttons."""
    buttons = []

    # Individual perfume buttons (numbered)
    start_idx = page * page_size
    for i, perfume in enumerate(perfumes[start_idx : start_idx + page_size], start=1):
        buttons.append(
            [
                InlineKeyboardButton(
                    text=f"{i}. {perfume.name[:30]}{'...' if len(perfume.name) > 30 else ''}",
                    callback_data=f"select:{perfume.id}",
                )
            ]
        )

    # Pagination row
    if show_pagination and len(perfumes) > page_size:
        nav_row = []
        total_pages = (len(perfumes) + page_size - 1) // page_size

        if page > 0:
            nav_row.append(InlineKeyboardButton(text="◀️ Prev", callback_data=f"page:{page - 1}"))

        nav_row.append(InlineKeyboardButton(text=f"{page + 1}/{total_pages}", callback_data="noop"))

        if page < total_pages - 1:
            nav_row.append(InlineKeyboardButton(text="Next ▶️", callback_data=f"page:{page + 1}"))

        buttons.append(nav_row)

    return InlineKeyboardMarkup(inline_keyboard=buttons)


def build_detail_keyboard(perfume_id: int) -> InlineKeyboardMarkup:
    """Build keyboard for detail view."""
    return InlineKeyboardMarkup(
        inline_keyboard=[
            [
                InlineKeyboardButton(text="🔄 Similar", callback_data=f"similar:{perfume_id}"),
                InlineKeyboardButton(text="🛒 Где купить", callback_data=f"offers:{perfume_id}"),
            ],
            [
                InlineKeyboardButton(text="👍", callback_data=f"like:{perfume_id}"),
                InlineKeyboardButton(text="👎", callback_data=f"dislike:{perfume_id}"),
                InlineKeyboardButton(text="💾 Save", callback_data=f"save:{perfume_id}"),
            ],
            [
                InlineKeyboardButton(text="🔙 Back", callback_data="back"),
            ],
        ]
    )


def build_offers_keyboard(perfume_id: int) -> InlineKeyboardMarkup:
    return InlineKeyboardMarkup(
        inline_keyboard=[
            [
                InlineKeyboardButton(
                    text="🔄 Обновить цены", callback_data=f"refresh_offers:{perfume_id}"
                )
            ],
            [InlineKeyboardButton(text="📋 К аромату", callback_data=f"details:{perfume_id}")],
        ]
    )


def build_back_keyboard() -> InlineKeyboardMarkup:
    """Build keyboard with just a back button."""
    return InlineKeyboardMarkup(
        inline_keyboard=[
            [InlineKeyboardButton(text="🔙 Back", callback_data="back")],
        ]
    )


def parse_callback_data(callback_data: str) -> tuple:
    """Parse callback data into (action, value)."""
    if ":" in callback_data:
        action, value = callback_data.split(":", 1)
        with suppress(ValueError):
            value = int(value)
        return action, value
    return callback_data, None
