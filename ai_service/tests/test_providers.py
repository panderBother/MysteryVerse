from __future__ import annotations

import asyncio

import httpx

from mysteryverse_ai.providers.mock import MockProvider, MockProviderStep
from mysteryverse_ai.providers.models import (
    ModelRequest,
    ProviderError,
    ProviderMessage,
    ProviderUsage,
)
from mysteryverse_ai.providers.openai_compatible import OpenAICompatibleProvider
from mysteryverse_ai.providers.retrying import RetryingProvider


def test_mock_provider_and_stream() -> None:
    async def scenario() -> None:
        provider = MockProvider(
            [
                MockProviderStep(
                    content='{"action":"final","output":{}}',
                    usage=ProviderUsage(input_tokens=3, output_tokens=4),
                )
            ]
        )
        request = ModelRequest(messages=(ProviderMessage(role="user", content="hello"),))

        result = await provider.generate(request, timeout_ms=1000)
        assert result.content.startswith('{"action"')
        assert result.usage.input_tokens == 3

        chunks = [chunk async for chunk in provider.stream(request, timeout_ms=1000)]
        assert chunks

    asyncio.run(scenario())


def test_retrying_provider_retries_retryable_errors() -> None:
    async def scenario() -> None:
        provider = MockProvider(
            [
                MockProviderStep(
                    error=ProviderError(
                        code="PROVIDER_UNAVAILABLE",
                        message="temporary",
                        retryable=True,
                    )
                ),
                MockProviderStep(content="ok"),
            ]
        )
        router = RetryingProvider(provider, max_retries=1, backoff_ms=10)
        result = await router.generate(
            ModelRequest(messages=(ProviderMessage(role="user", content="hello"),)),
            timeout_ms=1000,
        )
        assert result.content == "ok"

    asyncio.run(scenario())


def test_openai_compatible_provider_parses_completion_and_stream() -> None:
    async def handler(request: httpx.Request) -> httpx.Response:
        payload = request.read().decode()
        if '"stream":true' in payload:
            return httpx.Response(
                200,
                text='data: {"choices":[{"delta":{"content":"hello"}}]}\n\ndata: [DONE]\n\n',
            )
        return httpx.Response(
            200,
            json={
                "model": "test-model",
                "choices": [
                    {
                        "message": {"content": '{"action":"final","output":{"ok":true}}'},
                        "finish_reason": "stop",
                    }
                ],
                "usage": {"prompt_tokens": 11, "completion_tokens": 7},
            },
        )

    async def scenario() -> None:
        client = httpx.AsyncClient(
            base_url="https://provider.invalid/v1",
            transport=httpx.MockTransport(handler),
        )
        provider = OpenAICompatibleProvider(
            base_url="https://provider.invalid/v1",
            api_key="test-key",
            model_name="test-model",
            client=client,
        )
        request = ModelRequest(messages=(ProviderMessage(role="user", content="hello"),))
        result = await provider.generate(request, timeout_ms=1000)
        assert result.model_name == "test-model"
        assert result.usage.input_tokens == 11
        chunks = [chunk async for chunk in provider.stream(request, timeout_ms=1000)]
        assert chunks == ["hello"]
        await client.aclose()

    asyncio.run(scenario())
