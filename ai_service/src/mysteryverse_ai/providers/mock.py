from __future__ import annotations

import asyncio
import json
from collections.abc import AsyncIterator
from dataclasses import dataclass, field

from mysteryverse_ai.providers.models import (
    ModelRequest,
    ProviderError,
    ProviderResult,
    ProviderUsage,
)


@dataclass(slots=True)
class MockProviderStep:
    content: str | None = None
    error: ProviderError | None = None
    delay_seconds: float = 0
    usage: ProviderUsage = field(default_factory=ProviderUsage)


class MockProvider:
    def __init__(
        self,
        steps: list[MockProviderStep] | None = None,
        *,
        model_name: str = "mock-v1",
    ) -> None:
        self._steps = list(steps or [])
        self._model_name = model_name

    @property
    def provider_name(self) -> str:
        return "mock"

    async def generate(self, request: ModelRequest, *, timeout_ms: int) -> ProviderResult:
        if self._steps:
            step = self._steps.pop(0)
            if step.delay_seconds:
                await asyncio.sleep(step.delay_seconds)
            if step.error is not None:
                raise step.error
            return ProviderResult(
                content=step.content or "",
                model_name=request.model or self._model_name,
                usage=step.usage,
                finish_reason="stop",
            )
        content = self._default_content(request)
        return ProviderResult(
            content=content,
            model_name=request.model or self._model_name,
            usage=ProviderUsage(input_tokens=self._estimate_tokens(request), output_tokens=24),
            finish_reason="stop",
        )

    async def stream(self, request: ModelRequest, *, timeout_ms: int) -> AsyncIterator[str]:
        result = await self.generate(request, timeout_ms=timeout_ms)
        midpoint = max(len(result.content) // 2, 1)
        yield result.content[:midpoint]
        if result.content[midpoint:]:
            yield result.content[midpoint:]

    def _default_content(self, request: ModelRequest) -> str:
        tool_messages = [message for message in request.messages if message.role == "tool"]
        if tool_messages:
            summary = "Authorized context was returned by the tool."
            try:
                payload = json.loads(tool_messages[-1].content)
                if isinstance(payload, dict) and isinstance(payload.get("summary"), str):
                    summary = payload["summary"]
            except json.JSONDecodeError:
                pass
            return json.dumps(
                {
                    "action": "final",
                    "output": {
                        "answer": summary,
                        "confidence": 0.9,
                        "used_tool": True,
                    },
                },
                ensure_ascii=False,
            )
        return json.dumps(
            {
                "action": "tool",
                "tool_name": "mock_context_lookup",
                "arguments": {"topic": "demo"},
            },
            ensure_ascii=False,
        )

    @staticmethod
    def _estimate_tokens(request: ModelRequest) -> int:
        characters = sum(len(message.content) for message in request.messages)
        return max(characters // 4, 1)
