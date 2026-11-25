from __future__ import annotations

import asyncio

from mysteryverse_ai.providers.models import (
    ModelProvider,
    ModelRequest,
    ProviderError,
    ProviderResult,
)


class RetryingProvider:
    def __init__(
        self,
        provider: ModelProvider,
        *,
        max_retries: int,
        backoff_ms: int,
    ) -> None:
        self._provider = provider
        self._max_retries = max_retries
        self._backoff_ms = backoff_ms

    async def aclose(self) -> None:
        close = getattr(self._provider, "aclose", None)
        if close is not None:
            await close()

    async def generate(self, request: ModelRequest, *, timeout_ms: int) -> ProviderResult:
        last_error: ProviderError | None = None
        for attempt in range(self._max_retries + 1):
            try:
                return await self._provider.generate(request, timeout_ms=timeout_ms)
            except ProviderError as exc:
                last_error = exc
                if not exc.retryable or attempt >= self._max_retries:
                    raise
                await asyncio.sleep(self._backoff_ms * (2**attempt) / 1000)
        if last_error is not None:
            raise last_error
        raise RuntimeError("provider retry loop exited without a result")
