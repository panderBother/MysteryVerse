from __future__ import annotations

import asyncio
import json

from pydantic import BaseModel, ConfigDict, Field, SecretStr

from mysteryverse_ai.agent_runtime.factory import build_agent_runtime
from mysteryverse_ai.agent_runtime.outputs import OutputSchemaRegistry
from mysteryverse_ai.agent_runtime.runtime import AgentRuntime
from mysteryverse_ai.config import Settings
from mysteryverse_ai.context.builder import ContextBuilder
from mysteryverse_ai.memory.in_memory import InMemoryMemoryStore
from mysteryverse_ai.providers.mock import MockProvider, MockProviderStep
from mysteryverse_ai.providers.models import ModelRequest, ProviderError
from mysteryverse_ai.providers.retrying import RetryingProvider
from mysteryverse_ai.schemas.generated.models import (
    AgentExecutionStatus,
    AgentRequest,
    AgentTaskType,
)
from mysteryverse_ai.tools.mock import register_mock_tools
from mysteryverse_ai.tools.registry import ToolRegistry


def _request(
    *,
    max_steps: int = 4,
    timeout_ms: int = 1000,
    output_schema_key: str | None = None,
    output_schema_version: int | None = None,
) -> AgentRequest:
    payload: dict[str, object] = {
        "run_id": "01J9Z3K4M7Q2R8T5V6W1XE0000",
        "request_id": "runtime-request",
        "trace_id": "trace-1",
        "actor_type": "dm",
        "actor_id": "dm:1",
        "allowed_entity_ids": [],
        "requested_topics": ["demo"],
        "task_type": "demo",
        "task_goal": "Summarize the authorized demo context.",
        "context": {
            "task_context": "Use only authorized context.",
            "recent_messages": [],
            "history_summary": None,
            "user_preferences": {},
        },
        "allowed_tools": ["mock_context_lookup"],
        "allowed_permissions": ["context:read"],
        "execution_limits": {
            "timeout_ms": timeout_ms,
            "max_steps": max_steps,
            "max_context_tokens": 1000,
            "max_output_tokens": 200,
        },
        "schema_version": 1,
        "budget_ms": timeout_ms,
    }
    if output_schema_key is not None:
        payload["output_schema_key"] = output_schema_key
    if output_schema_version is not None:
        payload["output_schema_version"] = output_schema_version
    return AgentRequest.model_validate(payload)


def _settings() -> Settings:
    return Settings.for_test().model_copy(
        update={
            "environment": "test",
            "log_level": "ERROR",
            "internal_service_token": SecretStr("test-service-token"),
        }
    )


def _runtime(
    provider: MockProvider,
    output_schemas: OutputSchemaRegistry | None = None,
) -> AgentRuntime:
    tools = ToolRegistry()
    register_mock_tools(tools)

    return AgentRuntime(
        provider=RetryingProvider(provider, max_retries=0, backoff_ms=10),
        tools=tools,
        memory=InMemoryMemoryStore(),
        context_builder=ContextBuilder(),
        settings=_settings(),
        output_schemas=output_schemas,
    )


def test_demo_agent_uses_tool_and_returns_validated_output() -> None:
    async def scenario() -> None:
        runtime = _runtime(MockProvider())
        response = await runtime.run(_request())
        assert response.status == AgentExecutionStatus.succeeded
        assert response.structured_output is not None
        assert response.structured_output.model_dump()["answer"]
        assert response.tool_calls is not None and len(response.tool_calls) == 1
        assert response.steps is not None and {step.step_type.value for step in response.steps} >= {
            "context",
            "model",
            "tool",
            "validate",
        }
        assert response.trace_id == "trace-1"
        assert response.output_schema_key == "demo_result"
        assert response.output_schema_version == 1
        assert response.validation_errors == []

    asyncio.run(scenario())


