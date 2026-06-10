from .generic import GenericHTMLStore


STORE_CONFIGS = (
    {
        "name": "Ozon",
        "search_url": "https://www.ozon.ru/search/?text={query}",
        "selectors": {
            "item": "[data-widget='searchResultsV2'] article",
            "title": "a span",
            "price": "span",
            "url": "a[href*='/product/']",
        },
    },
    {
        "name": "ЛЭТУАЛЬ",
        "search_url": "https://www.letu.ru/search?text={query}",
        "selectors": {
            "item": "[data-test='product-card']",
            "title": "[data-test='product-name']",
            "price": "[data-test='product-price']",
            "url": "a",
        },
    },
    {
        "name": "Золотое Яблоко",
        "search_url": "https://goldapple.ru/catalogsearch/result?q={query}",
        "selectors": {
            "item": "[data-transaction-name='product']",
            "title": "[itemprop='name']",
            "price": "[itemprop='price']",
            "url": "a",
        },
    },
    {
        "name": "Рандеву",
        "search_url": "https://randewoo.ru/search/?q={query}",
        "selectors": {
            "item": ".product-card, [data-product-id]",
            "title": ".product-card__title, [itemprop='name']",
            "price": ".product-card__price, [itemprop='price']",
            "url": "a[href]",
        },
    },
)


def build_stores(enabled_names: set[str] | None = None) -> list[GenericHTMLStore]:
    configs = STORE_CONFIGS
    if enabled_names:
        configs = tuple(config for config in configs if config["name"] in enabled_names)
    return [GenericHTMLStore(config) for config in configs]
