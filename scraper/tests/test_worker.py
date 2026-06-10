from scraper.matching import deduplicate_offers
from scraper.models import Offer


def test_deduplicate_prefers_product_card_over_catalog_link():
    offers = [
        Offer(
            store="ЛЭТУАЛЬ",
            title="VERSACE Versense",
            price=8899,
            url="https://www.letu.ru/brand/versace",
            is_catalog_link=True,
        ),
        Offer(
            store="ЛЭТУАЛЬ",
            title="VERSACE Versense",
            price=8899,
            url="https://www.letu.ru/product/versense",
        ),
    ]

    result = deduplicate_offers(offers)

    assert len(result) == 1
    assert result[0].url.endswith("/product/versense")


def test_deduplicate_keeps_different_volumes():
    offers = [
        Offer(store="Shop", title="Versense", price=7000, url="https://shop/30", volume_ml=30),
        Offer(store="Shop", title="Versense", price=9000, url="https://shop/50", volume_ml=50),
    ]

    assert len(deduplicate_offers(offers)) == 2
