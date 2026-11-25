from __future__ import annotations

from typing import Protocol

from mysteryverse_ai.memory.models import AgentMemoryView, MemoryMessage, MemoryScope


class MemoryStore(Protocol):
    async def load(
        self,
        scope: MemoryScope,
        *,
        max_messages: int,
    ) -> AgentMemoryView: ...

    async def append_short_term(
        self,
        scope: MemoryScope,
        messages: list[MemoryMessage],
        *,
        max_messages: int,
    ) -> None: ...
