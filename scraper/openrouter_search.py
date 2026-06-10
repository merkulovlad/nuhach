import asyncio
import json
import logging
import os
import re
from urllib.parse import urlsplit

import httpx

from .models import Offer, PerfumeTarget

STORE_DOMAINS = {
    "Ozon": ("ozon.ru",),
    "ЛЭТУАЛЬ": ("letu.ru",),
    "Золотое Яблоко": ("goldapple.ru",),
    "Рандеву": ("randewoo.ru",),
}
ALLOWED_DOMAINS = tuple(domain for domains in STORE_DOMAINS.values() for domain in domains)
logger = logging.getLogger(__name__)


class OpenRouterResponseError(RuntimeError):
    pass


async def search_offers(target: PerfumeTarget) -> list[Offer]:
    api_key = os.getenv("OPENROUTER_API_KEY", "").strip()
    model = os.getenv("OPENROUTER_MODEL", "").strip()
    if not api_key or not model:
        return []

    results = await asyncio.gather(
        *(
            _search_store(target, store, domains, api_key, model)
            for store, domains in STORE_DOMAINS.items()
        ),
        return_exceptions=True,
    )
    offers: list[Offer] = []
    for store, result in zip(STORE_DOMAINS, results, strict=False):
        if isinstance(result, Exception):
            logger.warning("OpenRouter search for %s failed: %s", store, result)
            continue
        offers.extend(result)
    return offers


async def _search_store(
    target: PerfumeTarget,
    store: str,
    domains: tuple[str, ...],
    api_key: str,
    model: str,
) -> list[Offer]:
    schema = {
        "name": "perfume_offers",
        "strict": True,
        "schema": {
            "type": "object",
            "properties": {
                "offers": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {
                            "store": {"type": "string"},
                            "seller": {"type": "string"},
                            "title": {"type": "string"},
                            "price": {"type": "number"},
                            "regular_price": {"type": ["number", "null"]},
                            "volume_ml": {"type": ["integer", "null"]},
                            "price_access": {
                                "type": "string",
                                "enum": ["public", "login", "loyalty", "promo", "unknown"],
                            },
                            "price_is_full": {"type": "boolean"},
                            "installment_payment": {"type": ["number", "null"]},
                            "installment_count": {"type": ["integer", "null"]},
                            "currency": {"type": "string"},
                            "url": {"type": "string"},
                            "in_stock": {"type": "boolean"},
                            "is_match": {"type": "boolean"},
                            "match_confidence": {"type": "number", "minimum": 0, "maximum": 1},
                        },
                        "required": [
                            "store",
                            "seller",
                            "title",
                            "price",
                            "regular_price",
                            "volume_ml",
                            "price_access",
                            "price_is_full",
                            "installment_payment",
                            "installment_count",
                            "currency",
                            "url",
                            "in_stock",
                            "is_match",
                            "match_confidence",
                        ],
                        "additionalProperties": False,
                    },
                }
            },
            "required": ["offers"],
            "additionalProperties": False,
        },
    }
    payload = {
        "model": model,
        "temperature": 0,
        "max_tokens": int(os.getenv("OPENROUTER_MAX_TOKENS", "4000")),
        "response_format": {"type": "json_schema", "json_schema": schema},
        "provider": {"require_parameters": True},
        "plugins": [
            {
                "id": "web",
                "engine": "exa",
                "max_results": 8,
                "include_domains": list(domains),
            },
            {"id": "response-healing"},
        ],
        "messages": [
            {
                "role": "system",
                "content": (
                    "Find current perfume offers only in the requested store. Return an offer only when "
                    "the evidence explicitly contains its title, full one-time purchase price, product URL "
                    "and availability. The price field must be the full price of the product, never a monthly "
                    "or installment payment. For example, if the page says 8,899 RUB and 2,224 RUB x 4, "
                    "price must be 8899, installment_payment 2224, installment_count 4 and price_is_full true. "
                    "When several full prices are visible for the same volume, use the lowest price available "
                    "to any customer without login, loyalty membership or promo code. Put the crossed-out or "
                    "regular price into regular_price. Set price_access accurately. Do not use a login-only, "
                    "loyalty-only or promo-only price as price. Pair each price with the exact volume shown; "
                    "do not mix prices between 30, 50 and 100 ml variants. If the full public price or volume "
                    "cannot be verified, omit the offer. Prefer an exact product-card URL. "
                    "If no product-card URL is present in the evidence, a store brand, category or search URL "
                    "is acceptable as a fallback, but never invent a product URL. "
                    "Do not estimate or invent missing values. Keep marketplace seller names when visible. "
                    "Set is_match only for the exact fragrance, not alternatives, decants or inspired copies. "
                    "Return at most 3 offers and keep all strings concise."
                ),
            },
            {
                "role": "user",
                "content": f"Find {target.brand} {target.name} in stock at {store}.",
            },
        ],
    }
    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
        "HTTP-Referer": os.getenv("OPENROUTER_SITE_URL", "https://github.com/merkulovlad/nuhach"),
        "X-OpenRouter-Title": "Nuhach",
    }
    message = await _request_message(payload, headers)
    data = _parse_json_content(message.get("content"))
    evidence_urls = {
        _canonical_url(annotation["url_citation"]["url"])
        for annotation in message.get("annotations", [])
        if annotation.get("type") == "url_citation"
        and annotation.get("url_citation", {}).get("url")
    }

    offers: list[Offer] = []
    for item in data.get("offers", []):
        hostname = (urlsplit(item["url"]).hostname or "").lower()
        if not any(hostname == domain or hostname.endswith(f".{domain}") for domain in domains):
            continue
        if _canonical_url(item["url"]) not in evidence_urls:
            continue
        if item["price"] <= 0 or not item["price_is_full"]:
            continue
        if item["price_access"] != "public":
            continue
        installment_payment = item.get("installment_payment")
        if installment_payment and item["price"] <= installment_payment * 1.1:
            continue
        if not item["is_match"] or item["match_confidence"] < 0.7:
            continue
        offers.append(
            Offer(
                store=store,
                seller=item["seller"],
                title=item["title"],
                price=float(item["price"]),
                old_price=float(item["regular_price"]) if item.get("regular_price") else None,
                currency=item["currency"] or "RUB",
                url=item["url"],
                volume_ml=item.get("volume_ml"),
                in_stock=bool(item["in_stock"]),
                match_confidence=float(item["match_confidence"]),
                is_catalog_link=_is_listing_url(item["url"]),
            )
        )
    return offers


