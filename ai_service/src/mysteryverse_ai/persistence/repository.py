from __future__ import annotations

import hashlib
from dataclasses import dataclass
from datetime import UTC, datetime, timedelta
from typing import Any

from sqlalchemy import delete, func, select, update
from sqlalchemy.exc import IntegrityError
from sqlalchemy.ext.asyncio import AsyncEngine

from mysteryverse_ai.agent_runtime.persistence_foundation.lifecycle import (
    RunStatus,
    transition_run_status,
)
from mysteryverse_ai.persistence.ids import new_ulid
from mysteryverse_ai.persistence.tables import (
    agent_checkpoints,
    agent_idempotency_records,
    agent_recovery_attempts,
    agent_runs,
    agent_step_attempts,
    agent_steps,
    runtime_audit_records,
    runtime_cleanup_jobs,
)


class IdempotencyConflictError(Exception):
    pass


class FencingRejectedError(Exception):
    pass


class RunStateConflictError(Exception):
    pass


@dataclass(frozen=True, slots=True)
class CreatedRun:
    run_id: str
    reused: bool


@dataclass(frozen=True, slots=True)
class OwnerClaim:
    run_id: str
    owner_id: str
    fencing_token: int
    execution_version: int


@dataclass(frozen=True, slots=True)
class RunSnapshot:
    run_id: str
    tenant_id: str | None
    task_type: str
    status: RunStatus
    execution_version: int
    owner_id: str | None
    fencing_token: int
    owner_lease_expires_at: datetime | None
    runtime_deadline_at: datetime | None
    terminal_at: datetime | None


@dataclass(frozen=True, slots=True)
class StoredCheckpoint:
    checkpoint_id: str
    run_id: str
    checkpoint_seq: int
    schema_version: str
    runtime_version: str
    status: str
    encrypted_blob: bytes
    compression: str
    blob_size_bytes: int
    key_id: str
    nonce: bytes
    state_hash: bytes
    created_at: datetime
    expires_at: datetime | None


def utc_now() -> datetime:
    return datetime.now(tz=UTC).replace(tzinfo=None)


def sha256_bytes(value: bytes) -> bytes:
    return hashlib.sha256(value).digest()


