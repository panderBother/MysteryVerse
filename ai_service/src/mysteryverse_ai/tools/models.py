from __future__ import annotations

from collections.abc import Awaitable, Callable
from dataclasses import dataclass
from typing import Any, Literal, Protocol

from pydantic import BaseModel, ConfigDict, Field


class ToolResult(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    content: dict[str, Any] = Field(default_factory=dict)
    summary: str


class ToolExecution(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    call_id: str
    tool_name: str
    status: Literal["succeeded", "failed", "denied", "timeout"]
    arguments_hash: str
    duration_ms: int = Field(ge=0)
    content: dict[str, Any] | None = None
    summary: str | None = None
    error_code: str | None = None


@dataclass(frozen=True, slots=True)
class ToolSpec:
    name: str
    description: str
    args_model: type[BaseModel]
    permissions: frozenset[str]
    allowed_actor_types: frozenset[str]
    timeout_ms: int = 2_000
    source_kind: Literal["local", "mcp", "http"] = "local"
    trust_level: Literal["local_builtin", "trusted_external", "untrusted_external"] = (
        "local_builtin"
    )
    external_network: bool = False
    requires_confirmation: bool = False
    max_result_bytes: int = 16_384
    mcp_server: str | None = None

    def parameter_schema(self) -> dict[str, Any]:
        return self.args_model.model_json_schema()


@dataclass(frozen=True, slots=True)
class ToolInvocationContext:
    run_id: str
    request_id: str
    trace_id: str
    actor_type: str
    actor_id: str
    confirmed_tools: frozenset[str] = frozenset()


ToolHandler = Callable[
    [dict[str, Any], ToolInvocationContext],
    Awaitable[ToolResult],
]


class ToolSource(Protocol):
    """Future adapter boundary for MCP-backed or other external tool sources."""

    async def discover(self) -> list[ToolSpec]: ...

    async def invoke(
        self,
        *,
        tool_name: str,
        arguments: dict[str, Any],
        context: ToolInvocationContext,
    ) -> ToolResult: ...
