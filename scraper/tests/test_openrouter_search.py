import json

import pytest

from scraper.openrouter_search import OpenRouterResponseError, _is_listing_url, _parse_json_content


def test_parse_json_content_accepts_json_object():
    assert _parse_json_content('{"offers": []}') == {"offers": []}


def test_parse_json_content_accepts_markdown_fence():
    assert _parse_json_content('```json\n{"offers": []}\n```') == {"offers": []}


def test_parse_json_content_accepts_openrouter_content_parts():
    content = [{"type": "text", "text": '{"offers": []}'}]
    assert _parse_json_content(content) == {"offers": []}


@pytest.mark.parametrize("content", ["", "{}", "[]"])
def test_parse_json_content_rejects_invalid_contract(content):
    with pytest.raises((json.JSONDecodeError, OpenRouterResponseError)):
        _parse_json_content(content)


@pytest.mark.parametrize(
    "url",
    [
        "https://www.letu.ru/brand/versace",
        "https://www.letu.ru/browse/parfyumeriya/tags/versace",
        "https://goldapple.ru/catalogsearch/result?q=versense",
        "https://goldapple.ru/brands/versace",
    ],
)
def test_listing_urls_are_detected(url):
    assert _is_listing_url(url)


def test_product_url_is_accepted():
    assert not _is_listing_url("https://www.ozon.ru/product/versace-versense-30-ml-123456/")
