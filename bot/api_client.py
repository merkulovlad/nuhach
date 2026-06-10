"""
HTTP client for the Nuhach API backend.

Centralizes all API calls with timeouts, retries, and error handling.
"""

import asyncio
import logging
from dataclasses import dataclass
from typing import Any

import aiohttp
from aiohttp import ClientTimeout

logger = logging.getLogger(__name__)


@dataclass
class PerfumeItem:
    """Perfume data from API."""

    id: int
    name: str
    brand: str
    rating_value: float | None
    rating_count: int | None
    year: int | None
    notes: str | None = None
    accords: str | None = None
    country: str | None = None
    gender: str | None = None
    top_notes: str | None = None
    middle_notes: str | None = None
    base_notes: str | None = None
    perfumer: str | None = None
    url: str | None = None

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "PerfumeItem":
        """Create PerfumeItem from API response dict."""
        return cls(
            id=data.get("id", 0),
            name=data.get("name", data.get("perfume_name", "Unknown")),
            brand=data.get("brand", "Unknown"),
            rating_value=data.get("rating_value"),
            rating_count=data.get("rating_count"),
            year=data.get("year"),
            notes=data.get("notes"),
            accords=data.get("accords"),
            country=data.get("country"),
            gender=data.get("gender"),
            top_notes=data.get("top_notes"),
            middle_notes=data.get("middle_notes"),
            base_notes=data.get("base_notes"),
            perfumer=data.get("perfumer", data.get("perfumer1")),
            url=data.get("url"),
        )


@dataclass
class SearchResult:
    """Search/recommendations result from API."""

    items: list[PerfumeItem]
    request_id: str
    total: int
    exploration_ids: list[int] | None = None


@dataclass
class StoreOffer:
    store: str
    title: str
    price: float
    currency: str
    url: str
    old_price: float | None = None
    seller: str | None = None
    volume_ml: int | None = None
    concentration: str | None = None
    product_type: str | None = None
    in_stock: bool = True
    risk_level: str = "unknown"
    comment: str | None = None
    checked_at: str | None = None

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "StoreOffer":
        return cls(
            store=data.get("store", "Unknown"),
            title=data.get("title", "Unknown"),
            price=float(data.get("price", 0)),
            currency=data.get("currency", "RUB"),
            url=data.get("url", ""),
            old_price=float(data["old_price"]) if data.get("old_price") is not None else None,
            seller=data.get("seller"),
            volume_ml=data.get("volume_ml"),
            concentration=data.get("concentration"),
            product_type=data.get("product_type"),
            in_stock=data.get("in_stock", True),
            risk_level=data.get("risk_level", "unknown"),
            comment=data.get("comment"),
            checked_at=data.get("checked_at"),
        )


@dataclass
class OfferSearchResult:
    perfume_id: int
    status: str
    offers: list[StoreOffer]
    job_id: int | None = None
    error: str | None = None

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "OfferSearchResult":
        return cls(
            perfume_id=data.get("perfume_id", 0),
            status=data.get("status", "empty"),
            offers=[StoreOffer.from_dict(item) for item in (data.get("offers") or [])],
            job_id=data.get("job_id"),
            error=data.get("error"),
        )


class APIError(Exception):
    """Custom exception for API errors."""

    def __init__(self, message: str, status_code: int | None = None):
        super().__init__(message)
        self.status_code = status_code


