from __future__ import annotations

from typing import Any, TypedDict


class AgentGraphState(TypedDict, total=False):
    messages: list[dict[str, str]]
    steps: list[dict[str, Any]]
    tool_calls: list[dict[str, Any]]
    usage: dict[str, Any]
    step_count: int
    next_action: str | None
    pending_tool_name: str | None
    pending_arguments: dict[str, object]
    pending_output: dict[str, object] | None
    status: str | None
    error_code: str | None
    error_message: str | None
    error_retryable: bool
    structured_output: dict[str, object] | None
    validation_errors: list[dict[str, str]]
    output_schema_key: str
    output_schema_version: int
