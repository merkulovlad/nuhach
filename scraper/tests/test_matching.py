from scraper.matching import assess_price_risk, enrich_and_filter
from scraper.models import Offer, PerfumeTarget


def test_matching_extracts_variant_and_rejects_unrelated_product():
    target = PerfumeTarget(id=1, brand="Tom Ford", name="Tobacco Vanille")
    offers = enrich_and_filter(
        target,
        [
            Offer(
                store="Example",
                title="Tom Ford Tobacco Vanille EDP 100 мл tester",
                price=10000,
                url="https://example.com/1",
            ),
            Offer(
                store="Example",
                title="Dior Sauvage EDT 100 мл",
                price=9000,
                url="https://example.com/2",
            ),
        ],
    )

    assert len(offers) == 1
    assert offers[0].volume_ml == 100
    assert offers[0].concentration == "EDP"
    assert offers[0].product_type == "tester"


def test_price_risk_warns_for_large_discount_within_same_variant():
    offers = [
        Offer(
            store="A",
            title="X",
            price=10000,
            url="https://a",
            volume_ml=100,
            concentration="EDP",
            product_type="retail",
        ),
        Offer(
            store="B",
            title="X",
            price=11000,
            url="https://b",
            volume_ml=100,
            concentration="EDP",
            product_type="retail",
        ),
        Offer(
            store="C",
            title="X",
            price=4000,
            url="https://c",
            volume_ml=100,
            concentration="EDP",
            product_type="retail",
        ),
    ]

    assess_price_risk(offers)

    assert offers[2].risk_level == "high"
    assert "Возможно" in offers[2].comment
    assert offers[0].risk_level == "low"


def test_single_offer_has_unknown_price_risk():
    offers = [Offer(store="A", title="X", price=10000, url="https://a")]

    assess_price_risk(offers)

    assert offers[0].risk_level == "unknown"


def test_ozon_offer_warns_about_marketplace_even_at_normal_price():
    offers = [
        Offer(
            store="Ozon", seller="Seller", title="X", price=9000, url="https://ozon.ru/product/x"
        ),
        Offer(
            store="ЛЭТУАЛЬ",
            seller="ЛЭТУАЛЬ",
            title="X",
            price=9100,
            url="https://letu.ru/product/x",
        ),
    ]

    assess_price_risk(offers)

    assert offers[0].risk_level == "medium"
    assert "маркетплейс" in offers[0].comment


def test_single_ozon_offer_still_warns_about_marketplace():
    offers = [Offer(store="Ozon", title="X", price=9000, url="https://ozon.ru/product/x")]

    assess_price_risk(offers)

    assert offers[0].risk_level == "medium"
    assert "продавца" in offers[0].comment
