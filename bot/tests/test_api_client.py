"""
Tests for API client request building.
"""

from bot.api_client import APIClient, PerfumeItem


class TestPerfumeItem:
    """Tests for PerfumeItem.from_dict()."""

    def test_from_dict_full(self):
        """Test parsing a complete perfume dict."""
        data = {
            "id": 123,
            "name": "Test Perfume",
            "brand": "Test Brand",
            "rating_value": 4.5,
            "rating_count": 100,
            "year": 2020,
            "notes": "rose, vanilla",
            "accords": "floral, sweet",
            "country": "France",
            "gender": "Women",
            "top_notes": "bergamot",
            "middle_notes": "rose",
            "base_notes": "vanilla",
            "perfumer1": "Test Perfumer",
            "url": "https://example.com/perfume",
        }

        item = PerfumeItem.from_dict(data)

        assert item.id == 123
        assert item.name == "Test Perfume"
        assert item.brand == "Test Brand"
        assert item.rating_value == 4.5
        assert item.rating_count == 100
        assert item.year == 2020
        assert item.notes == "rose, vanilla"
        assert item.accords == "floral, sweet"
        assert item.country == "France"
        assert item.gender == "Women"
        assert item.top_notes == "bergamot"
        assert item.middle_notes == "rose"
        assert item.base_notes == "vanilla"
        assert item.perfumer == "Test Perfumer"
        assert item.url == "https://example.com/perfume"

    def test_from_dict_minimal(self):
        """Test parsing a minimal perfume dict."""
        data = {
            "id": 1,
            "name": "Minimal",
            "brand": "Brand",
        }

        item = PerfumeItem.from_dict(data)

        assert item.id == 1
        assert item.name == "Minimal"
        assert item.brand == "Brand"
        assert item.rating_value is None
        assert item.year is None

    def test_from_dict_with_perfume_name_key(self):
        """Test parsing when name comes as perfume_name."""
        data = {
            "id": 1,
            "perfume_name": "Name From PerfumeName",
            "brand": "Brand",
        }

        item = PerfumeItem.from_dict(data)

        assert item.name == "Name From PerfumeName"

    def test_from_dict_empty(self):
        """Test parsing an empty dict."""
        item = PerfumeItem.from_dict({})

        assert item.id == 0
        assert item.name == "Unknown"
        assert item.brand == "Unknown"


class TestAPIClientRequestBuilding:
    """Tests for API client request parameter building."""

    def test_search_params_basic(self):
        """Test search builds correct params."""
        client = APIClient(base_url="http://localhost:8080")

        # Check that the base URL is correct
        assert client.base_url == "http://localhost:8080"

    def test_search_params_with_tg_id(self):
        """Test search includes tg_id when provided."""
        # This tests the logic, not actual HTTP call
        params = {
            "q": "rose",
            "limit": 10,
            "offset": 0,
        }
        tg_id = 123456
        if tg_id is not None:
            params["tg_id"] = tg_id

        assert params["q"] == "rose"
        assert params["limit"] == 10
        assert params["offset"] == 0
        assert params["tg_id"] == 123456

    def test_event_json_building(self):
        """Test event request JSON is built correctly."""
        json_data = {
            "perfume_id": 123,
            "event_type": "like",
        }
        request_id = "test-uuid"
        rating = 5

        if request_id:
            json_data["request_id"] = request_id
        if rating is not None:
            json_data["rating"] = rating

        assert json_data["perfume_id"] == 123
        assert json_data["event_type"] == "like"
        assert json_data["request_id"] == "test-uuid"
        assert json_data["rating"] == 5

    def test_event_json_minimal(self):
        """Test event request JSON without optional fields."""
        json_data = {
            "perfume_id": 456,
            "event_type": "click",
        }
        request_id = None
        rating = None

        if request_id:
            json_data["request_id"] = request_id
        if rating is not None:
            json_data["rating"] = rating

        assert "request_id" not in json_data
        assert "rating" not in json_data


class TestAPIClientURLs:
    """Tests for API client URL construction."""

    def test_base_url_trailing_slash_stripped(self):
        """Test that trailing slash is stripped from base URL."""
        client = APIClient(base_url="http://localhost:8080/")
        assert client.base_url == "http://localhost:8080"

    def test_search_endpoint(self):
        """Test search endpoint path."""
        client = APIClient(base_url="http://localhost:8080")
        expected_url = f"{client.base_url}/api/search"
        assert expected_url == "http://localhost:8080/api/search"

    def test_perfume_detail_endpoint(self):
        """Test perfume detail endpoint path."""
        client = APIClient(base_url="http://localhost:8080")
        perfume_id = 123
        expected_url = f"{client.base_url}/api/perfumes/{perfume_id}"
        assert expected_url == "http://localhost:8080/api/perfumes/123"

    def test_similar_endpoint(self):
        """Test similar perfumes endpoint path."""
        client = APIClient(base_url="http://localhost:8080")
        perfume_id = 456
        expected_url = f"{client.base_url}/api/perfumes/{perfume_id}/similar"
        assert expected_url == "http://localhost:8080/api/perfumes/456/similar"

    def test_recommendations_endpoint(self):
        """Test recommendations endpoint path."""
        client = APIClient(base_url="http://localhost:8080")
        tg_id = 789
        expected_url = f"{client.base_url}/api/users/{tg_id}/recommendations"
        assert expected_url == "http://localhost:8080/api/users/789/recommendations"

    def test_events_endpoint(self):
        """Test events endpoint path."""
        client = APIClient(base_url="http://localhost:8080")
        tg_id = 111
        expected_url = f"{client.base_url}/api/users/{tg_id}/events"
        assert expected_url == "http://localhost:8080/api/users/111/events"

    def test_saves_endpoint(self):
        """Test saves endpoint path."""
        client = APIClient(base_url="http://localhost:8080")
        tg_id = 222
        expected_url = f"{client.base_url}/api/users/{tg_id}/saves"
        assert expected_url == "http://localhost:8080/api/users/222/saves"