class APIClient:
    """HTTP client for Nuhach API."""

    def __init__(
        self,
        base_url: str,
        timeout: float = 10.0,
        max_retries: int = 3,
        retry_delay: float = 0.5,
    ):
        self.base_url = base_url.rstrip("/")
        self.timeout = ClientTimeout(total=timeout)
        self.max_retries = max_retries
        self.retry_delay = retry_delay
        self._session: aiohttp.ClientSession | None = None

    async def _get_session(self) -> aiohttp.ClientSession:
        """Get or create HTTP session."""
        if self._session is None or self._session.closed:
            self._session = aiohttp.ClientSession(timeout=self.timeout)
        return self._session

    async def close(self):
        """Close the HTTP session."""
        if self._session and not self._session.closed:
            await self._session.close()

    async def _request(
        self,
        method: str,
        path: str,
        params: dict[str, Any] | None = None,
        json_data: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        """Make HTTP request with retries."""
        url = f"{self.base_url}{path}"
        session = await self._get_session()

        last_error: Exception | None = None

        for attempt in range(self.max_retries):
            try:
                async with session.request(method, url, params=params, json=json_data) as response:
                    if response.status >= 400:
                        error_text = await response.text()
                        logger.error(
                            "API error: %s %s -> %d: %s", method, url, response.status, error_text
                        )
                        raise APIError(
                            f"API returned {response.status}: {error_text}",
                            status_code=response.status,
                        )
                    return await response.json()

            except aiohttp.ClientError as e:
                last_error = e
                logger.warning(
                    "Request failed (attempt %d/%d): %s", attempt + 1, self.max_retries, e
                )
                if attempt < self.max_retries - 1:
                    await asyncio.sleep(self.retry_delay * (attempt + 1))
            except APIError:
                raise
            except Exception as e:
                last_error = e
                logger.error("Unexpected error during request: %s", e)
                if attempt < self.max_retries - 1:
                    await asyncio.sleep(self.retry_delay * (attempt + 1))

        raise APIError(f"Request failed after {self.max_retries} attempts: {last_error}")

    async def health_check(self) -> bool:
        """Check if API is healthy."""
        try:
            result = await self._request("GET", "/api/health")
            return result.get("status") == "ok"
        except Exception as e:
            logger.error("Health check failed: %s", e)
            return False

    async def search(
        self,
        query: str,
        limit: int = 10,
        offset: int = 0,
        tg_id: int | None = None,
        embedding: list[float] | None = None,
    ) -> SearchResult:
        """
        Search for perfumes.

        Args:
            query: Text search query
            limit: Max results to return
            offset: Pagination offset
            tg_id: Telegram user ID for personalization
            embedding: Optional pre-computed query embedding for vector search
        """
        params: dict[str, Any] = {
            "q": query,
            "limit": limit,
            "offset": offset,
        }
        if tg_id is not None:
            params["tg_id"] = tg_id

        # If embedding provided, use POST with vector search
        if embedding is not None:
            json_data = {
                "query": query,
                "embedding": embedding,
                "limit": limit,
                "offset": offset,
            }
            if tg_id is not None:
                json_data["tg_id"] = tg_id
            data = await self._request("POST", "/api/search/vector", json_data=json_data)
        else:
            data = await self._request("GET", "/api/search", params=params)

        # Handle null items (API returns null instead of [])
        raw_items = data.get("items") or []
        items = [PerfumeItem.from_dict(item) for item in raw_items]
        return SearchResult(
            items=items,
            request_id=data.get("request_id", ""),
            total=data.get("total", 0),
        )

    async def get_perfume(self, perfume_id: int) -> PerfumeItem | None:
        """Get perfume details by ID."""
        try:
            data = await self._request("GET", f"/api/perfumes/{perfume_id}")
            return PerfumeItem.from_dict(data)
        except APIError as e:
            if e.status_code == 404:
                return None
            raise

    async def get_similar(
        self,
        perfume_id: int,
        limit: int = 10,
        tg_id: int | None = None,
    ) -> SearchResult:
        """Get similar perfumes."""
        params: dict[str, Any] = {"limit": limit}
        if tg_id is not None:
            params["tg_id"] = tg_id

        data = await self._request("GET", f"/api/perfumes/{perfume_id}/similar", params=params)

        raw_items = data.get("items") or []
        items = [PerfumeItem.from_dict(item) for item in raw_items]
        return SearchResult(
            items=items,
            request_id=data.get("request_id", ""),
            total=data.get("total", 0),
        )

    async def get_recommendations(
        self,
        tg_id: int,
        limit: int = 10,
    ) -> SearchResult:
        """Get personalized recommendations for user."""
        params = {"limit": limit}

        data = await self._request("GET", f"/api/users/{tg_id}/recommendations", params=params)

        raw_items = data.get("items") or []
        items = [PerfumeItem.from_dict(item) for item in raw_items]
        return SearchResult(
            items=items,
            request_id=data.get("request_id", ""),
            total=len(items),
            exploration_ids=data.get("exploration_ids"),
        )

    async def get_saves(self, tg_id: int) -> list[PerfumeItem]:
        """Get user's saved perfumes."""
        data = await self._request("GET", f"/api/users/{tg_id}/saves")
        raw_items = data.get("items") or []
        return [PerfumeItem.from_dict(item) for item in raw_items]

    async def get_offers(self, perfume_id: int) -> OfferSearchResult:
        data = await self._request("GET", f"/api/perfumes/{perfume_id}/offers")
        return OfferSearchResult.from_dict(data)

    async def search_offers(self, perfume_id: int, force: bool = False) -> OfferSearchResult:
        params = {"force": "true"} if force else None
        data = await self._request(
            "POST", f"/api/perfumes/{perfume_id}/offers/search", params=params
        )
        return OfferSearchResult.from_dict(data)

    async def create_event(
        self,
        tg_id: int,
        perfume_id: int,
        event_type: str,
        request_id: str | None = None,
        rating: int | None = None,
    ) -> bool:
        """Create a user event."""
        json_data: dict[str, Any] = {
            "perfume_id": perfume_id,
            "event_type": event_type,
        }
        if request_id:
            json_data["request_id"] = request_id
        if rating is not None:
            json_data["rating"] = rating

        try:
            result = await self._request("POST", f"/api/users/{tg_id}/events", json_data=json_data)
            return result.get("status") == "ok"
        except APIError as e:
            logger.error("Failed to create event: %s", e)
            return False

    async def log_impression(
        self,
        tg_id: int,
        perfume_ids: list[int],
        request_id: str,
    ) -> None:
        """Log impression events for a list of perfumes."""
        for perfume_id in perfume_ids:
            await self.create_event(
                tg_id=tg_id,
                perfume_id=perfume_id,
                event_type="impression",
                request_id=request_id,
            )
