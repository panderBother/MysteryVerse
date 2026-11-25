from __future__ import annotations

import asyncio
import hashlib
import json
from typing import Any

from pydantic import ValidationError

from mysteryverse_ai.tools.models import (
    ToolExecution,
    ToolHandler,
    ToolInvocationContext,
    ToolResult,
    ToolSpec,
)


class ToolRegistry:
    def __init__(self) -> None:
        self._handlers: dict[str, ToolHandler] = {}
        self._specs: dict[str, ToolSpec] = {}

    def register(self, spec: ToolSpec, handler: ToolHandler) -> None:
        if spec.name in self._specs:
            raise ValueError(f"tool already registered: {spec.name}")
        self._specs[spec.name] = spec
        self._handlers[spec.name] = handler

    def describe(self, allowed_tools: set[str]) -> list[ToolSpec]:
        return [spec for name, spec in sorted(self._specs.items()) if name in allowed_tools]

    async def invoke(
        self,
        *,
        call_id: str,
        tool_name: str,
        arguments: dict[str, Any],
        allowed_tools: set[str],
        allowed_permissions: set[str],
        context: ToolInvocationContext,
    ) -> ToolExecution:
        arguments_hash = hash_json(arguments)
        spec = self._specs.get(tool_name)
        if spec is None or tool_name not in allowed_tools:
            return _denied(call_id, tool_name, arguments_hash, "FORBIDDEN_ACTION")
        if context.actor_type not in spec.allowed_actor_types:
            return _denied(call_id, tool_name, arguments_hash, "FORBIDDEN_ACTION")
        if not spec.permissions.issubset(allowed_permissions):
            return _denied(call_id, tool_name, arguments_hash, "FORBIDDEN_ACTION")
        if (
            spec.source_kind != "local"
            or spec.trust_level != "local_builtin"
            or spec.external_network
        ):
            return _denied(call_id, tool_name, arguments_hash, "TOOL_TRUST_BLOCKED")
        if spec.requires_confirmation and tool_name not in context.confirmed_tools:
            return _denied(
                call_id,
                tool_name,
                arguments_hash,
                "TOOL_CONFIRMATION_REQUIRED",
            )
        try:
            parsed = spec.args_model.model_validate(arguments)
        except ValidationError:
            return ToolExecution(
                call_id=call_id,
                tool_name=tool_name,
                status="failed",
                arguments_hash=arguments_hash,
                duration_ms=0,
                error_code="VALIDATION_ERROR",
            )

        started = asyncio.get_running_loop().time()
        try:
            async with asyncio.timeout(spec.timeout_ms / 1000):
                result = await self._handlers[tool_name](parsed.model_dump(), context)
        except TimeoutError:
            return ToolExecution(
                call_id=call_id,
                tool_name=tool_name,
                status="timeout",
                arguments_hash=arguments_hash,
                duration_ms=_elapsed_ms(started),
                error_code="AI_TIMEOUT",
            )
        except Exception:
            return ToolExecution(
                call_id=call_id,
                tool_name=tool_name,
                status="failed",
                arguments_hash=arguments_hash,
                duration_ms=_elapsed_ms(started),
                error_code="TOOL_EXECUTION_FAILED",
            )
        if not isinstance(result, ToolResult):
            return ToolExecution(
                call_id=call_id,
                tool_name=tool_name,
                status="failed",
                arguments_hash=arguments_hash,
                duration_ms=_elapsed_ms(started),
                error_code="TOOL_EXECUTION_FAILED",
            )
        result_size = len(
            json.dumps(
                {"content": result.content, "summary": result.summary},
                ensure_ascii=False,
                separators=(",", ":"),
            ).encode()
        )
        if result_size > spec.max_result_bytes:
            return ToolExecution(
                call_id=call_id,
                tool_name=tool_name,
                status="failed",
                arguments_hash=arguments_hash,
                duration_ms=_elapsed_ms(started),
                error_code="TOOL_RESULT_TOO_LARGE",
            )
        return ToolExecution(
            call_id=call_id,
            tool_name=tool_name,
            status="succeeded",
            arguments_hash=arguments_hash,
            duration_ms=_elapsed_ms(started),
            content=result.content,
            summary=result.summary,
        )


def _denied(
    call_id: str,
    tool_name: str,
    arguments_hash: str,
    error_code: str,
) -> ToolExecution:
    return ToolExecution(
        call_id=call_id,
        tool_name=tool_name,
        status="denied",
        arguments_hash=arguments_hash,
        duration_ms=0,
        error_code=error_code,
    )


def _elapsed_ms(started: float) -> int:
    return max(round((asyncio.get_running_loop().time() - started) * 1000), 0)


def hash_json(value: dict[str, Any]) -> str:
    raw = json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))
    return hashlib.sha256(raw.encode()).hexdigest()
