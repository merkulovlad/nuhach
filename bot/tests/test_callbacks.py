"""
Tests for callback data parsing.
"""
import pytest
from bot.keyboards import parse_callback_data, build_perfume_keyboard
from bot.api_client import PerfumeItem


class TestCallbackParsing:
    """Tests for callback data parsing."""
    
    def test_parse_details_callback(self):
        """Test parsing details callback."""
        action, value = parse_callback_data("details:123")
        
        assert action == "details"
        assert value == 123
    
    def test_parse_similar_callback(self):
        """Test parsing similar callback."""
        action, value = parse_callback_data("similar:456")
        
        assert action == "similar"
        assert value == 456
    
    def test_parse_like_callback(self):
        """Test parsing like callback."""
        action, value = parse_callback_data("like:789")
        
        assert action == "like"
        assert value == 789
    
    def test_parse_dislike_callback(self):
        """Test parsing dislike callback."""
        action, value = parse_callback_data("dislike:111")
        
        assert action == "dislike"
        assert value == 111
    
    def test_parse_save_callback(self):
        """Test parsing save callback."""
        action, value = parse_callback_data("save:222")
        
        assert action == "save"
        assert value == 222
    
    def test_parse_select_callback(self):
        """Test parsing select callback."""
        action, value = parse_callback_data("select:333")
        
        assert action == "select"
        assert value == 333
    
    def test_parse_page_callback(self):
        """Test parsing page callback."""
        action, value = parse_callback_data("page:2")
        
        assert action == "page"
        assert value == 2
    
    def test_parse_back_callback(self):
        """Test parsing back callback (no colon)."""
        action, value = parse_callback_data("back")
        
        assert action == "back"
        assert value is None
    
    def test_parse_noop_callback(self):
        """Test parsing noop callback."""
        action, value = parse_callback_data("noop")
        
        assert action == "noop"
        assert value is None
    
    def test_parse_callback_with_string_value(self):
        """Test parsing callback with non-integer value."""
        action, value = parse_callback_data("test:abc")
        
        assert action == "test"
        assert value == "abc"  # String when not parseable as int
    
    def test_parse_callback_with_zero(self):
        """Test parsing callback with zero."""
        action, value = parse_callback_data("test:0")
        
        assert action == "test"
        assert value == 0


class TestKeyboardBuilding:
    """Tests for keyboard building."""
    
    def create_test_perfume(self, id: int = 1, name: str = "Test") -> PerfumeItem:
        """Create a test perfume."""
        return PerfumeItem(
            id=id,
            name=name,
            brand="Test Brand",
            rating_value=4.0,
            rating_count=100,
            year=2020,
        )
    
    def test_build_perfume_keyboard_full(self):
        """Test building full keyboard."""
        perfume = self.create_test_perfume(id=123)
        keyboard = build_perfume_keyboard(perfume)
        
        # Should have 2 rows
        assert len(keyboard.inline_keyboard) == 2
        
        # First row: Details, Similar
        assert len(keyboard.inline_keyboard[0]) == 2
        assert keyboard.inline_keyboard[0][0].callback_data == "details:123"
        assert keyboard.inline_keyboard[0][1].callback_data == "similar:123"
        
        # Second row: Like, Dislike, Save
        assert len(keyboard.inline_keyboard[1]) == 3
        assert keyboard.inline_keyboard[1][0].callback_data == "like:123"
        assert keyboard.inline_keyboard[1][1].callback_data == "dislike:123"
        assert keyboard.inline_keyboard[1][2].callback_data == "save:123"
    
    def test_build_perfume_keyboard_no_details(self):
        """Test building keyboard without details button."""
        perfume = self.create_test_perfume(id=456)
        keyboard = build_perfume_keyboard(perfume, show_details=False)
        
        # First row should only have Similar
        assert len(keyboard.inline_keyboard[0]) == 1
        assert keyboard.inline_keyboard[0][0].callback_data == "similar:456"
    
    def test_build_perfume_keyboard_no_reactions(self):
        """Test building keyboard without reaction buttons."""
        perfume = self.create_test_perfume(id=789)
        keyboard = build_perfume_keyboard(perfume, show_reactions=False)
        
        # Should have only 1 row (Details, Similar)
        assert len(keyboard.inline_keyboard) == 1
        assert len(keyboard.inline_keyboard[0]) == 2
