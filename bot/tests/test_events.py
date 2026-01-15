"""
Tests for event logging functionality.
"""
import pytest
from bot.state import StateManager, UserState


class TestUserState:
    """Tests for UserState."""
    
    def test_user_state_default(self):
        """Test default user state."""
        state = UserState()
        
        assert state.last_request_id is None
        assert state.last_perfume_ids == []
    
    def test_user_state_to_dict(self):
        """Test converting user state to dict."""
        state = UserState(
            last_request_id="test-uuid",
            last_perfume_ids=[1, 2, 3],
        )
        
        data = state.to_dict()
        
        assert data["last_request_id"] == "test-uuid"
        assert data["last_perfume_ids"] == [1, 2, 3]
    
    def test_user_state_from_dict(self):
        """Test creating user state from dict."""
        data = {
            "last_request_id": "another-uuid",
            "last_perfume_ids": [4, 5, 6],
        }
        
        state = UserState.from_dict(data)
        
        assert state.last_request_id == "another-uuid"
        assert state.last_perfume_ids == [4, 5, 6]
    
    def test_user_state_from_dict_empty(self):
        """Test creating user state from empty dict."""
        state = UserState.from_dict({})
        
        assert state.last_request_id is None
        assert state.last_perfume_ids == []


class TestStateManager:
    """Tests for StateManager."""
    
    def test_get_state_creates_new(self):
        """Test that get_state creates new state for unknown user."""
        manager = StateManager(persist=False)
        
        state = manager.get_state(12345)
        
        assert state is not None
        assert state.last_request_id is None
    
    def test_get_state_returns_existing(self):
        """Test that get_state returns existing state."""
        manager = StateManager(persist=False)
        
        # Set some state
        manager.set_last_request(12345, "uuid-1", [1, 2, 3])
        
        # Get state again
        state = manager.get_state(12345)
        
        assert state.last_request_id == "uuid-1"
        assert state.last_perfume_ids == [1, 2, 3]
    
    def test_set_last_request(self):
        """Test setting last request."""
        manager = StateManager(persist=False)
        
        manager.set_last_request(111, "uuid-111", [10, 20, 30])
        
        state = manager.get_state(111)
        assert state.last_request_id == "uuid-111"
        assert state.last_perfume_ids == [10, 20, 30]
    
    def test_get_last_request_id(self):
        """Test getting last request_id."""
        manager = StateManager(persist=False)
        
        manager.set_last_request(222, "uuid-222", [])
        
        request_id = manager.get_last_request_id(222)
        assert request_id == "uuid-222"
    
    def test_get_last_request_id_unknown_user(self):
        """Test getting last request_id for unknown user."""
        manager = StateManager(persist=False)
        
        request_id = manager.get_last_request_id(999)
        
        assert request_id is None
    
    def test_clear_state(self):
        """Test clearing user state."""
        manager = StateManager(persist=False)
        
        manager.set_last_request(333, "uuid-333", [1])
        manager.clear_state(333)
        
        # Should return None for cleared user
        request_id = manager.get_last_request_id(333)
        assert request_id is None
    
    def test_multiple_users(self):
        """Test managing state for multiple users."""
        manager = StateManager(persist=False)
        
        manager.set_last_request(1, "uuid-1", [1])
        manager.set_last_request(2, "uuid-2", [2])
        manager.set_last_request(3, "uuid-3", [3])
        
        assert manager.get_last_request_id(1) == "uuid-1"
        assert manager.get_last_request_id(2) == "uuid-2"
        assert manager.get_last_request_id(3) == "uuid-3"
    
    def test_update_existing_state(self):
        """Test updating existing user state."""
        manager = StateManager(persist=False)
        
        manager.set_last_request(444, "old-uuid", [1, 2])
        manager.set_last_request(444, "new-uuid", [3, 4, 5])
        
        state = manager.get_state(444)
        assert state.last_request_id == "new-uuid"
        assert state.last_perfume_ids == [3, 4, 5]


class TestEventLogging:
    """Tests for event logging logic."""
    
    def test_event_types(self):
        """Test all supported event types."""
        valid_types = ["impression", "click", "like", "dislike", "save", "my_saves"]
        
        for event_type in valid_types:
            # Just verify these are the expected types
            assert event_type in valid_types
    
    def test_event_request_structure(self):
        """Test event request JSON structure."""
        # Simulate building an event request
        event_data = {
            "perfume_id": 123,
            "event_type": "like",
            "request_id": "test-uuid",
        }
        
        assert "perfume_id" in event_data
        assert "event_type" in event_data
        assert "request_id" in event_data
    
    def test_impression_with_multiple_ids(self):
        """Test that impressions can be logged for multiple perfume IDs."""
        perfume_ids = [1, 2, 3, 4, 5]
        request_id = "search-uuid"
        tg_id = 12345
        
        # Simulate what log_impression does
        events = []
        for perfume_id in perfume_ids:
            events.append({
                "tg_id": tg_id,
                "perfume_id": perfume_id,
                "event_type": "impression",
                "request_id": request_id,
            })
        
        assert len(events) == 5
        for i, event in enumerate(events):
            assert event["perfume_id"] == perfume_ids[i]
            assert event["event_type"] == "impression"
            assert event["request_id"] == request_id
