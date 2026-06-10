from dataclasses import dataclass
from typing import Optional


@dataclass
class PerfumeTarget:
    id: int
    brand: str
    name: str

    @property
    def query(self) -> str:
        return f"{self.brand} {self.name}".strip()


@dataclass
class Offer:
    store: str
    title: str
    price: float
    url: str
    seller: str = ""
    old_price: Optional[float] = None
    currency: str = "RUB"
    volume_ml: Optional[int] = None
    concentration: str = ""
    product_type: str = ""
    in_stock: bool = True
    rating: Optional[float] = None
    reviews_count: Optional[int] = None
    match_confidence: float = 0.0
    risk_level: str = "unknown"
    risk_score: float = 0.0
    comment: str = ""
    is_catalog_link: bool = False
