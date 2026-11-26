from __future__ import annotations

import asyncio
from collections.abc import Callable
from datetime import UTC, datetime, timedelta
from typing import Literal, Protocol

from pydantic import BaseModel, ConfigDict, Field

from mysteryverse_ai.schemas.generated.models import AgentTraceStep

RunLifecycleStatus = Literal[
    "running",
    "succeeded",
    "failed",
    "timeout",
    "cancelled",
]


class AgentRunRecord(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    run_id: str
    request_id: str
    trace_id: str
    input_hash: str
    output_schema_key: str
    output_schema_version: int
    status: RunLifecycleStatus
    step_count: int = Field(default=0, ge=0)
    input_tokens: int = Field(default=0, ge=0)
    output_tokens: int = Field(default=0, ge=0)
    error_code: str | None = None
    created_at: datetime
    updated_at: datetime
    expires_at: datetime


class CheckpointReference(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    run_id: str
    checkpoint_id: str
    checkpoint_version: int = Field(ge=1)
    storage_ref: str
    content_hash: str = Field(pattern="^[a-f0-9]{64}$")
    created_at: datetime
    expires_at: datetime


class AgentRunStore(Protocol):
    async def start(
        self,
        *,
        run_id: str,
        request_id: str,
        trace_id: str,
        input_hash: str,
        output_schema_key: str,
        output_schema_version: int,
        ttl_seconds: int,
    ) -> AgentRunRecord: ...

    async def finish(
        self,
        run_id: str,
        *,
        status: RunLifecycleStatus,
        step_count: int,
        input_tokens: int,
        output_tokens: int,
        error_code: str | None,
    ) -> AgentRunRecord: ...

    async def get(self, run_id: str) -> AgentRunRecord | None: ...

    async def record_checkpoint(self, checkpoint: CheckpointReference) -> None: ...

    async def record_steps(self, run_id: str, steps: list[AgentTraceStep]) -> None: ...

    async def cleanup_expired(self) -> int: ...


class InMemoryAgentRunStore:
    def __init__(
        self,
        *,
        max_records: int = 1_000,
        clock: Callable[[], datetime] | None = None,
    ) -> None:
        self._runs: dict[str, AgentRunRecord] = {}
        self._checkpoints: dict[str, list[CheckpointReference]] = {}
        self._steps: dict[str, list[AgentTraceStep]] = {}
        self._max_records = max_records
        self._clock = clock or (lambda: datetime.now(tz=UTC))
        self._lock = asyncio.Lock()

    async def start(
        self,
        *,
        run_id: str,
        request_id: str,
        trace_id: str,
        input_hash: str,
        output_schema_key: str,
        output_schema_version: int,
        ttl_seconds: int,
    ) -> AgentRunRecord:
        now = self._clock()
        record = AgentRunRecord(
            run_id=run_id,
            request_id=request_id,
            trace_id=trace_id,
            input_hash=input_hash,
            output_schema_key=output_schema_key,
            output_schema_version=output_schema_version,
            status="running",
            created_at=now,
            updated_at=now,
            expires_at=now + timedelta(seconds=ttl_seconds),
        )
        async with self._lock:
            await self._cleanup_locked()
            if run_id in self._runs:
                raise RuntimeError(f"agent run already exists: {run_id}")
            if len(self._runs) >= self._max_records:
                raise RuntimeError("in-memory agent run store capacity exceeded")
            self._runs[run_id] = record
        return record

    async def finish(
        self,
        run_id: str,
        *,
        status: RunLifecycleStatus,
        step_count: int,
        input_tokens: int,
        output_tokens: int,
        error_code: str | None,
    ) -> AgentRunRecord:
        async with self._lock:
            record = self._runs.get(run_id)
            if record is None:
                raise KeyError(run_id)
            if record.status != "running":
                raise RuntimeError(f"agent run is already finalized: {run_id}")
            updated = record.model_copy(
                update={
                    "status": status,
                    "step_count": step_count,
                    "input_tokens": input_tokens,
                    "output_tokens": output_tokens,
                    "error_code": error_code,
                    "updated_at": self._clock(),
                }
            )
            self._runs[run_id] = updated
        return updated

    async def get(self, run_id: str) -> AgentRunRecord | None:
        async with self._lock:
            return self._runs.get(run_id)

    async def record_checkpoint(self, checkpoint: CheckpointReference) -> None:
        async with self._lock:
            if checkpoint.run_id not in self._runs:
                raise KeyError(checkpoint.run_id)
            self._checkpoints.setdefault(checkpoint.run_id, []).append(checkpoint)

    async def record_steps(self, run_id: str, steps: list[AgentTraceStep]) -> None:
        async with self._lock:
            if run_id not in self._runs:
                raise KeyError(run_id)
            self._steps.setdefault(run_id, []).extend(steps)

    async def cleanup_expired(self) -> int:
        async with self._lock:
            return await self._cleanup_locked()

    async def _cleanup_locked(self) -> int:
        now = self._clock()
        expired = [run_id for run_id, record in self._runs.items() if record.expires_at <= now]
        for run_id in expired:
            self._runs.pop(run_id, None)
            self._checkpoints.pop(run_id, None)
            self._steps.pop(run_id, None)
        return len(expired)
