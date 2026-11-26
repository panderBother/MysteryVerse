from __future__ import annotations

from datetime import datetime, timedelta
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field

from mysteryverse_ai.agent_runtime.persistence_foundation.lifecycle import (
    TERMINAL_RUN_STATUSES,
    RunStatus,
)

RetentionResource = Literal["run", "checkpoint", "audit", "idempotency"]


class RetentionPolicy(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    active_run_max_runtime: timedelta = timedelta(hours=24)
    completed_run: timedelta = timedelta(days=30)
    failed_run: timedelta = timedelta(days=90)
    cancelled_run: timedelta = timedelta(days=30)
    completed_checkpoint: timedelta = timedelta(days=30)
    failed_checkpoint: timedelta = timedelta(days=90)
    cancelled_checkpoint: timedelta = timedelta(days=30)
    runtime_audit: timedelta = timedelta(days=180)
    idempotency: timedelta = timedelta(days=7)
    checkpoint_max_blob_bytes: int = Field(default=1 << 20, ge=1)
    cleanup_batch_size: int = Field(default=100, ge=1, le=10_000)


class RetentionDecision(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    eligible: bool
    reason: str
    deadline: datetime | None = None


class CleanupEligibility(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    eligible: bool
    reason: str
    retention_deadline: datetime | None = None


def _terminal_deadline(
    *,
    status: RunStatus,
    terminal_at: datetime,
    policy: RetentionPolicy,
    checkpoint: bool = False,
) -> datetime | None:
    if status == RunStatus.COMPLETED:
        duration = policy.completed_checkpoint if checkpoint else policy.completed_run
    elif status == RunStatus.FAILED:
        duration = policy.failed_checkpoint if checkpoint else policy.failed_run
    elif status == RunStatus.CANCELLED:
        duration = policy.cancelled_checkpoint if checkpoint else policy.cancelled_run
    else:
        return None
    return terminal_at + duration


def evaluate_run_cleanup(
    *,
    status: RunStatus,
    terminal_at: datetime | None,
    now: datetime,
    policy: RetentionPolicy,
    has_active_references: bool = False,
    recovery_in_progress: bool = False,
) -> CleanupEligibility:
    if status not in TERMINAL_RUN_STATUSES:
        return CleanupEligibility(
            eligible=False,
            reason="active_run_not_cleanup_eligible",
        )
    if terminal_at is None:
        return CleanupEligibility(
            eligible=False,
            reason="terminal_timestamp_missing",
        )
    deadline = _terminal_deadline(status=status, terminal_at=terminal_at, policy=policy)
    if deadline is None or now < deadline:
        return CleanupEligibility(
            eligible=False,
            reason="retention_period_active",
            retention_deadline=deadline,
        )
    if has_active_references:
        return CleanupEligibility(
            eligible=False,
            reason="active_reference",
            retention_deadline=deadline,
        )
    if recovery_in_progress:
        return CleanupEligibility(
            eligible=False,
            reason="recovery_in_progress",
            retention_deadline=deadline,
        )
    return CleanupEligibility(
        eligible=True,
        reason="retention_expired",
        retention_deadline=deadline,
    )


def evaluate_checkpoint_cleanup(
    *,
    run_status: RunStatus,
    checkpoint_status: Literal["valid", "superseded", "incompatible"],
    terminal_at: datetime | None,
    now: datetime,
    policy: RetentionPolicy,
    recovery_in_progress: bool = False,
    referenced: bool = False,
) -> CleanupEligibility:
    if run_status not in TERMINAL_RUN_STATUSES:
        return CleanupEligibility(
            eligible=False,
            reason="run_not_terminal",
        )
    if recovery_in_progress or referenced:
        return CleanupEligibility(
            eligible=False,
            reason="checkpoint_referenced_or_recovering",
        )
    if terminal_at is None:
        return CleanupEligibility(
            eligible=False,
            reason="terminal_timestamp_missing",
        )
    deadline = _terminal_deadline(
        status=run_status,
        terminal_at=terminal_at,
        policy=policy,
        checkpoint=True,
    )
    if deadline is None or now < deadline:
        return CleanupEligibility(
            eligible=False,
            reason="retention_period_active",
            retention_deadline=deadline,
        )
    if checkpoint_status == "incompatible":
        return CleanupEligibility(
            eligible=False,
            reason="incompatible_checkpoint_requires_inspection",
            retention_deadline=deadline,
        )
    return CleanupEligibility(
        eligible=True,
        reason="retention_expired",
        retention_deadline=deadline,
    )


def evaluate_idempotency_cleanup(
    *,
    created_at: datetime,
    now: datetime,
    policy: RetentionPolicy,
) -> RetentionDecision:
    deadline = created_at + policy.idempotency
    return RetentionDecision(
        eligible=now >= deadline,
        reason="retention_expired" if now >= deadline else "retention_period_active",
        deadline=deadline,
    )
