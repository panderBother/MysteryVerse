from __future__ import annotations

import asyncio
from datetime import UTC, datetime, timedelta

import pytest

from mysteryverse_ai.agent_runtime.persistence_foundation.checkpoint import (
    CheckpointCompatibilityError,
    CheckpointEnvelope,
    CheckpointIntegrityError,
    CheckpointSizeError,
    encode_checkpoint_state,
    validate_checkpoint_blob_size,
    verify_checkpoint,
)
from mysteryverse_ai.agent_runtime.persistence_foundation.idempotency import (
    IdempotencyConflictError,
    InMemoryRunIdempotencyStore,
)
from mysteryverse_ai.agent_runtime.persistence_foundation.lease import (
    InMemoryOwnerLeaseStore,
    LeaseConflictError,
    LeaseExpiredError,
)
from mysteryverse_ai.agent_runtime.persistence_foundation.lifecycle import (
    InvalidRunTransitionError,
    RunStatus,
    transition_run_status,
)
from mysteryverse_ai.agent_runtime.persistence_foundation.recovery import (
    RecoveryAttempt,
    assess_recovery_eligibility,
)
from mysteryverse_ai.agent_runtime.persistence_foundation.retention import (
    RetentionPolicy,
    evaluate_checkpoint_cleanup,
    evaluate_idempotency_cleanup,
    evaluate_run_cleanup,
)
from mysteryverse_ai.agent_runtime.persistence_foundation.retry import (
    RetryPolicy,
    decide_retry,
)


def test_run_lifecycle_allows_recovery_and_blocks_terminal_reversal() -> None:
    assert transition_run_status(RunStatus.CREATED, RunStatus.QUEUED) == RunStatus.QUEUED
    assert transition_run_status(RunStatus.QUEUED, RunStatus.RUNNING) == RunStatus.RUNNING
    assert transition_run_status(RunStatus.RUNNING, RunStatus.RECOVERING) == RunStatus.RECOVERING
    assert transition_run_status(RunStatus.RECOVERING, RunStatus.RUNNING) == RunStatus.RUNNING
    assert transition_run_status(RunStatus.RUNNING, RunStatus.COMPLETED) == RunStatus.COMPLETED
    assert transition_run_status(RunStatus.COMPLETED, RunStatus.COMPLETED) == RunStatus.COMPLETED

    with pytest.raises(InvalidRunTransitionError):
        transition_run_status(RunStatus.COMPLETED, RunStatus.RUNNING)
    assert transition_run_status(RunStatus.FAILED, RunStatus.RECOVERING) == RunStatus.RECOVERING
    with pytest.raises(InvalidRunTransitionError):
        transition_run_status(RunStatus.CANCELLED, RunStatus.RECOVERING)
    with pytest.raises(InvalidRunTransitionError):
        transition_run_status(RunStatus.CREATED, RunStatus.RUNNING)


def test_owner_lease_is_atomic_and_stale_fence_is_rejected() -> None:
    current = [datetime(2026, 11, 25, tzinfo=UTC)]
    store = InMemoryOwnerLeaseStore(clock=lambda: current[0])

    async def scenario() -> None:
        acquired = await asyncio.gather(
            *[
                store.acquire(run_id="run-1", owner_id=f"pod-{index}", ttl_seconds=10)
                for index in range(20)
            ],
            return_exceptions=True,
        )
        successes = [item for item in acquired if not isinstance(item, Exception)]
        conflicts = [item for item in acquired if isinstance(item, LeaseConflictError)]
        assert len(successes) == 1
        assert len(conflicts) == 19
        lease = successes[0]
        assert lease.fencing_token == 1
        assert await store.validate_fencing_token(
            run_id="run-1",
            owner_id=lease.owner_id,
            fencing_token=lease.fencing_token,
        )

        current[0] += timedelta(seconds=11)
        with pytest.raises(LeaseExpiredError):
            await store.renew(lease, ttl_seconds=10)

        replacement = await store.acquire(
            run_id="run-1",
            owner_id="pod-replacement",
            ttl_seconds=10,
        )
        assert replacement.fencing_token == 2
        assert not await store.validate_fencing_token(
            run_id="run-1",
            owner_id=lease.owner_id,
            fencing_token=lease.fencing_token,
        )
        with pytest.raises(LeaseConflictError):
            await store.release(lease)

    asyncio.run(scenario())


