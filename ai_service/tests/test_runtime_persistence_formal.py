from __future__ import annotations

import asyncio
import hashlib
import secrets
from datetime import UTC, datetime, timedelta

import pytest

from mysteryverse_ai.agent_runtime.persistence_foundation.lease import (
    LeaseConflictError,
    OwnerLease,
)
from mysteryverse_ai.agent_runtime.persistence_foundation.lifecycle import RunStatus
from mysteryverse_ai.agent_runtime.persistence_foundation.retention import RetentionPolicy
from mysteryverse_ai.persistence.checkpoint_cipher import (
    CheckpointCipher,
    CheckpointDecryptionError,
)
from mysteryverse_ai.persistence.cleanup import RuntimeCleanupWorker
from mysteryverse_ai.persistence.recovery_service import (
    RecoveryRejectedError,
    RuntimeRecoveryService,
)
from mysteryverse_ai.persistence.repository import (
    OwnerClaim,
    RunSnapshot,
    StoredCheckpoint,
    utc_now,
)


def _encrypted_checkpoint() -> tuple[CheckpointCipher, StoredCheckpoint]:
    cipher = CheckpointCipher(
        key=b"k" * 32,
        key_id="checkpoint-v1",
        max_blob_bytes=1024,
    )
    created_at = datetime(2026, 11, 25, tzinfo=UTC).replace(tzinfo=None)
    encrypted = cipher.encrypt(
        state={"cursor": 2, "pending": ["step-3"]},
        checkpoint_id="checkpoint-1",
        run_id="run-1",
        checkpoint_seq=2,
        schema_version="1.0",
        runtime_version="runtime-v1",
        created_at=created_at,
        nonce=secrets.token_bytes(12),
    )
    stored = StoredCheckpoint(
        checkpoint_id=encrypted.checkpoint_id,
        run_id="run-1",
        checkpoint_seq=encrypted.checkpoint_seq,
        schema_version=encrypted.schema_version,
        runtime_version=encrypted.runtime_version,
        status="valid",
        encrypted_blob=encrypted.encrypted_blob,
        compression=encrypted.compression,
        blob_size_bytes=encrypted.blob_size_bytes,
        key_id=encrypted.key_id,
        nonce=encrypted.nonce,
        state_hash=encrypted.state_hash,
        created_at=created_at,
        expires_at=None,
    )
    return cipher, stored


def test_checkpoint_cipher_encrypts_compresses_and_detects_tampering() -> None:
    cipher, checkpoint = _encrypted_checkpoint()
    payload = cipher.decrypt(
        checkpoint=checkpoint,
        run_id="run-1",
        supported_schema_versions=frozenset({"1.0"}),
    )
    assert b'"cursor":2' in payload

    tampered = StoredCheckpoint(
        checkpoint_id=checkpoint.checkpoint_id,
        run_id=checkpoint.run_id,
        checkpoint_seq=checkpoint.checkpoint_seq,
        schema_version=checkpoint.schema_version,
        runtime_version=checkpoint.runtime_version,
        status=checkpoint.status,
        encrypted_blob=checkpoint.encrypted_blob[:-1] + b"x",
        compression=checkpoint.compression,
        blob_size_bytes=checkpoint.blob_size_bytes,
        key_id=checkpoint.key_id,
        nonce=checkpoint.nonce,
        state_hash=checkpoint.state_hash,
        created_at=checkpoint.created_at,
        expires_at=checkpoint.expires_at,
    )
    with pytest.raises(CheckpointDecryptionError):
        cipher.decrypt(
            checkpoint=tampered,
            run_id="run-1",
            supported_schema_versions=frozenset({"1.0"}),
        )
    with pytest.raises(CheckpointDecryptionError):
        cipher.decrypt(
            checkpoint=checkpoint,
            run_id="run-2",
            supported_schema_versions=frozenset({"1.0"}),
        )


