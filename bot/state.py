"""
User state management for tracking request_ids per user.

Uses in-memory storage with optional persistence to a local JSON file.
"""
import json
import logging
import os
from dataclasses import dataclass, field
from typing import Dict, Optional
from pathlib import Path

logger = logging.getLogger(__name__)

# State file path (in bot directory)
STATE_FILE = Path(__file__).parent / ".bot_state.json"


@dataclass
class UserState:
    """State for a single user."""
    last_request_id: Optional[str] = None
    last_perfume_ids: list = field(default_factory=list)
    
    def to_dict(self) -> dict:
        return {
            "last_request_id": self.last_request_id,
            "last_perfume_ids": self.last_perfume_ids,
        }
    
    @classmethod
    def from_dict(cls, data: dict) -> "UserState":
        return cls(
            last_request_id=data.get("last_request_id"),
            last_perfume_ids=data.get("last_perfume_ids", []),
        )


class StateManager:
    """Manages user states with optional persistence."""
    
    def __init__(self, persist: bool = False):
        self._states: Dict[int, UserState] = {}
        self._persist = persist
        if persist:
            self._load()
    
    def _load(self) -> None:
        """Load state from file."""
        if STATE_FILE.exists():
            try:
                with open(STATE_FILE, "r") as f:
                    data = json.load(f)
                    for tg_id_str, state_data in data.items():
                        self._states[int(tg_id_str)] = UserState.from_dict(state_data)
                logger.info("Loaded state for %d users", len(self._states))
            except Exception as e:
                logger.error("Failed to load state: %s", e)
    
    def _save(self) -> None:
        """Save state to file."""
        if not self._persist:
            return
        try:
            data = {str(tg_id): state.to_dict() for tg_id, state in self._states.items()}
            with open(STATE_FILE, "w") as f:
                json.dump(data, f)
        except Exception as e:
            logger.error("Failed to save state: %s", e)
    
    def get_state(self, tg_id: int) -> UserState:
        """Get or create user state."""
        if tg_id not in self._states:
            self._states[tg_id] = UserState()
        return self._states[tg_id]
    
    def set_last_request(
        self,
        tg_id: int,
        request_id: str,
        perfume_ids: list,
    ) -> None:
        """Update user's last request info."""
        state = self.get_state(tg_id)
        state.last_request_id = request_id
        state.last_perfume_ids = perfume_ids
        self._save()
    
    def get_last_request_id(self, tg_id: int) -> Optional[str]:
        """Get user's last request_id."""
        return self.get_state(tg_id).last_request_id
    
    def clear_state(self, tg_id: int) -> None:
        """Clear user state."""
        if tg_id in self._states:
            del self._states[tg_id]
            self._save()


# Global state manager instance (in-memory by default)
state_manager = StateManager(persist=False)