def test_checkpoint_hash_and_schema_compatibility() -> None:
    payload, payload_hash = encode_checkpoint_state(
        {
            "execution_cursor": 3,
            "completed_steps": ["step-1", "step-2"],
        }
    )
    envelope = CheckpointEnvelope(
        checkpoint_id="checkpoint-1",
        run_id="run-1",
        checkpoint_seq=1,
        schema_version="1.0",
        runtime_version="runtime-v1",
        payload_hash=payload_hash,
        created_at=datetime(2026, 11, 25, tzinfo=UTC),
    )

    verify_checkpoint(
        envelope,
        payload,
        supported_schema_versions=frozenset({"1.0"}),
    )
    with pytest.raises(CheckpointIntegrityError):
        verify_checkpoint(
            envelope,
            payload + b" ",
            supported_schema_versions=frozenset({"1.0"}),
        )
    with pytest.raises(CheckpointCompatibilityError):
        verify_checkpoint(
            envelope,
            payload,
            supported_schema_versions=frozenset({"2.0"}),
        )
    with pytest.raises(CheckpointCompatibilityError):
        verify_checkpoint(
            envelope.model_copy(update={"status": "incompatible"}),
            payload,
            supported_schema_versions=frozenset({"1.0"}),
        )


def test_run_idempotency_distinguishes_request_content() -> None:
    store = InMemoryRunIdempotencyStore()

    async def scenario() -> None:
        first = await store.claim(
            scope="tenant-a:demo",
            idempotency_key="request-key",
            request_hash="a" * 64,
            run_id="run-1",
        )
        duplicate = await store.claim(
            scope="tenant-a:demo",
            idempotency_key="request-key",
            request_hash="a" * 64,
            run_id="run-2",
        )
        assert first == "run-1"
        assert duplicate == "run-1"
        with pytest.raises(IdempotencyConflictError):
            await store.claim(
                scope="tenant-a:demo",
                idempotency_key="request-key",
                request_hash="b" * 64,
                run_id="run-3",
            )

    asyncio.run(scenario())


def test_retry_policy_respects_attempts_deadline_and_non_retryable_errors() -> None:
    policy = RetryPolicy(
        max_attempts=3,
        base_delay_ms=100,
        max_delay_ms=250,
        jitter_ratio=0,
    )
    first = decide_retry(
        policy=policy,
        attempt_number=1,
        error_code="AI_TIMEOUT",
    )
    second = decide_retry(
        policy=policy,
        attempt_number=2,
        error_code="AI_TIMEOUT",
    )
    exhausted = decide_retry(
        policy=policy,
        attempt_number=3,
        error_code="AI_TIMEOUT",
    )
    expired = decide_retry(
        policy=policy,
        attempt_number=1,
        error_code="AI_TIMEOUT",
        deadline_exceeded=True,
    )
    non_retryable = decide_retry(
        policy=policy,
        attempt_number=1,
        error_code="VALIDATION_ERROR",
    )

    assert first.should_retry and first.delay_ms == 100
    assert second.should_retry and second.delay_ms == 200
    assert not exhausted.should_retry and exhausted.reason == "attempt_limit"
    assert not expired.should_retry and expired.reason == "deadline_exceeded"
    assert not non_retryable.should_retry and non_retryable.reason == "non_retryable"


