import re
import unicodedata
from statistics import median
from typing import Iterable

from .models import Offer, PerfumeTarget


VOLUME_RE = re.compile(r"(?<!\d)(\d{1,3})\s*(?:ml|мл)\b", re.IGNORECASE)
CONCENTRATIONS = {
    "edp": ("edp", "eau de parfum", "парфюмерная вода", "парфюмированная вода"),
    "edt": ("edt", "eau de toilette", "туалетная вода"),
    "parfum": ("parfum", "духи", "экстракт"),
}


def normalize(value: str) -> str:
    value = unicodedata.normalize("NFKD", value).casefold()
    value = re.sub(r"[^\w\s]", " ", value)
    return " ".join(value.split())


def enrich_and_filter(target: PerfumeTarget, offers: Iterable[Offer]) -> list[Offer]:
    expected = set(normalize(target.query).split())
    accepted: list[Offer] = []
    for offer in offers:
        title = normalize(offer.title)
        tokens = set(title.split())
        overlap = len(expected & tokens) / max(len(expected), 1)
        brand_tokens = set(normalize(target.brand).split())
        brand_match = brand_tokens.issubset(tokens)
        local_confidence = min(1.0, overlap * 0.8 + (0.2 if brand_match else 0.0))
        offer.match_confidence = max(offer.match_confidence, local_confidence)
        if offer.match_confidence < 0.55:
            continue

        volume = VOLUME_RE.search(offer.title)
        if volume:
            offer.volume_ml = int(volume.group(1))
        for canonical, aliases in CONCENTRATIONS.items():
            if any(alias in title for alias in aliases):
                offer.concentration = canonical.upper()
                break
        if "tester" in title or "тестер" in title:
            offer.product_type = "tester"
        elif "отливант" in title or "распив" in title:
            offer.product_type = "decant"
        else:
            offer.product_type = "retail"
        accepted.append(offer)
    return accepted


def deduplicate_offers(offers: Iterable[Offer]) -> list[Offer]:
    best: dict[tuple[str, str, int | None], Offer] = {}
    for offer in offers:
        key = (offer.store.casefold(), offer.title.casefold(), offer.volume_ml)
        current = best.get(key)
        if current is None or (current.is_catalog_link and not offer.is_catalog_link):
            best[key] = offer
    return list(best.values())


def assess_price_risk(offers: list[Offer]) -> None:
    groups: dict[tuple, list[Offer]] = {}
    for offer in offers:
        key = (offer.volume_ml, offer.concentration, offer.product_type)
        groups.setdefault(key, []).append(offer)

    for group in groups.values():
        prices = [item.price for item in group if item.in_stock and item.price > 0]
        market_price = median(prices) if prices else 0
        for offer in group:
            is_ozon = normalize(offer.store) == "ozon"
            if market_price <= 0 or len(prices) < 2:
                if is_ozon:
                    offer.risk_level = "medium"
                    offer.risk_score = 0.2
                    offer.comment = (
                        "Ozon — маркетплейс: подлинность зависит от продавца. "
                        "Проверьте продавца, отзывы, маркировку и возможность возврата."
                    )
                else:
                    offer.risk_level = "unknown"
                    offer.comment = "Недостаточно предложений для оценки цены."
                continue
            discount = max(0.0, 1 - offer.price / market_price)
            seller_penalty = 0.15 if not offer.seller else 0.0
            marketplace_penalty = 0.2 if is_ozon else 0.0
            offer.risk_score = min(1.0, discount + seller_penalty + marketplace_penalty)
            if discount >= 0.45:
                offer.risk_level = "high"
                offer.comment = (
                    "Осторожно: цена существенно ниже других предложений. "
                    "Возможно, товар неоригинальный; проверьте продавца и маркировку."
                )
            elif discount >= 0.25 or is_ozon:
                offer.risk_level = "medium"
                if is_ozon:
                    offer.comment = (
                        "Ozon — маркетплейс: подлинность зависит от продавца. "
                        "Проверьте продавца, отзывы, маркировку и возможность возврата."
                    )
                else:
                    offer.comment = "Цена заметно ниже рынка. Стоит дополнительно проверить продавца и отзывы."
            else:
                offer.risk_level = "low"
                offer.comment = "Цена находится в обычном диапазоне найденных предложений."
