from __future__ import annotations

from collections.abc import AsyncIterator
from typing import Literal, Protocol

from pydantic import BaseModel, ConfigDict, Field


class ProviderMessage(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    role: Literal["system", "user", "assistant", "tool"]
    content: str


class ModelRequest(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    messages: tuple[ProviderMessage, ...]
    max_output_tokens: int = Field(default=512, ge=1)
    model: str | None = None
    metadata: dict[str, str] = Field(default_factory=dict)


class ProviderUsage(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    input_tokens: int = Field(default=0, ge=0)
    output_tokens: int = Field(default=0, ge=0)
    cost_usd: str | None = None


class ProviderResult(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    content: str
    model_name: str
    usage: ProviderUsage = Field(default_factory=ProviderUsage)
    finish_reason: str | None = None


class ProviderError(Exception):
    def __init__(
        self,
        *,
        code: str,
        message: str,
        retryable: bool,
        status_code: int | None = None,
    ) -> None:
        super().__init__(message)
        self.code = code
        self.message = message
        self.retryable = retryable
        self.status_code = status_code


class ModelProvider(Protocol):
    @property
    def provider_name(self) -> str: ...

    async def generate(self, request: ModelRequest, *, timeout_ms: int) -> ProviderResult: ...

    def stream(self, request: ModelRequest, *, timeout_ms: int) -> AsyncIterator[str]: ...


class ProviderRouter(Protocol):
    async def generate(self, request: ModelRequest, *, timeout_ms: int) -> ProviderResult: ...
