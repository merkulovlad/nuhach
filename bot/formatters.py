"""
Message formatters for Telegram output.
"""
from html import escape
from typing import List, Optional
from .api_client import PerfumeItem, StoreOffer


def format_rating(rating_value: Optional[float], rating_count: Optional[int]) -> str:
    """Format rating as stars and count."""
    if rating_value is None:
        return "—"
    
    # Convert to 5-star scale visualization
    stars = "★" * int(round(rating_value)) + "☆" * (5 - int(round(rating_value)))
    
    count_str = ""
    if rating_count:
        if rating_count >= 1000:
            count_str = f" ({rating_count / 1000:.1f}k)"
        else:
            count_str = f" ({rating_count})"
    
    return f"{rating_value:.2f} {stars}{count_str}"


def format_perfume_card(perfume: PerfumeItem, index: Optional[int] = None) -> str:
    """Format a perfume as a short card for list display."""
    prefix = f"{index}. " if index is not None else ""
    
    year_str = f" ({perfume.year})" if perfume.year else ""
    rating_str = format_rating(perfume.rating_value, perfume.rating_count)
    
    lines = [
        f"{prefix}<b>{perfume.name}</b>{year_str}",
        f"🏷 {perfume.brand}",
        f"⭐ {rating_str}",
    ]
    
    if perfume.accords:
        lines.append(f"🎨 {perfume.accords[:50]}{'...' if len(perfume.accords) > 50 else ''}")
    
    return "\n".join(lines)


def format_perfume_list(
    perfumes: List[PerfumeItem],
    title: str = "Results",
    show_index: bool = True,
) -> str:
    """Format a list of perfumes for display."""
    if not perfumes:
        return f"<b>{title}</b>\n\nNo perfumes found."
    
    header = f"<b>{title}</b> ({len(perfumes)} items)\n"
    separator = "\n" + "─" * 20 + "\n"
    
    cards = []
    for i, perfume in enumerate(perfumes, 1):
        index = i if show_index else None
        cards.append(format_perfume_card(perfume, index))
    
    return header + separator.join(cards)


def format_perfume_details(perfume: PerfumeItem) -> str:
    """Format full perfume details."""
    year_str = f" ({perfume.year})" if perfume.year else ""
    rating_str = format_rating(perfume.rating_value, perfume.rating_count)
    
    lines = [
        f"<b>{perfume.name}</b>{year_str}",
        "",
        f"🏷 <b>Brand:</b> {perfume.brand}",
        f"⭐ <b>Rating:</b> {rating_str}",
    ]
    
    if perfume.gender:
        lines.append(f"👤 <b>Gender:</b> {perfume.gender}")
    
    if perfume.country:
        lines.append(f"🌍 <b>Country:</b> {perfume.country}")
    
    if perfume.perfumer:
        lines.append(f"👃 <b>Perfumer:</b> {perfume.perfumer}")
    
    lines.append("")  # Empty line before notes
    
    if perfume.top_notes:
        lines.append(f"🔝 <b>Top:</b> {perfume.top_notes}")
    
    if perfume.middle_notes:
        lines.append(f"💗 <b>Heart:</b> {perfume.middle_notes}")
    
    if perfume.base_notes:
        lines.append(f"🌲 <b>Base:</b> {perfume.base_notes}")
    
    if perfume.accords:
        lines.append("")
        lines.append(f"🎨 <b>Accords:</b> {perfume.accords}")
    
    if perfume.url:
        lines.append("")
        lines.append(f"🔗 <a href=\"{perfume.url}\">View on Fragrantica</a>")
    
    return "\n".join(lines)


def format_error_message(error: str = "default") -> str:
    """Format error message for user."""
    messages = {
        "default": "❌ Something went wrong. Please try again later.",
        "api_unavailable": "❌ The service is temporarily unavailable. Please try again in a few minutes.",
        "not_found": "❌ Perfume not found.",
        "no_query": "❌ Please provide a search query.\n\nUsage: /search <query>\nExample: /search rose vanilla",
        "no_results": "🔍 No perfumes found for your query. Try a different search term.",
        "no_recommendations": "🤔 No recommendations yet. Search for some perfumes and like them to get personalized recommendations!",
        "no_saves": "📑 You haven't saved any perfumes yet. Use the Save button to save perfumes you like!",
    }
    return messages.get(error, messages["default"])


def format_store_offers(offers: List[StoreOffer]) -> str:
    if not offers:
        return "<b>Предложения в магазинах</b>\n\nПодходящих предложений не найдено."

    lines = ["<b>Предложения в магазинах</b>", ""]
    for offer in offers[:5]:
        variant = " ".join(
            value for value in (
                offer.concentration or "",
                f"{offer.volume_ml} мл" if offer.volume_ml else "",
                "тестер" if offer.product_type == "tester" else "",
            ) if value
        )
        price = f"{offer.price:,.0f}".replace(",", " ")
        old_price = ""
        if offer.old_price and offer.old_price > offer.price:
            formatted_old = f"{offer.old_price:,.0f}".replace(",", " ")
            old_price = f" (обычная цена {formatted_old} {escape(offer.currency)})"
        lines.append(
            f"<b>{escape(offer.store)}</b> — {price} {escape(offer.currency)}{old_price}"
        )
        lines.append(f'<a href="{escape(offer.url, quote=True)}">{escape(offer.title[:120])}</a>')
        if variant:
            lines.append(escape(variant))
        if offer.seller:
            lines.append(f"Продавец: {escape(offer.seller)}")
        if offer.comment:
            prefix = "⚠️ " if offer.risk_level in {"medium", "high"} else ""
            lines.append(prefix + escape(offer.comment[:250]))
        lines.append("")
    lines.append("Оценка риска основана на косвенных признаках и не подтверждает подделку.")
    return "\n".join(lines)
