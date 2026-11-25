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
    internal_signing_key_id: str = "dev-go-core-v1"
    internal_request_signing_previous_key_id: str | None = None
    internal_request_signing_key: SecretStr | None = None
    internal_request_signing_previous_key: SecretStr | None = None
    internal_revoked_key_ids: str = ""
    internal_signature_window_seconds: int = Field(default=300, ge=1, le=3600)
    internal_required_agent_run_scope: str = "ai:agent:run"
    internal_nonce_backend: Literal["memory", "redis"] = "memory"
    internal_nonce_key_prefix: str = "mv:internal:nonce"
    internal_nonce_store_timeout_seconds: float = Field(default=1.0, gt=0, le=30)
    internal_kms_provider: Literal["environment", "external"] = "environment"
    internal_kms_uri: str | None = None
    internal_mtls_enabled: bool = False
    internal_tls_cert_file: str | None = None
    internal_tls_key_file: str | None = None
    internal_tls_ca_file: str | None = None
    internal_tls_require_client_cert: bool = True
    allow_restricted_model_context: bool = False
    log_payloads: bool = False

    @property
    def internal_revoked_key_ids_set(self) -> frozenset[str]:
        return frozenset(
            item.strip() for item in self.internal_revoked_key_ids.split(",") if item.strip()
        )

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
        self._validate_internal_security()
        return self

    def _validate_internal_security(self) -> None:
        active_secret = (
            self.internal_request_signing_key.get_secret_value()
            if self.internal_request_signing_key is not None
            else ""
        )
        previous_secret = (
            self.internal_request_signing_previous_key.get_secret_value()
            if self.internal_request_signing_previous_key is not None
            else ""
        )
        if active_secret:
            if not self.internal_signing_key_id:
                raise ValueError("AI_INTERNAL_SIGNING_KEY_ID is required")
            if len(active_secret) < 32:
                raise ValueError("AI_INTERNAL_REQUEST_SIGNING_KEY must be at least 32 bytes")
        if previous_secret:
            if not self.internal_request_signing_previous_key_id:
                raise ValueError("AI_INTERNAL_REQUEST_SIGNING_PREVIOUS_KEY_ID is required")
            if len(previous_secret) < 32:
                raise ValueError(
                    "AI_INTERNAL_REQUEST_SIGNING_PREVIOUS_KEY must be at least 32 bytes"
                )
        if (
            self.internal_signing_key_id
            and self.internal_request_signing_previous_key_id
            and self.internal_signing_key_id == self.internal_request_signing_previous_key_id
        ):
            raise ValueError("active and previous signing key IDs must differ")
        if self.environment != "production":
            return
        if (
            self.internal_service_token is None
            or not self.internal_service_token.get_secret_value()
        ):
            raise ValueError("production internal service token is required")
        if not active_secret:
            raise ValueError("production request signing is required")
        if self.internal_nonce_backend != "redis" or not self.redis_url:
            raise ValueError("production nonce storage requires Redis")
        if not self.internal_mtls_enabled:
            raise ValueError("production internal service mTLS is required")
        if self.internal_kms_provider != "external":
            raise ValueError("production signing keys require an external KMS provider")

    @classmethod
    def for_test(cls) -> "Settings":
        return cls(_env_file=None)  # type: ignore[call-arg]


@lru_cache(maxsize=1)
def get_settings() -> Settings:
    return Settings()
