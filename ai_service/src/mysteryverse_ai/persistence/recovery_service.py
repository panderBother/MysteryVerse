from __future__ import annotations

from dataclasses import dataclass

from mysteryverse_ai.agent_runtime.persistence_foundation.lease import OwnerLease
from mysteryverse_ai.agent_runtime.persistence_foundation.lifecycle import (
    TERMINAL_RUN_STATUSES,
    RunStatus,
)
from mysteryverse_ai.persistence.checkpoint_cipher import CheckpointCipher
from mysteryverse_ai.persistence.redis_lease import RedisOwnerLeaseStore
from mysteryverse_ai.persistence.repository import MySQLRuntimeRepository


class RecoveryRejectedError(Exception):
    def __init__(self, *, reason: str) -> None:
        super().__init__(reason)
        self.reason = reason


@dataclass(frozen=True, slots=True)
class RecoveryResult:
    run_id: str
    recovery_attempt_id: str
    checkpoint_id: str
    state: bytes
    lease: OwnerLease


class RuntimeRecoveryService:
    def __init__(
        self,
        *,
        repository: MySQLRuntimeRepository,
        lease_store: RedisOwnerLeaseStore,
        checkpoint_cipher: CheckpointCipher,
        owner_id: str,
        lease_ttl_seconds: int,
        supported_checkpoint_schema_versions: frozenset[str],
    ) -> None:
        self._repository = repository
        self._lease_store = lease_store
        self._checkpoint_cipher = checkpoint_cipher
        self._owner_id = owner_id
        self._lease_ttl_seconds = lease_ttl_seconds
        self._supported_schema_versions = supported_checkpoint_schema_versions

    async def recover(
        self,
        *,
        run_id: str,
        checkpoint_id: str | None = None,
        trigger_category: str,
    ) -> RecoveryResult:
        run = await self._repository.get_run(run_id)
        if run is None:
            raise RecoveryRejectedError(reason="run_not_found")
        if run.status in TERMINAL_RUN_STATUSES and run.status != RunStatus.FAILED:
            raise RecoveryRejectedError(reason="run_status_not_recoverable")
        if run.status not in {
            RunStatus.FAILED,
            RunStatus.SUSPENDED,
            RunStatus.RECOVERING,
        }:
            raise RecoveryRejectedError(reason="run_status_not_recoverable")

        checkpoint = await self._repository.load_checkpoint(
            run_id=run_id,
            checkpoint_id=checkpoint_id,
        )
        if checkpoint is None:
            raise RecoveryRejectedError(reason="checkpoint_unavailable")
        try:
            state = self._checkpoint_cipher.decrypt(
                checkpoint=checkpoint,
                run_id=run_id,
                supported_schema_versions=self._supported_schema_versions,
            )
        except Exception as exc:
            raise RecoveryRejectedError(reason="checkpoint_invalid_or_incompatible") from exc

        claim = await self._repository.claim_owner(
            run_id=run_id,
            owner_id=self._owner_id,
            allowed_statuses=frozenset(
                {
                    RunStatus.FAILED,
                    RunStatus.SUSPENDED,
                    RunStatus.RECOVERING,
                }
            ),
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

        try:
            recovery_attempt_id = await self._repository.create_recovery_attempt(
                run_id=run_id,
                checkpoint_id=checkpoint.checkpoint_id,
                previous_owner_id=run.owner_id,
                new_owner_id=self._owner_id,
                fencing_token=claim.fencing_token,
                trigger_category=trigger_category,
            )
            await self._repository.transition_run(
                run_id=run_id,
                current=RunStatus.RECOVERING,
                target=RunStatus.RUNNING,
                owner_id=self._owner_id,
                fencing_token=claim.fencing_token,
            )
        except Exception as exc:
            await self._repository.transition_run(
                run_id=run_id,
                current=RunStatus.RECOVERING,
                target=RunStatus.FAILED,
                owner_id=self._owner_id,
                fencing_token=claim.fencing_token,
                failure_category="RECOVERY_FAILED",
                failure_summary="Runtime recovery setup failed.",
            )
            raise RecoveryRejectedError(reason="recovery_setup_failed") from exc

        return RecoveryResult(
            run_id=run_id,
            recovery_attempt_id=recovery_attempt_id,
            checkpoint_id=checkpoint.checkpoint_id,
            state=state,
            lease=lease,
        )
