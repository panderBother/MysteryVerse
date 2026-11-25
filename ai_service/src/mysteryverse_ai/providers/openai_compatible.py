from __future__ import annotations

import asyncio
import json
from collections.abc import AsyncIterator
from typing import Any

import httpx
from pydantic import BaseModel, ConfigDict, Field

from mysteryverse_ai.providers.models import (
    ModelRequest,
    ProviderError,
    ProviderResult,
    ProviderUsage,
)


class _ChatMessage(BaseModel):
    content: str | None = None


class _ChatChoice(BaseModel):
    message: _ChatMessage
    finish_reason: str | None = None


class _ChatUsage(BaseModel):
    prompt_tokens: int = Field(default=0, ge=0)
    completion_tokens: int = Field(default=0, ge=0)


class _ChatCompletion(BaseModel):
    model_config = ConfigDict(extra="ignore")

    model: str | None = None
    choices: list[_ChatChoice]
    usage: _ChatUsage | None = None


class OpenAICompatibleProvider:
    def __init__(
        self,
        *,
        base_url: str,
        api_key: str,
        model_name: str,
        client: httpx.AsyncClient | None = None,
    ) -> None:
        self._model_name = model_name
        self._owns_client = client is None
        self._client = client or httpx.AsyncClient(
            base_url=base_url.rstrip("/"),
            headers={"Authorization": f"Bearer {api_key}"},
            timeout=None,
        )

    @property
    def provider_name(self) -> str:
        return "openai_compatible"

    async def aclose(self) -> None:
        if self._owns_client:
            await self._client.aclose()

    async def generate(self, request: ModelRequest, *, timeout_ms: int) -> ProviderResult:
        payload = self._payload(request, stream=False)
        try:
            async with asyncio.timeout(timeout_ms / 1000):
                response = await self._client.post("/chat/completions", json=payload)
        except TimeoutError as exc:
            raise ProviderError(
                code="AI_TIMEOUT",
                message="The model provider timed out.",
                retryable=True,
            ) from exc
        except httpx.HTTPError as exc:
            raise ProviderError(
                code="PROVIDER_UNAVAILABLE",
                message="The model provider is unavailable.",
                retryable=True,
            ) from exc
        self._raise_for_status(response)
        try:
            parsed = _ChatCompletion.model_validate(response.json())
        except (ValueError, json.JSONDecodeError) as exc:
            raise ProviderError(
                code="PROVIDER_UNAVAILABLE",
                message="The model provider returned an invalid response.",
                retryable=True,
            ) from exc
        if not parsed.choices:
            raise ProviderError(
                code="PROVIDER_UNAVAILABLE",
                message="The model provider returned no choices.",
                retryable=True,
            )
        content = parsed.choices[0].message.content
        if content is None:
            raise ProviderError(
                code="PROVIDER_UNAVAILABLE",
                message="The model provider returned empty content.",
                retryable=True,
            )
        usage = ProviderUsage()
        if parsed.usage is not None:
            usage = ProviderUsage(
                input_tokens=parsed.usage.prompt_tokens,
                output_tokens=parsed.usage.completion_tokens,
            )
        return ProviderResult(
            content=content,
            model_name=request.model or parsed.model or self._model_name,
            usage=usage,
            finish_reason=parsed.choices[0].finish_reason,
        )

    async def stream(self, request: ModelRequest, *, timeout_ms: int) -> AsyncIterator[str]:
        payload = self._payload(request, stream=True)
        try:
            async with asyncio.timeout(timeout_ms / 1000):
                async with self._client.stream(
                    "POST",
                    "/chat/completions",
                    json=payload,
                ) as response:
                    self._raise_for_status(response)
                    async for line in response.aiter_lines():
                        if not line.startswith("data: "):
                            continue
                        data = line[6:].strip()
                        if data == "[DONE]":
                            break
                        try:
                            payload_data: Any = json.loads(data)
                        except json.JSONDecodeError:
                            continue
                        choices = payload_data.get("choices", [])
                        if not choices:
                            continue
                        delta = choices[0].get("delta", {})
                        content = delta.get("content")
                        if isinstance(content, str) and content:
                            yield content
        except TimeoutError as exc:
            raise ProviderError(
                code="AI_TIMEOUT",
                message="The model provider stream timed out.",
                retryable=True,
            ) from exc
        except httpx.HTTPError as exc:
            raise ProviderError(
                code="PROVIDER_UNAVAILABLE",
                message="The model provider stream is unavailable.",
                retryable=True,
            ) from exc

    def _payload(self, request: ModelRequest, *, stream: bool) -> dict[str, Any]:
        return {
            "model": request.model or self._model_name,
            "messages": [
                {"role": message.role, "content": message.content} for message in request.messages
            ],
            "max_tokens": request.max_output_tokens,
            "response_format": {"type": "json_object"},
            "stream": stream,
        }

    @staticmethod
    def _raise_for_status(response: httpx.Response) -> None:
        if response.status_code < 400:
            return
        retryable = response.status_code == 429 or response.status_code >= 500
        raise ProviderError(
            code="PROVIDER_UNAVAILABLE",
            message="The model provider rejected the request.",
            retryable=retryable,
            status_code=response.status_code,
        )
