from __future__ import annotations

import logging
from dataclasses import dataclass
from datetime import timedelta

from mysteryverse_ai.agent_runtime.persistence_foundation.retention import (
    RetentionPolicy,
    evaluate_run_cleanup,
)
from mysteryverse_ai.config import Settings
from mysteryverse_ai.observability.redaction import log_structured
from mysteryverse_ai.persistence.repository import MySQLRuntimeRepository, utc_now


@dataclass(frozen=True, slots=True)
class CleanupBatchResult:
    scanned: int = 0
    cleaned: int = 0
    skipped: int = 0
    failed: int = 0
    idempotency_records_purged: int = 0
    audit_records_purged: int = 0


def retention_policy_from_settings(settings: Settings) -> RetentionPolicy:
    return RetentionPolicy(
        active_run_max_runtime=timedelta(seconds=settings.run_runtime_deadline_seconds),
        completed_run=timedelta(seconds=settings.completed_run_retention_seconds),
        failed_run=timedelta(seconds=settings.failed_run_retention_seconds),
        cancelled_run=timedelta(seconds=settings.cancelled_run_retention_seconds),
        completed_checkpoint=timedelta(seconds=settings.completed_run_retention_seconds),
        failed_checkpoint=timedelta(seconds=settings.failed_run_retention_seconds),
        cancelled_checkpoint=timedelta(seconds=settings.cancelled_run_retention_seconds),
        runtime_audit=timedelta(seconds=settings.runtime_audit_retention_seconds),
        idempotency=timedelta(seconds=settings.idempotency_retention_seconds),
        checkpoint_max_blob_bytes=settings.checkpoint_max_blob_bytes,
        cleanup_batch_size=settings.cleanup_batch_size,
    )


class RuntimeCleanupWorker:
    def __init__(
        self,
        *,
        repository: MySQLRuntimeRepository,
        policy: RetentionPolicy,
        audit_retention: timedelta,
    ) -> None:
        self._repository = repository
        self._policy = policy
        self._audit_retention = audit_retention
        self._logger = logging.getLogger("mysteryverse_ai.runtime_cleanup")

    async def run_once(self) -> CleanupBatchResult:
        scanned = cleaned = skipped = failed = 0
        runs = await self._repository.list_terminal_runs(limit=self._policy.cleanup_batch_size)
        for run in runs:
            scanned += 1
            has_references = await self._repository.has_run_references(run.run_id)
            recovery_active = await self._repository.run_has_active_recovery(run.run_id)
            decision = evaluate_run_cleanup(
                status=run.status,
                terminal_at=run.terminal_at,
                now=utc_now(),
                policy=self._policy,
                has_active_references=has_references,
                recovery_in_progress=recovery_active,
            )
            if not decision.eligible:
                skipped += 1
                continue

            job_id = await self._repository.create_cleanup_job(
                resource_type="run",
                resource_id=run.run_id,
            )
            try:
                await self._repository.write_audit(
                    event_category="runtime.cleanup",
                    result="started",
                    expires_at=utc_now() + self._audit_retention,
                    tenant_id=run.tenant_id,
                    run_id=run.run_id,
                    correlation_id=run.run_id,
                    detail_summary="Retention cleanup started.",
                )
                removed = await self._repository.cleanup_run(run_id=run.run_id)
                if removed:
                    cleaned += 1
                    await self._repository.finish_cleanup_job(
                        job_id=job_id,
                        status="succeeded",
                    )
                else:
                    skipped += 1
                    await self._repository.finish_cleanup_job(
                        job_id=job_id,
                        status="skipped",
                        failure_category="CLEANUP_INELIGIBLE",
                    )
            except Exception as exc:
                failed += 1
                await self._repository.finish_cleanup_job(
                    job_id=job_id,
                    status="failed",
                    failure_category="CLEANUP_FAILED",
                    failure_summary="Runtime cleanup failed.",
                )
                await self._repository.write_audit(
                    event_category="runtime.cleanup",
                    result="failed",
                    expires_at=utc_now() + self._audit_retention,
                    tenant_id=run.tenant_id,
                    run_id=run.run_id,
                    correlation_id=run.run_id,
                    failure_category="CLEANUP_FAILED",
                    detail_summary="Retention cleanup failed.",
                )
                log_structured(
                    self._logger,
                    "runtime cleanup failed",
                    cleanup_job_id=job_id,
                    run_id=run.run_id,
                    error_type=type(exc).__name__,
                )

        idempotency_purged = await self._repository.purge_expired_idempotency(
            limit=self._policy.cleanup_batch_size
        )
        audit_purged = await self._repository.purge_expired_audit(
            limit=self._policy.cleanup_batch_size
        )
        result = CleanupBatchResult(
            scanned=scanned,
            cleaned=cleaned,
            skipped=skipped,
            failed=failed,
            idempotency_records_purged=idempotency_purged,
            audit_records_purged=audit_purged,
        )
        log_structured(
            self._logger,
            "runtime cleanup batch completed",
            scanned=result.scanned,
            cleaned=result.cleaned,
            skipped=result.skipped,
            failed=result.failed,
            idempotency_records_purged=result.idempotency_records_purged,
            audit_records_purged=result.audit_records_purged,
        )
        return result