class MySQLRuntimeRepository:
    def __init__(
        self,
        *,
        engine: AsyncEngine,
        clock: Any = utc_now,
    ) -> None:
        self._engine = engine
        self._clock = clock

    async def create_run(
        self,
        *,
        run_id: str,
        tenant_id: str | None,
        task_type: str,
        actor_type: str,
        actor_id: str | None,
        trace_id: str,
        correlation_id: str | None,
        schema_version: str,
        idempotency_scope: str,
        idempotency_key: str,
        request_hash: bytes,
        runtime_deadline_seconds: int,
        idempotency_ttl_seconds: int,
        parent_run_id: str | None = None,
        recovery_source_run_id: str | None = None,
        workflow_key: str | None = None,
        workflow_version: str | None = None,
    ) -> CreatedRun:
        now = self._clock()
        try:
            async with self._engine.begin() as connection:
                existing = (
                    (
                        await connection.execute(
                            select(agent_idempotency_records).where(
                                agent_idempotency_records.c.idempotency_scope == idempotency_scope,
                                agent_idempotency_records.c.idempotency_key == idempotency_key,
                            )
                        )
                    )
                    .mappings()
                    .first()
                )
                if existing is not None:
                    if existing["expires_at"] > now:
                        if bytes(existing["request_hash"]) != request_hash:
                            raise IdempotencyConflictError(
                                "idempotency key was reused with a different request"
                            )
                        return CreatedRun(run_id=str(existing["run_id"]), reused=True)
                    await connection.execute(
                        delete(agent_idempotency_records).where(
                            agent_idempotency_records.c.id == existing["id"]
                        )
                    )

                await connection.execute(
                    agent_runs.insert().values(
                        id=run_id,
                        parent_run_id=parent_run_id,
                        recovery_source_run_id=recovery_source_run_id,
                        tenant_id=tenant_id,
                        task_type=task_type,
                        workflow_key=workflow_key,
                        workflow_version=workflow_version,
                        actor_type=actor_type,
                        actor_id=actor_id,
                        status=RunStatus.QUEUED.value,
                        execution_version=0,
                        current_step_no=0,
                        owner_id=None,
                        fencing_token=0,
                        retry_count=0,
                        runtime_deadline_at=now + timedelta(seconds=runtime_deadline_seconds),
                        trace_id=trace_id,
                        correlation_id=correlation_id,
                        schema_version=schema_version,
                        created_at=now,
                        updated_at=now,
                    )
                )
                await connection.execute(
                    agent_idempotency_records.insert().values(
                        id=new_ulid(),
                        idempotency_scope=idempotency_scope,
                        idempotency_key=idempotency_key,
                        request_hash=request_hash,
                        run_id=run_id,
                        created_at=now,
                        expires_at=now + timedelta(seconds=idempotency_ttl_seconds),
                    )
                )
                return CreatedRun(run_id=run_id, reused=False)
        except IntegrityError as exc:
            existing = await self._find_idempotency(
                scope=idempotency_scope,
                key=idempotency_key,
            )
            if existing is None:
                raise
            if bytes(existing["request_hash"]) != request_hash:
                raise IdempotencyConflictError(
                    "idempotency key was reused with a different request"
                ) from exc
            return CreatedRun(run_id=str(existing["run_id"]), reused=True)

    async def claim_owner(
        self,
        *,
        run_id: str,
        owner_id: str,
        allowed_statuses: frozenset[RunStatus],
        lease_ttl_seconds: int,
    ) -> OwnerClaim:
        now = self._clock()
        async with self._engine.begin() as connection:
            row = (
                (
                    await connection.execute(
                        select(agent_runs).where(agent_runs.c.id == run_id).with_for_update()
                    )
                )
                .mappings()
                .first()
            )
            if row is None:
                raise KeyError(run_id)
            current_status = RunStatus(str(row["status"]))
            if current_status not in allowed_statuses:
                raise RunStateConflictError(f"run status {current_status.value} cannot be claimed")
            current_owner = row["owner_id"]
            current_lease_expiry = row["owner_lease_expires_at"]
            if (
                current_owner is not None
                and current_lease_expiry is not None
                and current_lease_expiry > now
            ):
                raise RunStateConflictError("run already has an unexpired owner lease")
            fencing_token = int(row["fencing_token"]) + 1
            execution_version = int(row["execution_version"]) + 1
            values: dict[str, Any] = {
                "owner_id": owner_id,
                "fencing_token": fencing_token,
                "owner_lease_expires_at": now + timedelta(seconds=lease_ttl_seconds),
                "execution_version": execution_version,
                "updated_at": now,
            }
            if current_status in {RunStatus.QUEUED, RunStatus.RECOVERING}:
                values["status"] = RunStatus.RUNNING.value
                values["started_at"] = row["started_at"] or now
            elif current_status in {RunStatus.FAILED, RunStatus.SUSPENDED}:
                values["status"] = RunStatus.RECOVERING.value
                values["started_at"] = row["started_at"] or now
            await connection.execute(
                update(agent_runs).where(agent_runs.c.id == run_id).values(**values)
            )
            return OwnerClaim(
                run_id=run_id,
                owner_id=owner_id,
                fencing_token=fencing_token,
                execution_version=execution_version,
            )

    async def release_owner(
        self,
        *,
        run_id: str,
        owner_id: str,
        fencing_token: int,
    ) -> None:
        async with self._engine.begin() as connection:
            result = await connection.execute(
                update(agent_runs)
                .where(
                    agent_runs.c.id == run_id,
                    agent_runs.c.owner_id == owner_id,
                    agent_runs.c.fencing_token == fencing_token,
                )
                .values(
                    owner_id=None,
                    owner_lease_expires_at=None,
                    updated_at=self._clock(),
                )
            )
            if result.rowcount != 1:
                raise FencingRejectedError("owner release was rejected by fencing validation")

    async def get_run(self, run_id: str) -> RunSnapshot | None:
        async with self._engine.connect() as connection:
            row = (
                (await connection.execute(select(agent_runs).where(agent_runs.c.id == run_id)))
                .mappings()
                .first()
            )
        return None if row is None else self._snapshot(row)

    async def transition_run(
        self,
        *,
        run_id: str,
        current: RunStatus,
        target: RunStatus,
        owner_id: str,
        fencing_token: int,
        failure_category: str | None = None,
        failure_summary: str | None = None,
    ) -> None:
        transition_run_status(current, target)
        now = self._clock()
        values: dict[str, Any] = {
            "status": target.value,
            "execution_version": agent_runs.c.execution_version + 1,
            "updated_at": now,
            "failure_category": failure_category,
            "failure_summary": failure_summary,
        }
        if target in {RunStatus.COMPLETED, RunStatus.FAILED}:
            values["finished_at"] = now
        if target == RunStatus.CANCELLED:
            values["cancelled_at"] = now
            values["finished_at"] = now
        async with self._engine.begin() as connection:
            result = await connection.execute(
                update(agent_runs)
                .where(
                    agent_runs.c.id == run_id,
                    agent_runs.c.status == current.value,
                    agent_runs.c.owner_id == owner_id,
                    agent_runs.c.fencing_token == fencing_token,
                )
                .values(**values)
            )
            if result.rowcount != 1:
                raise FencingRejectedError(
                    "run state write was rejected by owner/fencing validation"
                )

    async def record_step(
        self,
        *,
        run_id: str,
        step_no: int,
        step_type: str,
        node_key: str | None,
        status: str,
        schema_version: str,
        owner_id: str,
        fencing_token: int,
        input_digest: bytes | None = None,
        output_digest: bytes | None = None,
        execution_metadata: dict[str, Any] | None = None,
        failure_category: str | None = None,
        failure_summary: str | None = None,
    ) -> str:
        now = self._clock()
        step_id = new_ulid()
        attempt_id = new_ulid()
        async with self._engine.begin() as connection:
            await self._verify_owner_locked(
                connection=connection,
                run_id=run_id,
                owner_id=owner_id,
                fencing_token=fencing_token,
            )
            await connection.execute(
                agent_steps.insert().values(
                    id=step_id,
                    agent_run_id=run_id,
                    step_no=step_no,
                    step_type=step_type,
                    node_key=node_key,
                    status=status,
                    input_digest=input_digest,
                    output_digest=output_digest,
                    schema_version=schema_version,
                    attempt_count=1,
                    started_at=now,
                    finished_at=now,
                    failure_category=failure_category,
                    failure_summary=failure_summary,
                    execution_metadata=execution_metadata,
                    created_at=now,
                    updated_at=now,
                )
            )
            await connection.execute(
                agent_step_attempts.insert().values(
                    id=attempt_id,
                    agent_step_id=step_id,
                    attempt_no=1,
                    status=status,
                    input_digest=input_digest,
                    output_digest=output_digest,
                    started_at=now,
                    finished_at=now,
                    failure_category=failure_category,
                    failure_summary=failure_summary,
                    created_at=now,
                )
            )
            await connection.execute(
                update(agent_runs)
                .where(
                    agent_runs.c.id == run_id,
                    agent_runs.c.owner_id == owner_id,
                    agent_runs.c.fencing_token == fencing_token,
                )
                .values(current_step_no=step_no, updated_at=now)
            )
        return step_id

    async def save_checkpoint(
        self,
        *,
        checkpoint_id: str,
        run_id: str,
        checkpoint_seq: int,
        schema_version: str,
        runtime_version: str,
        encrypted_blob: bytes,
        compression: str,
        blob_size_bytes: int,
        key_id: str,
        nonce: bytes,
        state_hash: bytes,
        owner_id: str,
        fencing_token: int,
        expires_at: datetime | None,
        thread_id: str | None = None,
    ) -> None:
        now = self._clock()
        async with self._engine.begin() as connection:
            await self._verify_owner_locked(
                connection=connection,
                run_id=run_id,
                owner_id=owner_id,
                fencing_token=fencing_token,
            )
            await connection.execute(
                agent_checkpoints.insert().values(
                    id=checkpoint_id,
                    agent_run_id=run_id,
                    thread_id=thread_id,
                    checkpoint_seq=checkpoint_seq,
                    schema_version=schema_version,
                    runtime_version=runtime_version,
                    status="valid",
                    storage_type="mysql_blob",
                    storage_ref=None,
                    compression=compression,
                    encrypted_blob=encrypted_blob,
                    blob_size_bytes=blob_size_bytes,
                    key_id=key_id,
                    nonce=nonce,
                    state_hash=state_hash,
                    created_at=now,
                    expires_at=expires_at,
                )
            )

    async def load_checkpoint(
        self,
        *,
        run_id: str,
        checkpoint_id: str | None = None,
    ) -> StoredCheckpoint | None:
        query = select(agent_checkpoints).where(
            agent_checkpoints.c.agent_run_id == run_id,
            agent_checkpoints.c.status == "valid",
        )
        if checkpoint_id is not None:
            query = query.where(agent_checkpoints.c.id == checkpoint_id)
        else:
            query = query.order_by(agent_checkpoints.c.checkpoint_seq.desc()).limit(1)
        async with self._engine.connect() as connection:
            row = (await connection.execute(query)).mappings().first()
        if row is None:
            return None
        return StoredCheckpoint(
            checkpoint_id=str(row["id"]),
            run_id=str(row["agent_run_id"]),
            checkpoint_seq=int(row["checkpoint_seq"]),
            schema_version=str(row["schema_version"]),
            runtime_version=str(row["runtime_version"]),
            status=str(row["status"]),
            encrypted_blob=bytes(row["encrypted_blob"]),
            compression=str(row["compression"]),
            blob_size_bytes=int(row["blob_size_bytes"]),
            key_id=str(row["key_id"]),
            nonce=bytes(row["nonce"]),
            state_hash=bytes(row["state_hash"]),
            created_at=row["created_at"],
            expires_at=row["expires_at"],
        )

    async def create_recovery_attempt(
        self,
        *,
        run_id: str,
        checkpoint_id: str,
        previous_owner_id: str | None,
        new_owner_id: str,
        fencing_token: int,
        trigger_category: str,
    ) -> str:
        now = self._clock()
        recovery_id = new_ulid()
        async with self._engine.begin() as connection:
            row = (
                await connection.execute(
                    select(func.coalesce(func.max(agent_recovery_attempts.c.attempt_no), 0)).where(
                        agent_recovery_attempts.c.agent_run_id == run_id
                    )
                )
            ).scalar_one()
            await connection.execute(
                agent_recovery_attempts.insert().values(
                    id=recovery_id,
                    agent_run_id=run_id,
                    attempt_no=int(row) + 1,
                    trigger_category=trigger_category,
                    recovery_checkpoint_id=checkpoint_id,
                    previous_owner_id=previous_owner_id,
                    new_owner_id=new_owner_id,
                    fencing_token=fencing_token,
                    status="started",
                    started_at=now,
                    created_at=now,
                )
            )
        return recovery_id

    async def finish_recovery_attempt(
        self,
        *,
        recovery_attempt_id: str,
        status: str,
        failure_category: str | None = None,
        failure_summary: str | None = None,
    ) -> None:
        now = self._clock()
        async with self._engine.begin() as connection:
            result = await connection.execute(
                update(agent_recovery_attempts)
                .where(
                    agent_recovery_attempts.c.id == recovery_attempt_id,
                    agent_recovery_attempts.c.status == "started",
                )
                .values(
                    status=status,
                    finished_at=now,
                    failure_category=failure_category,
                    failure_summary=failure_summary,
                )
            )
            if result.rowcount != 1:
                raise RunStateConflictError("recovery attempt is already finalized")

    async def write_audit(
        self,
        *,
        event_category: str,
        result: str,
        expires_at: datetime,
        tenant_id: str | None = None,
        run_id: str | None = None,
        step_id: str | None = None,
        recovery_attempt_id: str | None = None,
        actor_type: str | None = None,
        actor_id: str | None = None,
        failure_category: str | None = None,
        correlation_id: str | None = None,
        detail_summary: str | None = None,
    ) -> str:
        audit_id = new_ulid()
        async with self._engine.begin() as connection:
            await connection.execute(
                runtime_audit_records.insert().values(
                    id=audit_id,
                    agent_run_id=run_id,
                    agent_step_id=step_id,
                    recovery_attempt_id=recovery_attempt_id,
                    tenant_id=tenant_id,
                    event_category=event_category,
                    actor_type=actor_type,
                    actor_id=actor_id,
                    result=result,
                    failure_category=failure_category,
                    correlation_id=correlation_id,
                    detail_summary=detail_summary,
                    created_at=self._clock(),
                    expires_at=expires_at,
                )
            )
        return audit_id

    async def list_terminal_runs(self, *, limit: int) -> list[RunSnapshot]:
        async with self._engine.connect() as connection:
            rows = (
                await connection.execute(
                    select(agent_runs)
                    .where(
                        agent_runs.c.status.in_(
                            [
                                RunStatus.COMPLETED.value,
                                RunStatus.FAILED.value,
                                RunStatus.CANCELLED.value,
                            ]
                        )
                    )
                    .order_by(agent_runs.c.finished_at.asc())
                    .limit(limit)
                )
            ).mappings()
            return [self._snapshot(row) for row in rows]

    async def run_has_active_recovery(self, run_id: str) -> bool:
        async with self._engine.connect() as connection:
            count = (
                await connection.execute(
                    select(func.count())
                    .select_from(agent_recovery_attempts)
                    .where(
                        agent_recovery_attempts.c.agent_run_id == run_id,
                        agent_recovery_attempts.c.status.in_(["started"]),
                    )
                )
            ).scalar_one()
        return int(count) > 0

    async def has_run_references(self, run_id: str) -> bool:
        async with self._engine.connect() as connection:
            count = (
                await connection.execute(
                    select(func.count())
                    .select_from(agent_runs)
                    .where(
                        (agent_runs.c.parent_run_id == run_id)
                        | (agent_runs.c.recovery_source_run_id == run_id)
                    )
                )
            ).scalar_one()
        return int(count) > 0

    async def count_run_artifacts(self, run_id: str) -> dict[str, int]:
        async with self._engine.connect() as connection:
            step_count = (
                await connection.execute(
                    select(func.count())
                    .select_from(agent_steps)
                    .where(agent_steps.c.agent_run_id == run_id)
                )
            ).scalar_one()
            attempt_count = (
                await connection.execute(
                    select(func.count())
                    .select_from(agent_step_attempts)
                    .join(
                        agent_steps,
                        agent_step_attempts.c.agent_step_id == agent_steps.c.id,
                    )
                    .where(agent_steps.c.agent_run_id == run_id)
                )
            ).scalar_one()
            checkpoint_count = (
                await connection.execute(
                    select(func.count())
                    .select_from(agent_checkpoints)
                    .where(agent_checkpoints.c.agent_run_id == run_id)
                )
            ).scalar_one()
            recovery_count = (
                await connection.execute(
                    select(func.count())
                    .select_from(agent_recovery_attempts)
                    .where(agent_recovery_attempts.c.agent_run_id == run_id)
                )
            ).scalar_one()
            audit_count = (
                await connection.execute(
                    select(func.count())
                    .select_from(runtime_audit_records)
                    .where(runtime_audit_records.c.agent_run_id == run_id)
                )
            ).scalar_one()
            idempotency_count = (
                await connection.execute(
                    select(func.count())
                    .select_from(agent_idempotency_records)
                    .where(agent_idempotency_records.c.run_id == run_id)
                )
            ).scalar_one()
        return {
            "steps": int(step_count),
            "step_attempts": int(attempt_count),
            "checkpoints": int(checkpoint_count),
            "recovery_attempts": int(recovery_count),
            "audit_records": int(audit_count),
            "idempotency_records": int(idempotency_count),
        }

    async def count_audit_records(self, run_id: str) -> int:
        async with self._engine.connect() as connection:
            count = (
                await connection.execute(
                    select(func.count())
                    .select_from(runtime_audit_records)
                    .where(runtime_audit_records.c.agent_run_id == run_id)
                )
            ).scalar_one()
        return int(count)

    async def cleanup_run(self, *, run_id: str) -> bool:
        async with self._engine.begin() as connection:
            row = (
                (
                    await connection.execute(
                        select(agent_runs).where(agent_runs.c.id == run_id).with_for_update()
                    )
                )
                .mappings()
                .first()
            )
            if row is None:
                return False
            if RunStatus(str(row["status"])) not in {
                RunStatus.COMPLETED,
                RunStatus.FAILED,
                RunStatus.CANCELLED,
            }:
                return False
            active_recovery = (
                await connection.execute(
                    select(func.count())
                    .select_from(agent_recovery_attempts)
                    .where(
                        agent_recovery_attempts.c.agent_run_id == run_id,
                        agent_recovery_attempts.c.status == "started",
                    )
                )
            ).scalar_one()
            if int(active_recovery) > 0:
                return False
            step_ids = (
                (
                    await connection.execute(
                        select(agent_steps.c.id).where(agent_steps.c.agent_run_id == run_id)
                    )
                )
                .scalars()
                .all()
            )
            if step_ids:
                await connection.execute(
                    delete(agent_step_attempts).where(
                        agent_step_attempts.c.agent_step_id.in_(step_ids)
                    )
                )
            await connection.execute(
                delete(agent_recovery_attempts).where(
                    agent_recovery_attempts.c.agent_run_id == run_id
                )
            )
            await connection.execute(
                delete(agent_checkpoints).where(agent_checkpoints.c.agent_run_id == run_id)
            )
            await connection.execute(
                delete(agent_steps).where(agent_steps.c.agent_run_id == run_id)
            )
            await connection.execute(
                delete(agent_idempotency_records).where(
                    agent_idempotency_records.c.run_id == run_id
                )
            )
            await connection.execute(delete(agent_runs).where(agent_runs.c.id == run_id))
            return True

    async def create_cleanup_job(
        self,
        *,
        resource_type: str,
        resource_id: str,
        job_type: str = "retention_cleanup",
    ) -> str:
        job_id = new_ulid()
        now = self._clock()
        async with self._engine.begin() as connection:
            await connection.execute(
                runtime_cleanup_jobs.insert().values(
                    id=job_id,
                    job_type=job_type,
                    resource_type=resource_type,
                    resource_id=resource_id,
                    status="running",
                    attempt=1,
                    started_at=now,
                    created_at=now,
                )
            )
        return job_id

    async def finish_cleanup_job(
        self,
        *,
        job_id: str,
        status: str,
        failure_category: str | None = None,
        failure_summary: str | None = None,
    ) -> None:
        async with self._engine.begin() as connection:
            await connection.execute(
                update(runtime_cleanup_jobs)
                .where(runtime_cleanup_jobs.c.id == job_id)
                .values(
                    status=status,
                    finished_at=self._clock(),
                    failure_category=failure_category,
                    failure_summary=failure_summary,
                )
            )

    async def purge_expired_audit(self, *, limit: int) -> int:
        now = self._clock()
        async with self._engine.begin() as connection:
            expired_ids = (
                (
                    await connection.execute(
                        select(runtime_audit_records.c.id)
                        .where(runtime_audit_records.c.expires_at <= now)
                        .limit(limit)
                    )
                )
                .scalars()
                .all()
            )
            if not expired_ids:
                return 0
            await connection.execute(
                delete(runtime_audit_records).where(runtime_audit_records.c.id.in_(expired_ids))
            )
            return len(expired_ids)

    async def purge_expired_idempotency(self, *, limit: int) -> int:
        now = self._clock()
        async with self._engine.begin() as connection:
            expired_ids = (
                (
                    await connection.execute(
                        select(agent_idempotency_records.c.id)
                        .where(agent_idempotency_records.c.expires_at <= now)
                        .limit(limit)
                    )
                )
                .scalars()
                .all()
            )
            if not expired_ids:
                return 0
            await connection.execute(
                delete(agent_idempotency_records).where(
                    agent_idempotency_records.c.id.in_(expired_ids)
                )
            )
            return len(expired_ids)

    async def _find_idempotency(
        self,
        *,
        scope: str,
        key: str,
    ) -> Any:
        async with self._engine.connect() as connection:
            return (
                (
                    await connection.execute(
                        select(agent_idempotency_records).where(
                            agent_idempotency_records.c.idempotency_scope == scope,
                            agent_idempotency_records.c.idempotency_key == key,
                        )
                    )
                )
                .mappings()
                .first()
            )

    async def _verify_owner_locked(
        self,
        *,
        connection: Any,
        run_id: str,
        owner_id: str,
        fencing_token: int,
    ) -> None:
        row = (
            (
                await connection.execute(
                    select(agent_runs.c.owner_id, agent_runs.c.fencing_token)
                    .where(agent_runs.c.id == run_id)
                    .with_for_update()
                )
            )
            .mappings()
            .first()
        )
        if (
            row is None
            or str(row["owner_id"]) != owner_id
            or int(row["fencing_token"]) != fencing_token
        ):
            raise FencingRejectedError("write was rejected by owner/fencing validation")

    def _snapshot(self, row: Any) -> RunSnapshot:
        terminal_at = row["cancelled_at"] or row["finished_at"]
        return RunSnapshot(
            run_id=str(row["id"]),
            tenant_id=row["tenant_id"],
            task_type=str(row["task_type"]),
            status=RunStatus(str(row["status"])),
            execution_version=int(row["execution_version"]),
            owner_id=row["owner_id"],
            fencing_token=int(row["fencing_token"]),
            owner_lease_expires_at=row["owner_lease_expires_at"],
            runtime_deadline_at=row["runtime_deadline_at"],
            terminal_at=terminal_at,
        )
