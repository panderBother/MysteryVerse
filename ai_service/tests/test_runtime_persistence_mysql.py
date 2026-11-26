from __future__ import annotations

import asyncio
import os
import secrets
from datetime import timedelta

import pytest
from redis.asyncio import Redis
from sqlalchemy.ext.asyncio import create_async_engine

from mysteryverse_ai.agent_runtime.persistence_foundation.lifecycle import RunStatus
from mysteryverse_ai.agent_runtime.persistence_foundation.retention import RetentionPolicy
from mysteryverse_ai.persistence.checkpoint_cipher import CheckpointCipher
from mysteryverse_ai.persistence.checkpoint_persistence import RuntimeCheckpointPersistence
from mysteryverse_ai.persistence.cleanup import RuntimeCleanupWorker
from mysteryverse_ai.persistence.ids import new_ulid
from mysteryverse_ai.persistence.recovery_service import RuntimeRecoveryService
from mysteryverse_ai.persistence.redis_lease import RedisOwnerLeaseStore
from mysteryverse_ai.persistence.repository import (
    FencingRejectedError,
    IdempotencyConflictError,
    MySQLRuntimeRepository,
    RunStateConflictError,
)


def _database_url() -> str:
    value = os.getenv("AI_RUNTIME_TEST_DATABASE_URL", "")
    if value.startswith("mysql://"):
        return "mysql+asyncmy://" + value.removeprefix("mysql://")
    return value


@pytest.mark.integration
def test_mysql_fencing_idempotency_checkpoint_and_multi_owner_conflict() -> None:
    database_url = _database_url()
    redis_url = os.getenv("AI_RUNTIME_TEST_REDIS_URL", "")
    if not database_url or not redis_url:
        pytest.skip("AI_RUNTIME_TEST_DATABASE_URL and AI_RUNTIME_TEST_REDIS_URL are required")

    async def scenario() -> None:
        engine = create_async_engine(database_url)
        redis = Redis.from_url(redis_url)
        lease_store = RedisOwnerLeaseStore(
            client=redis,
            key_prefix=f"mv:test:lease:{new_ulid()}",
        )
        repository = MySQLRuntimeRepository(engine=engine)
        run_id = new_ulid()
        request_hash = b"a" * 32
        try:
            created = await repository.create_run(
                run_id=run_id,
                tenant_id="tenant-test",
                task_type="demo",
                actor_type="system",
                actor_id=None,
                trace_id="trace-test",
                correlation_id="correlation-test",
                schema_version="1",
                idempotency_scope=f"test:{run_id}",
                idempotency_key="idempotency-key",
                request_hash=request_hash,
                runtime_deadline_seconds=60,
                idempotency_ttl_seconds=60,
            )
            assert not created.reused
            duplicate = await repository.create_run(
                run_id=new_ulid(),
                tenant_id="tenant-test",
                task_type="demo",
                actor_type="system",
                actor_id=None,
                trace_id="trace-test",
                correlation_id="correlation-test",
                schema_version="1",
                idempotency_scope=f"test:{run_id}",
                idempotency_key="idempotency-key",
                request_hash=request_hash,
                runtime_deadline_seconds=60,
                idempotency_ttl_seconds=60,
            )
            assert duplicate.run_id == run_id and duplicate.reused
            with pytest.raises(IdempotencyConflictError):
                await repository.create_run(
                    run_id=new_ulid(),
                    tenant_id="tenant-test",
                    task_type="demo",
                    actor_type="system",
                    actor_id=None,
                    trace_id="trace-test",
                    correlation_id="correlation-test",
                    schema_version="1",
                    idempotency_scope=f"test:{run_id}",
                    idempotency_key="idempotency-key",
                    request_hash=b"b" * 32,
                    runtime_deadline_seconds=60,
                    idempotency_ttl_seconds=60,
                )

            claim = await repository.claim_owner(
                run_id=run_id,
                owner_id="pod-a",
                allowed_statuses=frozenset({RunStatus.QUEUED}),
                lease_ttl_seconds=60,
            )
            with pytest.raises(RunStateConflictError):
                await repository.claim_owner(
                    run_id=run_id,
                    owner_id="pod-b",
                    allowed_statuses=frozenset({RunStatus.RUNNING}),
                    lease_ttl_seconds=60,
                )
            lease = await lease_store.acquire(
                run_id=run_id,
                owner_id="pod-a",
                fencing_token=claim.fencing_token,
                ttl_seconds=60,
            )
            assert lease.fencing_token == 1

            cipher = CheckpointCipher(
                key=b"c" * 32,
                key_id="test-checkpoint-v1",
                max_blob_bytes=1024,
            )
            checkpoint_store = RuntimeCheckpointPersistence(
                repository=repository,
                cipher=cipher,
            )
            checkpoint_id = await checkpoint_store.save_state(
                run_id=run_id,
                checkpoint_seq=1,
                schema_version="1.0",
                runtime_version="test-runtime",
                state={"cursor": 1},
                claim=claim,
            )
            restored = await checkpoint_store.load_state(
                run_id=run_id,
                checkpoint_id=checkpoint_id,
                supported_schema_versions=frozenset({"1.0"}),
            )
            assert restored == {"cursor": 1}

            with pytest.raises(FencingRejectedError):
                await repository.record_step(
                    run_id=run_id,
                    step_no=1,
                    step_type="model",
                    node_key="stale",
                    status="succeeded",
                    schema_version="1",
                    owner_id="pod-a",
                    fencing_token=claim.fencing_token - 1,
                )

            await repository.transition_run(
                run_id=run_id,
                current=RunStatus.RUNNING,
                target=RunStatus.FAILED,
                owner_id="pod-a",
                fencing_token=claim.fencing_token,
                failure_category="TEST_FAILURE",
                failure_summary="Integration test failure.",
            )
            await lease_store.release(lease)
            await repository.release_owner(
                run_id=run_id,
                owner_id="pod-a",
                fencing_token=claim.fencing_token,
            )
            await repository.cleanup_run(run_id=run_id)
        finally:
            await lease_store.close()
            await engine.dispose()

    asyncio.run(scenario())


