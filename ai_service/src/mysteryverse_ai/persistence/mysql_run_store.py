from __future__ import annotations

import hashlib
from contextlib import suppress
from datetime import UTC, datetime

from mysteryverse_ai.agent_runtime.persistence_foundation.lease import OwnerLease
from mysteryverse_ai.agent_runtime.persistence_foundation.lifecycle import RunStatus
from mysteryverse_ai.agent_runtime.run_store import (
    AgentRunRecord,
    CheckpointReference,
    RunLifecycleStatus,
)
from mysteryverse_ai.persistence.database import RuntimeDatabase
from mysteryverse_ai.persistence.redis_lease import RedisOwnerLeaseStore
from mysteryverse_ai.persistence.repository import MySQLRuntimeRepository, OwnerClaim
from mysteryverse_ai.schemas.generated.models import AgentTraceStep


class MySQLAgentRunStore:
    def __init__(
        self,
        *,
        database: RuntimeDatabase,
        repository: MySQLRuntimeRepository,
        lease_store: RedisOwnerLeaseStore,
        owner_id: str,
        runtime_deadline_seconds: int,
        idempotency_ttl_seconds: int,
        lease_ttl_seconds: int,
    ) -> None:
        self._database = database
        self._repository = repository
        self._lease_store = lease_store
        self._owner_id = owner_id
        self._runtime_deadline_seconds = runtime_deadline_seconds
        self._idempotency_ttl_seconds = idempotency_ttl_seconds
        self._lease_ttl_seconds = lease_ttl_seconds
        self._claims: dict[str, OwnerClaim] = {}
        self._leases: dict[str, OwnerLease] = {}

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
        created = await self._repository.create_run(
            run_id=run_id,
            tenant_id=None,
            task_type=f"agent:{output_schema_key}",
            actor_type="runtime",
            actor_id=None,
            trace_id=trace_id,
            correlation_id=request_id,
            schema_version=str(output_schema_version),
            idempotency_scope=f"runtime:{request_id}",
            idempotency_key=run_id,
            request_hash=bytes.fromhex(input_hash),
            runtime_deadline_seconds=self._runtime_deadline_seconds,
            idempotency_ttl_seconds=self._idempotency_ttl_seconds,
        )
        if created.reused:
            raise RuntimeError(f"agent run already exists: {run_id}")
        claim = await self._repository.claim_owner(
            run_id=run_id,
            owner_id=self._owner_id,
            allowed_statuses=frozenset({RunStatus.QUEUED}),
            lease_ttl_seconds=self._lease_ttl_seconds,
        )
        try:
            lease = await self._lease_store.acquire(
                run_id=run_id,
                owner_id=self._owner_id,
                fencing_token=claim.fencing_token,
                ttl_seconds=self._lease_ttl_seconds,
            )
        except Exception:
            await self._repository.release_owner(
                run_id=run_id,
                owner_id=self._owner_id,
                fencing_token=claim.fencing_token,
            )
            raise
        self._claims[run_id] = claim
        self._leases[run_id] = lease
        now = datetime.now(tz=UTC)
        return AgentRunRecord(
            run_id=run_id,
            request_id=request_id,
            trace_id=trace_id,
            input_hash=input_hash,
            output_schema_key=output_schema_key,
            output_schema_version=output_schema_version,
            status="running",
            created_at=now,
            updated_at=now,
            expires_at=now,
        )

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
        _ = step_count, input_tokens, output_tokens
        claim = self._claims.get(run_id)
        if claim is None:
            raise KeyError(run_id)
        target = {
            "succeeded": RunStatus.COMPLETED,
            "failed": RunStatus.FAILED,
            "timeout": RunStatus.FAILED,
            "cancelled": RunStatus.CANCELLED,
        }[status]
        await self._repository.transition_run(
            run_id=run_id,
            current=RunStatus.RUNNING,
            target=target,
            owner_id=claim.owner_id,
            fencing_token=claim.fencing_token,
            failure_category=error_code,
            failure_summary="Agent execution failed." if error_code else None,
        )
        lease = self._leases.pop(run_id, None)
        if lease is not None:
            with suppress(Exception):
                await self._lease_store.release(lease)
        await self._repository.release_owner(
            run_id=run_id,
            owner_id=claim.owner_id,
            fencing_token=claim.fencing_token,
        )
        self._claims.pop(run_id, None)
        record = await self.get(run_id)
        if record is None:
            raise KeyError(run_id)
        return record

    async def get(self, run_id: str) -> AgentRunRecord | None:
        snapshot = await self._repository.get_run(run_id)
        if snapshot is None:
            return None
        status: RunLifecycleStatus = "running"
        if snapshot.status == RunStatus.COMPLETED:
            status = "succeeded"
        elif snapshot.status == RunStatus.FAILED:
            status = "failed"
        elif snapshot.status == RunStatus.CANCELLED:
            status = "cancelled"
        now = datetime.now(tz=UTC)
        return AgentRunRecord(
            run_id=snapshot.run_id,
            request_id=snapshot.run_id,
            trace_id=snapshot.run_id,
            input_hash=hashlib.sha256(snapshot.run_id.encode()).hexdigest(),
            output_schema_key="runtime",
            output_schema_version=1,
            status=status,
            created_at=now,
            updated_at=now,
            expires_at=now,
        )

    async def record_checkpoint(self, checkpoint: CheckpointReference) -> None:
        _ = checkpoint
        raise NotImplementedError(
            "use RuntimeCheckpointPersistence for encrypted checkpoint writes"
        )

    async def record_steps(self, run_id: str, steps: list[AgentTraceStep]) -> None:
        claim = self._claims.get(run_id)
        if claim is None:
            raise KeyError(run_id)
        for step in steps:
            status = step.status.value
            if status in {"denied", "timeout"}:
                status = "failed"
            await self._repository.record_step(
                run_id=run_id,
                step_no=step.step_index,
                step_type=step.step_type.value,
                node_key=step.name,
                status=status,
                schema_version="1",
                owner_id=claim.owner_id,
                fencing_token=claim.fencing_token,
                output_digest=(bytes.fromhex(step.summary_hash) if step.summary_hash else None),
                execution_metadata={
                    "model_name": step.model_name,
                    "duration_ms": step.duration_ms,
                    "input_tokens": step.input_tokens,
                    "output_tokens": step.output_tokens,
                },
                failure_category=step.error_code,
            )

    async def cleanup_expired(self) -> int:
        return 0

    async def aclose(self) -> None:
        await self._lease_store.close()
        await self._database.close()
