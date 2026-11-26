from __future__ import annotations

from datetime import datetime
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field

from mysteryverse_ai.agent_runtime.persistence_foundation.lifecycle import RunStatus

RecoveryAttemptStatus = Literal["started", "succeeded", "failed", "cancelled"]


class RecoveryAttempt(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    recovery_attempt_id: str = Field(min_length=1)
    run_id: str = Field(min_length=1)
    attempt_number: int = Field(ge=1)
    trigger_category: str = Field(min_length=1)
    recovery_checkpoint_id: str = Field(min_length=1)
    previous_owner_id: str = Field(min_length=1)
    new_owner_id: str = Field(min_length=1)
    fencing_token: int = Field(ge=1)
    status: RecoveryAttemptStatus
    started_at: datetime
    finished_at: datetime | None = None
    failure_category: str | None = None


class RecoveryEligibility(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    eligible: bool
    reason: str


def assess_recovery_eligibility(
    *,
    run_status: RunStatus,
    checkpoint_available: bool,
    checkpoint_valid: bool,
    schema_compatible: bool,
    owner_lease_available: bool,
) -> RecoveryEligibility:
    if run_status not in {RunStatus.FAILED, RunStatus.SUSPENDED, RunStatus.RECOVERING}:
        return RecoveryEligibility(eligible=False, reason="run_status_not_recoverable")
    if not checkpoint_available:
        return RecoveryEligibility(eligible=False, reason="checkpoint_unavailable")
    if not checkpoint_valid:
        return RecoveryEligibility(eligible=False, reason="checkpoint_invalid")
    if not schema_compatible:
        return RecoveryEligibility(eligible=False, reason="schema_incompatible")
    if not owner_lease_available:
        return RecoveryEligibility(eligible=False, reason="owner_lease_unavailable")
    return RecoveryEligibility(eligible=True, reason="recoverable")
