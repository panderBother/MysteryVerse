from __future__ import annotations

import asyncio
from datetime import UTC, datetime, timedelta

import pytest
from pydantic import BaseModel, ConfigDict, Field, ValidationError

from mysteryverse_ai.agent_runtime.outputs import (
    OutputSchemaRegistry,
    OutputSchemaSelectionError,
    validation_issues,
)
from mysteryverse_ai.agent_runtime.run_store import (
    CheckpointReference,
    InMemoryAgentRunStore,
)
from mysteryverse_ai.schemas.generated.models import AgentTaskType


class _NestedOutput(BaseModel):
    model_config = ConfigDict(extra="forbid")

    answer: str
    count: int = Field(ge=1, le=3)


def test_versioned_output_schema_registry() -> None:
    registry = OutputSchemaRegistry()
    registry.register(
        task_type=AgentTaskType.demo,
        key="demo_nested",
        version=1,
        schema=_NestedOutput,
    )
    registry.register(
        task_type=AgentTaskType.demo,
        key="demo_nested",
        version=2,
        schema=_NestedOutput,
    )
    selected = registry.resolve(
        AgentTaskType.demo,
        requested_key="demo_nested",
    )
    assert selected.version == 2
    assert selected.display_name == "demo_nested@2"

    with pytest.raises(OutputSchemaSelectionError) as unsupported:
        registry.resolve(
            AgentTaskType.demo,
            requested_key="demo_nested",
            requested_version=9,
        )
    assert unsupported.value.code == "OUTPUT_SCHEMA_VERSION_UNSUPPORTED"

    with pytest.raises(OutputSchemaSelectionError) as missing:
        registry.resolve(
            AgentTaskType.demo,
            requested_key="unknown",
        )
    assert missing.value.code == "OUTPUT_SCHEMA_NOT_FOUND"

    with pytest.raises(ValidationError) as invalid:
        _NestedOutput.model_validate({"answer": "ok", "count": 0, "unknown": "field"})
    issues = validation_issues(invalid.value)
    assert issues
    assert {issue.path for issue in issues} >= {"count", "unknown"}


def test_in_memory_run_store_lifecycle_and_checkpoint_metadata() -> None:
    current = [datetime(2026, 11, 23, tzinfo=UTC)]
    store = InMemoryAgentRunStore(clock=lambda: current[0])

    async def scenario() -> None:
        record = await store.start(
            run_id="run-1",
            request_id="request-1",
            trace_id="trace-1",
            input_hash="a" * 64,
            output_schema_key="demo_result",
            output_schema_version=1,
            ttl_seconds=60,
        )
        assert record.status == "running"
        assert "context" not in record.model_dump()
        finished = await store.finish(
            "run-1",
            status="succeeded",
            step_count=5,
            input_tokens=10,
            output_tokens=5,
            error_code=None,
        )
        assert finished.status == "succeeded"
        await store.record_checkpoint(
            CheckpointReference(
                run_id="run-1",
                checkpoint_id="checkpoint-1",
                checkpoint_version=1,
                storage_ref="mv_ai://candidate/checkpoint-1",
                content_hash="b" * 64,
                created_at=current[0],
                expires_at=current[0] + timedelta(seconds=60),
            )
        )
        current[0] += timedelta(seconds=61)
        assert await store.cleanup_expired() == 1
        assert await store.get("run-1") is None

    asyncio.run(scenario())
