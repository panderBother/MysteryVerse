from functools import lru_cache
from typing import Literal

from pydantic import Field, SecretStr, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_prefix="AI_",
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
        frozen=True,
    )

    environment: Literal["development", "test", "staging", "production"] = "development"
    log_level: Literal["DEBUG", "INFO", "WARNING", "ERROR"] = "INFO"
    host: str = "127.0.0.1"
    port: int = Field(default=8000, ge=1, le=65535)
    database_url: str | None = None
    redis_url: str | None = None

    model_provider: Literal["mock", "openai_compatible"] = "mock"
    model_name: str = "mock-v1"
    model_base_url: str = "https://api.openai.com/v1"
    model_api_key: SecretStr | None = None
    model_timeout_ms: int = Field(default=10_000, ge=1)
    model_max_retries: int = Field(default=2, ge=0, le=5)
    model_retry_backoff_ms: int = Field(default=100, ge=10, le=5_000)

    agent_default_timeout_ms: int = Field(default=15_000, ge=1)
    agent_default_max_steps: int = Field(default=6, ge=1, le=20)
    agent_default_context_tokens: int = Field(default=4_000, ge=128)
    agent_run_ttl_seconds: int = Field(default=86_400, ge=60)

    internal_service_id: str = "go-core"
    internal_service_token: SecretStr | None = None
    internal_request_signing_key: SecretStr | None = None
    internal_request_signing_previous_key: SecretStr | None = None
    internal_signature_window_seconds: int = Field(default=300, ge=1, le=3600)
    allow_restricted_model_context: bool = False
    log_payloads: bool = False

    @model_validator(mode="after")
    def validate_model_provider(self) -> "Settings":
        if self.model_provider == "openai_compatible":
            if self.model_api_key is None or not self.model_api_key.get_secret_value():
                raise ValueError("AI_MODEL_API_KEY is required for openai_compatible provider")
            if not self.model_name or not self.model_base_url:
                raise ValueError("AI_MODEL_NAME and AI_MODEL_BASE_URL are required")
        retry_budget_ms = self.model_timeout_ms * max(self.model_max_retries + 1, 1)
        if self.agent_default_timeout_ms > retry_budget_ms:
            raise ValueError("agent default timeout must fit the configured model retry budget")
        return self

    @classmethod
    def for_test(cls) -> "Settings":
        return cls(_env_file=None)  # type: ignore[call-arg]


@lru_cache(maxsize=1)
def get_settings() -> Settings:
    return Settings()