class _FakeCleanupRepository:
    def __init__(self, runs: list[RunSnapshot], *, referenced: bool = False) -> None:
        self.runs = runs
        self.referenced = referenced
        self.cleaned: list[str] = []
        self.jobs: list[tuple[str, str]] = []

    async def list_terminal_runs(self, *, limit: int) -> list[RunSnapshot]:
        return self.runs[:limit]

    async def has_run_references(self, run_id: str) -> bool:
        _ = run_id
        return self.referenced

    async def run_has_active_recovery(self, run_id: str) -> bool:
        _ = run_id
        return False

    async def create_cleanup_job(self, *, resource_type: str, resource_id: str) -> str:
        _ = resource_type
        return f"job-{resource_id}"

    async def write_audit(self, **_: object) -> str:
        return "audit-1"

    async def cleanup_run(self, *, run_id: str) -> bool:
        self.cleaned.append(run_id)
        return True

    async def finish_cleanup_job(
        self,
        *,
        job_id: str,
        status: str,
        failure_category: str | None = None,
        failure_summary: str | None = None,
    ) -> None:
        _ = failure_category, failure_summary
        self.jobs.append((job_id, status))

    async def purge_expired_idempotency(self, *, limit: int) -> int:
        _ = limit
        return 2

    async def purge_expired_audit(self, *, limit: int) -> int:
        _ = limit
        return 3


def test_cleanup_worker_respects_references_and_reports_metrics() -> None:
    now = utc_now()
    run = RunSnapshot(
        run_id="run-1",
        tenant_id="tenant-a",
        task_type="demo",
        status=RunStatus.COMPLETED,
        execution_version=4,
        owner_id=None,
        fencing_token=2,
        owner_lease_expires_at=None,
        runtime_deadline_at=None,
        terminal_at=now - timedelta(days=31),
    )
    referenced_repository = _FakeCleanupRepository([run], referenced=True)
    worker = RuntimeCleanupWorker(
        repository=referenced_repository,  # type: ignore[arg-type]
        policy=RetentionPolicy(),
        audit_retention=timedelta(days=180),
    )
    result = asyncio.run(worker.run_once())
    assert result.scanned == 1
    assert result.skipped == 1
    assert result.cleaned == 0
    assert referenced_repository.cleaned == []

    eligible_repository = _FakeCleanupRepository([run])
    worker = RuntimeCleanupWorker(
        repository=eligible_repository,  # type: ignore[arg-type]
        policy=RetentionPolicy(),
        audit_retention=timedelta(days=180),
    )
    result = asyncio.run(worker.run_once())
    assert result.cleaned == 1
    assert result.idempotency_records_purged == 2
    assert result.audit_records_purged == 3
    assert eligible_repository.cleaned == ["run-1"]
    assert eligible_repository.jobs == [("job-run-1", "succeeded")]


def test_stored_checkpoint_hash_is_stable_for_retention_audit() -> None:
    _, checkpoint = _encrypted_checkpoint()
    assert hashlib.sha256(checkpoint.encrypted_blob).digest() != checkpoint.state_hash
    assert len(checkpoint.state_hash) == 32


class _FakeRecoveryRepository:
    def __init__(self, run: RunSnapshot, checkpoint: StoredCheckpoint) -> None:
        self.run = run
        self.checkpoint = checkpoint
        self.owner_id: str | None = None
        self.fencing_token = run.fencing_token
        self.released = False
        self.recovery_attempt: str | None = None

    async def get_run(self, run_id: str) -> RunSnapshot | None:
        return self.run if run_id == self.run.run_id else None

    async def load_checkpoint(
        self,
        *,
        run_id: str,
        checkpoint_id: str | None,
    ) -> StoredCheckpoint | None:
        _ = run_id
        if checkpoint_id is None or checkpoint_id == self.checkpoint.checkpoint_id:
            return self.checkpoint
        return None

    async def claim_owner(
        self,
        *,
        run_id: str,
        owner_id: str,
        allowed_statuses: frozenset[RunStatus],
        lease_ttl_seconds: int,
    ) -> OwnerClaim:
        _ = run_id, lease_ttl_seconds
        assert self.run.status in allowed_statuses
        self.owner_id = owner_id
        self.fencing_token += 1
        return OwnerClaim(
            run_id=self.run.run_id,
            owner_id=owner_id,
            fencing_token=self.fencing_token,
            execution_version=self.run.execution_version + 1,
        )

    async def release_owner(
        self,
        *,
        run_id: str,
        owner_id: str,
        fencing_token: int,
    ) -> None:
        _ = run_id, owner_id, fencing_token
        self.owner_id = None
        self.released = True

    async def create_recovery_attempt(self, **_: object) -> str:
        self.recovery_attempt = "recovery-1"
        return self.recovery_attempt

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
        _ = (
            run_id,
            current,
            owner_id,
            fencing_token,
            failure_category,
            failure_summary,
        )
        assert target == RunStatus.RUNNING