async def _request_message(payload: dict, headers: dict) -> dict:
    attempts = int(os.getenv("OPENROUTER_MAX_ATTEMPTS", "2"))
    last_error: Exception | None = None
    async with httpx.AsyncClient(timeout=60) as client:
        for attempt in range(attempts):
            request_payload = dict(payload)
            request_payload["max_tokens"] = payload["max_tokens"] * (attempt + 1)
            response = await client.post(
                "https://openrouter.ai/api/v1/chat/completions",
                headers=headers,
                json=request_payload,
            )
            response.raise_for_status()
            body = response.json()
            choices = body.get("choices") or []
            if not choices:
                last_error = OpenRouterResponseError("OpenRouter returned no choices")
                continue

            choice = choices[0]
            message = choice.get("message") or {}
            finish_reason = choice.get("finish_reason")
            try:
                _parse_json_content(message.get("content"))
            except (json.JSONDecodeError, OpenRouterResponseError) as exc:
                last_error = exc
                logger.warning(
                    "Invalid OpenRouter response on attempt %d/%d (finish_reason=%s): %s",
                    attempt + 1,
                    attempts,
                    finish_reason,
                    exc,
                )
                continue
            if finish_reason == "length":
                last_error = OpenRouterResponseError("OpenRouter response was truncated")
                continue
            return message

    raise OpenRouterResponseError(
        f"OpenRouter did not return valid JSON after {attempts} attempts: {last_error}"
    )


def _parse_json_content(content: object) -> dict:
    if isinstance(content, list):
        content = "".join(
            part.get("text", "")
            for part in content
            if isinstance(part, dict) and part.get("type") == "text"
        )
    if not isinstance(content, str) or not content.strip():
        raise OpenRouterResponseError("OpenRouter returned empty content")

    value = content.strip()
    fenced = re.fullmatch(r"```(?:json)?\s*(.*?)\s*```", value, flags=re.DOTALL | re.IGNORECASE)
    if fenced:
        value = fenced.group(1)
    data = json.loads(value)
    if not isinstance(data, dict) or not isinstance(data.get("offers"), list):
        raise OpenRouterResponseError("OpenRouter response does not contain an offers array")
    return data


def _canonical_url(value: str) -> tuple[str, str]:
    parts = urlsplit(value)
    return (parts.netloc.lower(), parts.path.rstrip("/"))


def _is_listing_url(value: str) -> bool:
    parts = urlsplit(value)
    path = parts.path.lower().rstrip("/")
    return (
        not path
        or path.startswith(
            ("/search", "/catalogsearch", "/brand/", "/brands/", "/browse/", "/category/")
        )
        or "/tags/" in path
    )