def test_ulid_generator_returns_pure_26_character_values() -> None:
    value = new_ulid()
    assert len(value) == 26
    assert set(value) <= set("0123456789ABCDEFGHJKMNPQRSTVWXYZ")
    assert len(secrets.token_hex(4)) == 8


@pytest.mark.integration
def test_mysql_recovery_attempt_and_cleanup_integration() -> None:
    database_url = _database_url()
    redis_url = os.getenv("AI_RUNTIME_TEST_REDIS_URL", "")
    if not database_url or not redis_url:
        pytest.skip("AI_RUNTIME_TEST_DATABASE_URL and AI_RUNTIME_TEST_REDIS_URL are required")

    async def scenario() -> None:
        engine = create_async_engine(database_url)
        redis = Redis.from_url(redis_url)
        lease_store = RedisOwnerLeaseStore(
            client=redis,
            key_prefix=f"mv:test:recovery:{new_ulid()}",
        )
        repository = MySQLRuntimeRepository(engine=engine)
        run_id = new_ulid()
        try:
            await repository.create_run(
                run_id=run_id,
                tenant_id="tenant-test",
                task_type="recovery",
                actor_type="system",
                actor_id=None,
                trace_id="trace-recovery",
                correlation_id="correlation-recovery",
                schema_version="1",
                idempotency_scope=f"recovery:{run_id}",
                idempotency_key="recovery",
                request_hash=b"d" * 32,
                runtime_deadline_seconds=120,
                idempotency_ttl_seconds=60,
            )
            claim = await repository.claim_owner(
                run_id=run_id,
                owner_id="recovery-source",
                allowed_statuses=frozenset({RunStatus.QUEUED}),
                lease_ttl_seconds=2,
            )
            lease = await lease_store.acquire(
                run_id=run_id,
                owner_id="recovery-source",
                fencing_token=claim.fencing_token,
                ttl_seconds=2,
            )
            cipher = CheckpointCipher(
                key=b"r" * 32,
                key_id="recovery-checkpoint-v1",
                max_blob_bytes=1024,
            )
            checkpoint_store = RuntimeCheckpointPersistence(
                repository=repository,
                cipher=cipher,
            )
            checkpoint_id = await checkpoint_store.save_state(
                run_id=run_id,
                checkpoint_seq=1,
                schema_version="1.0",
                runtime_version="recovery-runtime",
                state={"cursor": 7},
                claim=claim,
            )
            await repository.transition_run(
                run_id=run_id,
                current=RunStatus.RUNNING,
                target=RunStatus.FAILED,
                owner_id=claim.owner_id,
                fencing_token=claim.fencing_token,
                failure_category="TEST_RECOVERY",
                failure_summary="Recovery integration fixture.",
            )
            await lease_store.release(lease)
            await repository.release_owner(
                run_id=run_id,
                owner_id=claim.owner_id,
                fencing_token=claim.fencing_token,
            )

            recovery = RuntimeRecoveryService(
                repository=repository,
                lease_store=lease_store,
                checkpoint_cipher=cipher,
                owner_id="recovery-target",
                lease_ttl_seconds=10,
                supported_checkpoint_schema_versions=frozenset({"1.0"}),
            )
            recovered = await recovery.recover(
                run_id=run_id,
                checkpoint_id=checkpoint_id,
                trigger_category="integration_test",
            )
            assert recovered.state == b'{"cursor":7}'
            await repository.finish_recovery_attempt(
                recovery_attempt_id=recovered.recovery_attempt_id,
                status="succeeded",
            )
            await repository.transition_run(
                run_id=run_id,
                current=RunStatus.RUNNING,
                target=RunStatus.COMPLETED,
                owner_id=recovered.lease.owner_id,
                fencing_token=recovered.lease.fencing_token,
            )
            await lease_store.release(recovered.lease)
            await repository.release_owner(
                run_id=run_id,
                owner_id=recovered.lease.owner_id,
                fencing_token=recovered.lease.fencing_token,
            )

            cleanup = RuntimeCleanupWorker(
                repository=repository,
                policy=RetentionPolicy(
                    completed_run=timedelta(0),
                    completed_checkpoint=timedelta(0),
                ),
                audit_retention=timedelta(days=180),
            )
            result = await cleanup.run_once()
            assert result.cleaned >= 1
            assert await repository.get_run(run_id) is None
            assert await repository.count_audit_records(run_id) >= 1
        finally:
            await lease_store.close()
            await engine.dispose()

    asyncio.run(scenario())
