from __future__ import annotations

import asyncio

from pydantic import BaseModel, ConfigDict, Field

from mysteryverse_ai.tools.models import ToolInvocationContext, ToolResult, ToolSpec
from mysteryverse_ai.tools.registry import ToolRegistry


class _Args(BaseModel):
    model_config = ConfigDict(extra="forbid")
    value: str = Field(min_length=1)


def _context() -> ToolInvocationContext:
    return ToolInvocationContext(
        run_id="run",
        request_id="request",
        trace_id="trace",
        actor_type="dm",
        actor_id="dm:1",
    )


def test_tool_registry_validates_permissions_and_arguments() -> None:
    async def handler(arguments: dict[str, object], _context: ToolInvocationContext) -> ToolResult:
        return ToolResult(summary=str(arguments["value"]), content={"summary": arguments["value"]})

    async def scenario() -> None:
        registry = ToolRegistry()
        registry.register(
            ToolSpec(
                name="demo",
                description="demo tool",
                args_model=_Args,
                permissions=frozenset({"context:read"}),
                allowed_actor_types=frozenset({"dm"}),
            ),
            handler,
        )
        success = await registry.invoke(
            call_id="1",
            tool_name="demo",
            arguments={"value": "ok"},
            allowed_tools={"demo"},
            allowed_permissions={"context:read"},
            context=_context(),
        )
        denied = await registry.invoke(
            call_id="2",
            tool_name="demo",
            arguments={"value": "ok"},
            allowed_tools={"demo"},
            allowed_permissions=set(),
            context=_context(),
        )
        invalid = await registry.invoke(
            call_id="3",
            tool_name="demo",
            arguments={},
            allowed_tools={"demo"},
            allowed_permissions={"context:read"},
            context=_context(),
        )
        assert success.status == "succeeded"
        assert denied.status == "denied"
        assert invalid.error_code == "VALIDATION_ERROR"

    asyncio.run(scenario())


def test_tool_timeout_and_failure_are_structured() -> None:
    async def slow_handler(
        _arguments: dict[str, object],
        _context: ToolInvocationContext,
    ) -> ToolResult:
        await asyncio.sleep(0.2)
        return ToolResult(summary="late")

    async def failing_handler(
        _arguments: dict[str, object],
        _context: ToolInvocationContext,
    ) -> ToolResult:
        raise RuntimeError("secret upstream detail")

    async def scenario() -> None:
        registry = ToolRegistry()
        registry.register(
            ToolSpec(
                name="slow",
                description="slow",
                args_model=_Args,
                permissions=frozenset(),
                allowed_actor_types=frozenset({"dm"}),
                timeout_ms=10,
            ),
            slow_handler,
        )
        registry.register(
            ToolSpec(
                name="fail",
                description="fail",
                args_model=_Args,
                permissions=frozenset(),
                allowed_actor_types=frozenset({"dm"}),
            ),
            failing_handler,
        )
        timed = await registry.invoke(
            call_id="1",
            tool_name="slow",
            arguments={"value": "ok"},
            allowed_tools={"slow"},
            allowed_permissions=set(),
            context=_context(),
        )
        failed = await registry.invoke(
            call_id="2",
            tool_name="fail",
            arguments={"value": "ok"},
            allowed_tools={"fail"},
            allowed_permissions=set(),
            context=_context(),
        )
        assert timed.status == "timeout"
        assert failed.status == "failed"
        assert "secret upstream detail" not in failed.model_dump_json()

    asyncio.run(scenario())


def test_external_untrusted_and_confirmation_tools_are_blocked() -> None:
    async def handler(
        _arguments: dict[str, object],
        _context: ToolInvocationContext,
    ) -> ToolResult:
        return ToolResult(summary="ok")

    async def scenario() -> None:
        registry = ToolRegistry()
        registry.register(
            ToolSpec(
                name="mcp_tool",
                description="future MCP",
                args_model=_Args,
                permissions=frozenset(),
                allowed_actor_types=frozenset({"dm"}),
                source_kind="mcp",
                trust_level="trusted_external",
                mcp_server="future-server",
            ),
            handler,
        )
        registry.register(
            ToolSpec(
                name="high_risk",
                description="confirmation required",
                args_model=_Args,
                permissions=frozenset(),
                allowed_actor_types=frozenset({"dm"}),
                requires_confirmation=True,
            ),
            handler,
        )
        external = await registry.invoke(
            call_id="1",
            tool_name="mcp_tool",
            arguments={"value": "ok"},
            allowed_tools={"mcp_tool"},
            allowed_permissions=set(),
            context=_context(),
        )
        unconfirmed = await registry.invoke(
            call_id="2",
            tool_name="high_risk",
            arguments={"value": "ok"},
            allowed_tools={"high_risk"},
            allowed_permissions=set(),
            context=_context(),
        )
        confirmed = await registry.invoke(
            call_id="3",
            tool_name="high_risk",
            arguments={"value": "ok"},
            allowed_tools={"high_risk"},
            allowed_permissions=set(),
            context=ToolInvocationContext(
                run_id="run",
                request_id="request",
                trace_id="trace",
                actor_type="dm",
                actor_id="dm:1",
                confirmed_tools=frozenset({"high_risk"}),
            ),
        )
        assert external.error_code == "TOOL_TRUST_BLOCKED"
        assert unconfirmed.error_code == "TOOL_CONFIRMATION_REQUIRED"
        assert confirmed.status == "succeeded"

    asyncio.run(scenario())


def test_tool_result_size_is_bounded() -> None:
    async def large_handler(
        _arguments: dict[str, object],
        _context: ToolInvocationContext,
    ) -> ToolResult:
        return ToolResult(summary="x" * 1000, content={"value": "x" * 1000})

    async def scenario() -> None:
        registry = ToolRegistry()
        registry.register(
            ToolSpec(
                name="large",
                description="large result",
                args_model=_Args,
                permissions=frozenset(),
                allowed_actor_types=frozenset({"dm"}),
                max_result_bytes=100,
            ),
            large_handler,
        )
        result = await registry.invoke(
            call_id="1",
            tool_name="large",
            arguments={"value": "ok"},
            allowed_tools={"large"},
            allowed_permissions=set(),
            context=_context(),
        )
        assert result.error_code == "TOOL_RESULT_TOO_LARGE"

    asyncio.run(scenario())
