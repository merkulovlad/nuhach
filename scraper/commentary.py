import os

import httpx

from .models import Offer, PerfumeTarget


async def refine_comment(target: PerfumeTarget, offer: Offer) -> str:
    """Optionally replace the deterministic warning through an OpenAI-compatible LLM."""
    api_url = "https://openrouter.ai/api/v1/chat/completions"
    api_key = os.getenv("OPENROUTER_API_KEY", "").strip()
    model = os.getenv("OPENROUTER_MODEL", "").strip()
    if not api_key or not model or offer.risk_level not in {"medium", "high"}:
        return offer.comment

    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
        "HTTP-Referer": os.getenv("OPENROUTER_SITE_URL", "https://github.com/merkulovlad/nuhach"),
        "X-OpenRouter-Title": "Nuhach",
    }
    payload = {
        "model": model,
        "temperature": 0.1,
        "max_tokens": 120,
        "messages": [
            {
                "role": "system",
                "content": (
                    "Write one short Russian warning about a perfume offer. Never claim that the "
                    "product is fake. Use cautious wording such as 'возможно' and mention the main signal."
                ),
            },
            {
                "role": "user",
                "content": (
                    f"Perfume: {target.query}; store: {offer.store}; seller: {offer.seller or 'unknown'}; "
                    f"price: {offer.price} {offer.currency}; risk: {offer.risk_level}; "
                    f"calculated warning: {offer.comment}"
                ),
            },
        ],
    }
    try:
        async with httpx.AsyncClient(timeout=10) as client:
            response = await client.post(api_url, headers=headers, json=payload)
            response.raise_for_status()
            content = response.json()["choices"][0]["message"]["content"].strip()
            return content[:500] or offer.comment
    except Exception:
        return offer.comment
