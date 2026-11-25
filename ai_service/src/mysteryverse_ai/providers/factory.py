from __future__ import annotations

from mysteryverse_ai.config import Settings
from mysteryverse_ai.providers.mock import MockProvider
from mysteryverse_ai.providers.models import ModelProvider
from mysteryverse_ai.providers.openai_compatible import OpenAICompatibleProvider
from mysteryverse_ai.providers.retrying import RetryingProvider


def build_provider(settings: Settings) -> RetryingProvider:
    provider: ModelProvider
    if settings.model_provider == "mock":
        provider = MockProvider(model_name=settings.model_name)
    elif settings.model_provider == "openai_compatible":
        if settings.model_api_key is None:
            raise ValueError("AI_MODEL_API_KEY is required")
        provider = OpenAICompatibleProvider(
            base_url=settings.model_base_url,
            api_key=settings.model_api_key.get_secret_value(),
            model_name=settings.model_name,
        )
    else:
        raise ValueError(f"unsupported model provider: {settings.model_provider}")
    return RetryingProvider(
        provider,
        max_retries=settings.model_max_retries,
        backoff_ms=settings.model_retry_backoff_ms,
    )