class _FakeLeaseStore:
    def __init__(self, *, conflict: bool = False) -> None:
        self.conflict = conflict

    async def acquire(
        self,
        *,
        run_id: str,
        owner_id: str,
        fencing_token: int,
        ttl_seconds: int,
    ) -> OwnerLease:
        if self.conflict:
            raise LeaseConflictError("lease conflict")
        return OwnerLease.from_redis(
            run_id=run_id,
            owner_id=owner_id,
            fencing_token=fencing_token,
            ttl_seconds=ttl_seconds,
        )


def test_recovery_service_uses_same_run_and_releases_owner_on_lease_conflict() -> None:
    cipher, checkpoint = _encrypted_checkpoint()
    run = RunSnapshot(
        run_id="run-1",
        tenant_id="tenant-a",
        task_type="demo",
        status=RunStatus.FAILED,
        execution_version=3,
        owner_id="pod-old",
        fencing_token=1,
        owner_lease_expires_at=None,
        runtime_deadline_at=None,
        terminal_at=utc_now(),
    )
    repository = _FakeRecoveryRepository(run, checkpoint)
    service = RuntimeRecoveryService(
        repository=repository,  # type: ignore[arg-type]
        lease_store=_FakeLeaseStore(),  # type: ignore[arg-type]
        checkpoint_cipher=cipher,
        owner_id="pod-new",
        lease_ttl_seconds=30,
        supported_checkpoint_schema_versions=frozenset({"1.0"}),
    )
    result = asyncio.run(
        service.recover(
            run_id="run-1",
            checkpoint_id="checkpoint-1",
            trigger_category="owner_failure",
        )
    )
    assert result.run_id == "run-1"
    assert result.checkpoint_id == "checkpoint-1"
    assert result.recovery_attempt_id == "recovery-1"
    assert result.lease.fencing_token == 2

    conflict_repository = _FakeRecoveryRepository(run, checkpoint)
    conflict_service = RuntimeRecoveryService(
        repository=conflict_repository,  # type: ignore[arg-type]
        lease_store=_FakeLeaseStore(conflict=True),  # type: ignore[arg-type]
        checkpoint_cipher=cipher,
        owner_id="pod-newer",
        lease_ttl_seconds=30,
        supported_checkpoint_schema_versions=frozenset({"1.0"}),
    )
    with pytest.raises(LeaseConflictError):
        asyncio.run(
            conflict_service.recover(
                run_id="run-1",
                checkpoint_id="checkpoint-1",
                trigger_category="owner_failure",
            )
        )
    assert conflict_repository.released


def test_recovery_service_rejects_incompatible_checkpoint_before_owner_claim() -> None:
    cipher, checkpoint = _encrypted_checkpoint()
    run = RunSnapshot(
        run_id="run-1",
        tenant_id=None,
        task_type="demo",
        status=RunStatus.FAILED,
        execution_version=3,
        owner_id=None,
        fencing_token=1,
        owner_lease_expires_at=None,
        runtime_deadline_at=None,
        terminal_at=utc_now(),
    )
    repository = _FakeRecoveryRepository(run, checkpoint)
    service = RuntimeRecoveryService(
        repository=repository,  # type: ignore[arg-type]
        lease_store=_FakeLeaseStore(),  # type: ignore[arg-type]
        checkpoint_cipher=cipher,
        owner_id="pod-new",
        lease_ttl_seconds=30,
        supported_checkpoint_schema_versions=frozenset({"2.0"}),
    )
    with pytest.raises(RecoveryRejectedError) as rejected:
        asyncio.run(
            service.recover(
                run_id="run-1",
                checkpoint_id="checkpoint-1",
                trigger_category="manual",
            )
        )
    assert rejected.value.reason == "checkpoint_invalid_or_incompatible"
    assert repository.owner_id is None
