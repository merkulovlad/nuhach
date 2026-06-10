from urllib.parse import quote_plus, urljoin

import httpx
from selectolax.parser import HTMLParser

from scraper.models import Offer, PerfumeTarget
from scraper.robots import ensure_allowed
from .base import StoreScraper


class GenericHTMLStore(StoreScraper):
    """CSS-selector adapter for server-rendered store search pages."""

    def __init__(self, config: dict):
        self.config = config
        self.name = config["name"]

    async def search(self, target: PerfumeTarget) -> list[Offer]:
        search_url = self.config["search_url"].format(query=quote_plus(target.query))
        user_agent = self.config.get(
            "user_agent",
            "NuhachOfferBot/1.0 (+https://github.com/merkulovlad/nuhach)",
        )
        await ensure_allowed(search_url, user_agent)
        headers = {"User-Agent": user_agent}
        async with httpx.AsyncClient(
            timeout=self.config.get("timeout_seconds", 12),
            follow_redirects=True,
            headers=headers,
        ) as client:
            response = await client.get(search_url)
            response.raise_for_status()

        tree = HTMLParser(response.text)
        selectors = self.config["selectors"]
        offers: list[Offer] = []
        for card in tree.css(selectors["item"]):
            title_node = card.css_first(selectors["title"])
            price_node = card.css_first(selectors["price"])
            link_node = card.css_first(selectors["url"])
            if not title_node or not price_node or not link_node:
                continue
            price = self._parse_price(price_node.text())
            href = link_node.attributes.get("href", "")
            if price is None or not href:
                continue
            seller_node = card.css_first(selectors.get("seller", "")) if selectors.get("seller") else None
            offers.append(
                Offer(
                    store=self.name,
                    title=title_node.text(strip=True),
                    price=price,
                    seller=seller_node.text(strip=True) if seller_node else "",
                    url=urljoin(search_url, href),
                )
            )
        return offers[: self.config.get("max_results", 20)]

    @staticmethod
    def _parse_price(value: str) -> float | None:
        normalized = "".join(ch for ch in value if ch.isdigit() or ch in ",.")
        normalized = normalized.replace(",", ".")
        if not normalized:
            return None
        try:
            return float(normalized)
        except ValueError:
            return None