def test_retention_never_deletes_active_or_referenced_runs() -> None:
    policy = RetentionPolicy()
    now = datetime(2026, 11, 25, tzinfo=UTC)

    active = evaluate_run_cleanup(
        status=RunStatus.RUNNING,
        terminal_at=None,
        now=now,
        policy=policy,
    )
    completed = evaluate_run_cleanup(
        status=RunStatus.COMPLETED,
        terminal_at=now - timedelta(days=31),
        now=now,
        policy=policy,
    )
    failed_recent = evaluate_run_cleanup(
        status=RunStatus.FAILED,
        terminal_at=now - timedelta(days=89),
        now=now,
        policy=policy,
    )
    failed = evaluate_run_cleanup(
        status=RunStatus.FAILED,
        terminal_at=now - timedelta(days=91),
        now=now,
        policy=policy,
    )
    referenced = evaluate_run_cleanup(
        status=RunStatus.COMPLETED,
        terminal_at=now - timedelta(days=31),
        now=now,
        policy=policy,
        has_active_references=True,
    )
    recovering = evaluate_run_cleanup(
        status=RunStatus.CANCELLED,
        terminal_at=now - timedelta(days=31),
        now=now,
        policy=policy,
        recovery_in_progress=True,
    )

    assert not active.eligible
    assert completed.eligible
    assert not failed_recent.eligible
    assert failed.eligible
    assert not referenced.eligible and referenced.reason == "active_reference"
    assert not recovering.eligible and recovering.reason == "recovery_in_progress"


def test_checkpoint_cleanup_and_size_limit() -> None:
    policy = RetentionPolicy(checkpoint_max_blob_bytes=32)
    now = datetime(2026, 11, 25, tzinfo=UTC)
    active_run = evaluate_checkpoint_cleanup(
        run_status=RunStatus.RUNNING,
        checkpoint_status="valid",
        terminal_at=None,
        now=now,
        policy=policy,
    )
    incompatible = evaluate_checkpoint_cleanup(
        run_status=RunStatus.FAILED,
        checkpoint_status="incompatible",
        terminal_at=now - timedelta(days=91),
        now=now,
        policy=policy,
    )
    expired = evaluate_checkpoint_cleanup(
        run_status=RunStatus.COMPLETED,
        checkpoint_status="valid",
        terminal_at=now - timedelta(days=31),
        now=now,
        policy=policy,
    )

    assert not active_run.eligible
    assert not incompatible.eligible
    assert expired.eligible
    validate_checkpoint_blob_size(b"x" * 32, max_bytes=policy.checkpoint_max_blob_bytes)
    with pytest.raises(CheckpointSizeError):
        validate_checkpoint_blob_size(b"x" * 33, max_bytes=policy.checkpoint_max_blob_bytes)


def test_idempotency_retention_and_recovery_eligibility() -> None:
    policy = RetentionPolicy()
    now = datetime(2026, 11, 25, tzinfo=UTC)
    recent = evaluate_idempotency_cleanup(
        created_at=now - timedelta(days=6),
        now=now,
        policy=policy,
    )
    expired = evaluate_idempotency_cleanup(
        created_at=now - timedelta(days=8),
        now=now,
        policy=policy,
    )
    assert not recent.eligible
    assert expired.eligible

    eligible = assess_recovery_eligibility(
        run_status=RunStatus.FAILED,
        checkpoint_available=True,
        checkpoint_valid=True,
        schema_compatible=True,
        owner_lease_available=True,
    )
    incompatible = assess_recovery_eligibility(
        run_status=RunStatus.FAILED,
        checkpoint_available=True,
        checkpoint_valid=True,
        schema_compatible=False,
        owner_lease_available=True,
    )
    assert eligible.eligible
    assert not incompatible.eligible
    assert incompatible.reason == "schema_incompatible"

    attempt = RecoveryAttempt(
        recovery_attempt_id="recovery-1",
        run_id="run-1",
        attempt_number=1,
        trigger_category="owner_failure",
        recovery_checkpoint_id="checkpoint-1",
        previous_owner_id="pod-old",
        new_owner_id="pod-new",
        fencing_token=2,
        status="started",
        started_at=now,
    )
    assert attempt.run_id == "run-1"
    assert attempt.finished_at is None
