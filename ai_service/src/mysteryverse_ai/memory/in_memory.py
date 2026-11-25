from __future__ import annotations

import asyncio

from mysteryverse_ai.memory.models import AgentMemoryView, MemoryMessage, MemoryScope


class InMemoryMemoryStore:
    def __init__(self) -> None:
        self._messages: dict[str, list[MemoryMessage]] = {}
        self._lock = asyncio.Lock()

    async def load(
        self,
        scope: MemoryScope,
        *,
        max_messages: int,
    ) -> AgentMemoryView:
        key = self._scope_key(scope)
        async with self._lock:
            messages = list(self._messages.get(key, []))
        return AgentMemoryView(recent_messages=messages[-max_messages:])

    async def append_short_term(
        self,
        scope: MemoryScope,
        messages: list[MemoryMessage],
        *,
        max_messages: int,
    ) -> None:
        key = self._scope_key(scope)
        async with self._lock:
            existing = self._messages.setdefault(key, [])
            existing.extend(messages)
            del existing[:-max_messages]

    @staticmethod
    def _scope_key(scope: MemoryScope) -> str:
        return ":".join(
            [
                scope.actor_type,
                scope.actor_id,
                scope.game_id or "",
            ]
        )
