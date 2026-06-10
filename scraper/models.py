from dataclasses import dataclass


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
    old_price: float | None = None
    currency: str = "RUB"
    volume_ml: int | None = None
    concentration: str = ""
    product_type: str = ""
    in_stock: bool = True
    rating: float | None = None
    reviews_count: int | None = None
    match_confidence: float = 0.0
    risk_level: str = "unknown"
    risk_score: float = 0.0
    comment: str = ""
    is_catalog_link: bool = False
