from __future__ import annotations

import asyncio
from dataclasses import dataclass


class IdempotencyConflictError(Exception):
    pass


@dataclass(frozen=True, slots=True)
class _IdempotencyRecord:
    run_id: str
    request_hash: str


class InMemoryRunIdempotencyStore:
    def __init__(self) -> None:
        self._records: dict[tuple[str, str], _IdempotencyRecord] = {}
        self._lock = asyncio.Lock()

    async def claim(
        self,
        *,
        scope: str,
        idempotency_key: str,
        request_hash: str,
        run_id: str,
    ) -> str:
        key = (scope, idempotency_key)
        async with self._lock:
            existing = self._records.get(key)
            if existing is not None:
                if existing.request_hash != request_hash:
                    raise IdempotencyConflictError(
                        "idempotency key was reused with a different request"
                    )
                return existing.run_id
            self._records[key] = _IdempotencyRecord(
                run_id=run_id,
                request_hash=request_hash,
            )
            return run_id
