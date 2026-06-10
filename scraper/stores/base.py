from abc import ABC, abstractmethod

from scraper.models import Offer, PerfumeTarget


class StoreScraper(ABC):
    @abstractmethod
    async def search(self, target: PerfumeTarget) -> list[Offer]:
        raise NotImplementedError