def test_explicit_output_schema_key_and_version() -> None:
    class VersionedOutput(BaseModel):
        model_config = ConfigDict(extra="forbid")
        answer: str
        score: int = Field(ge=1, le=5)

    class CustomProvider(MockProvider):
        def _default_content(self, request: ModelRequest) -> str:
            _ = request
            return json.dumps(
                {
                    "action": "final",
                    "output": {"answer": "versioned", "score": 4},
                }
            )

    async def scenario() -> None:
        registry = OutputSchemaRegistry()
        registry.register(
            task_type=AgentTaskType.demo,
            key="demo_versioned",
            version=3,
            schema=VersionedOutput,
        )
        runtime = _runtime(CustomProvider(), registry)
        response = await runtime.run(
            _request(output_schema_key="demo_versioned", output_schema_version=3)
        )
        assert response.status == AgentExecutionStatus.succeeded
        assert response.output_schema_key == "demo_versioned"
        assert response.output_schema_version == 3

        unsupported_runtime = _runtime(CustomProvider(), registry)
        unsupported = await unsupported_runtime.run(
            _request(output_schema_key="demo_versioned", output_schema_version=9)
        )
        assert unsupported.status == AgentExecutionStatus.failed
        assert unsupported.error is not None
        assert unsupported.error.code.value == "OUTPUT_SCHEMA_VERSION_UNSUPPORTED"

    asyncio.run(scenario())


def test_max_steps_and_invalid_output_fail_without_authority_changes() -> None:
    async def scenario() -> None:
        runtime = _runtime(MockProvider())
        limited = await runtime.run(_request(max_steps=2))
        assert limited.status == AgentExecutionStatus.failed
        assert limited.error is not None
        assert limited.error.code.value == "WORKFLOW_FAILED"

        invalid_runtime = _runtime(
            MockProvider(
                [
                    MockProviderStep(
                        content=json.dumps(
                            {
                                "action": "final",
                                "output": {"answer": "", "confidence": 2, "used_tool": False},
                            }
                        )
                    )
                ]
            )
        )
        invalid = await invalid_runtime.run(_request())
        assert invalid.status == AgentExecutionStatus.failed
        assert invalid.error is not None
        assert invalid.error.code.value == "VALIDATION_ERROR"
        assert invalid.validation_errors
        assert {issue.path for issue in invalid.validation_errors} >= {"confidence"}

    asyncio.run(scenario())


def test_timeout_cancellation_and_provider_failure() -> None:
    async def scenario() -> None:
        slow_runtime = _runtime(MockProvider([MockProviderStep(content="{}", delay_seconds=0.2)]))
        timed = await slow_runtime.run(_request(timeout_ms=30))
        assert timed.status == AgentExecutionStatus.timeout
        assert timed.error is not None
        assert timed.error.code.value == "AI_TIMEOUT"

        cancel_event = asyncio.Event()
        cancel_runtime = _runtime(MockProvider([MockProviderStep(content="{}", delay_seconds=0.2)]))
        cancel_task = asyncio.create_task(
            cancel_runtime.run(
                _request(timeout_ms=1000),
                cancellation_event=cancel_event,
            )
        )
        await asyncio.sleep(0.03)
        cancel_event.set()
        cancelled = await cancel_task
        assert cancelled.status == AgentExecutionStatus.cancelled

        failed_runtime = _runtime(
            MockProvider(
                [
                    MockProviderStep(
                        error=ProviderError(
                            code="PROVIDER_UNAVAILABLE",
                            message="down",
                            retryable=False,
                        )
                    )
                ]
            )
        )
        failed = await failed_runtime.run(_request())
        assert failed.status == AgentExecutionStatus.failed
        assert failed.error is not None
        assert failed.error.code.value == "PROVIDER_UNAVAILABLE"

    asyncio.run(scenario())


def test_default_factory_wires_mock_provider() -> None:
    runtime = build_agent_runtime(_settings())
    assert runtime is not None
