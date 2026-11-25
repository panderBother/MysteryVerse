from __future__ import annotations

import asyncio
import time
from collections.abc import Callable
from typing import Protocol

from redis.asyncio import Redis

from mysteryverse_ai.config import Settings


class NonceStore(Protocol):
    async def accept(self, key: str, *, ttl_seconds: int) -> bool: ...

    async def close(self) -> None: ...


class InMemoryNonceStore:
    def __init__(self, *, clock: Callable[[], float] = time.time) -> None:
        self._clock = clock
        self._entries: dict[str, float] = {}
        self._lock = asyncio.Lock()

    async def accept(self, key: str, *, ttl_seconds: int) -> bool:
        now = self._clock()
        async with self._lock:
            expired = [item for item, expiry in self._entries.items() if expiry <= now]
            for item in expired:
                self._entries.pop(item, None)
            if key in self._entries:
                return False
            self._entries[key] = now + ttl_seconds
            return True

    async def close(self) -> None:
        self._entries.clear()


class RedisNonceStore:
    def __init__(self, *, client: Redis, key_prefix: str) -> None:
        self._client = client
        self._key_prefix = key_prefix.rstrip(":")

    async def accept(self, key: str, *, ttl_seconds: int) -> bool:
        result = await self._client.set(
            f"{self._key_prefix}:{key}",
            "1",
            ex=ttl_seconds,
            nx=True,
        )
        return bool(result)

    async def close(self) -> None:
        await self._client.aclose()


def build_nonce_store(settings: Settings) -> NonceStore:
    if settings.internal_nonce_backend == "memory":
        return InMemoryNonceStore()
    if not settings.redis_url:
        raise ValueError("AI_REDIS_URL is required for the redis nonce backend")
    client = Redis.from_url(
        settings.redis_url,
        encoding="utf-8",
        decode_responses=False,
        socket_timeout=settings.internal_nonce_store_timeout_seconds,
        socket_connect_timeout=settings.internal_nonce_store_timeout_seconds,
    )
    return RedisNonceStore(
        client=client,
        key_prefix=settings.internal_nonce_key_prefix,
    )
