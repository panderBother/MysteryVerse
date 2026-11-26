from __future__ import annotations

import argparse
import asyncio
import hashlib
import json
import os
import secrets
from datetime import timedelta
from pathlib import Path

from mysteryverse_ai.agent_runtime.persistence_foundation.lifecycle import RunStatus
from mysteryverse_ai.persistence.checkpoint_cipher import CheckpointCipher
from mysteryverse_ai.persistence.database import RuntimeDatabase
from mysteryverse_ai.persistence.ids import new_ulid
from mysteryverse_ai.persistence.repository import MySQLRuntimeRepository, utc_now


async def seed(database_url: str, metadata_path: Path) -> None:
    database = RuntimeDatabase(database_url=database_url)
    repository = MySQLRuntimeRepository(engine=database.engine)
    run_id = new_ulid()
    checkpoint_id = new_ulid()
    state = {"cursor": 1, "pending": ["step-2"]}
    key = os.getenv("AI_RUNTIME_TEST_CHECKPOINT_KEY", "x" * 32).encode()
    cipher = CheckpointCipher(
        key=key,
        key_id="ci-checkpoint-v1",
        max_blob_bytes=1 << 20,
    )
    try:
        await repository.create_run(
            run_id=run_id,
            tenant_id="tenant-ci",
            task_type="backup-restore",
            actor_type="system",
            actor_id=None,
            trace_id="trace-backup-restore",
            correlation_id="correlation-backup-restore",
            schema_version="1",
            idempotency_scope=f"backup:{run_id}",
            idempotency_key="backup-restore",
            request_hash=hashlib.sha256(run_id.encode()).digest(),
            runtime_deadline_seconds=120,
            idempotency_ttl_seconds=3600,
        )
        claim = await repository.claim_owner(
            run_id=run_id,
            owner_id="seed-worker",
            allowed_statuses=frozenset({RunStatus.QUEUED}),
            lease_ttl_seconds=30,
        )
        await repository.record_step(
            run_id=run_id,
            step_no=1,
            step_type="model",
            node_key="seed",
            status="succeeded",
            schema_version="1",
            owner_id=claim.owner_id,
            fencing_token=claim.fencing_token,
        )
        encrypted = cipher.encrypt(
            state=state,
            checkpoint_id=checkpoint_id,
            run_id=run_id,
            checkpoint_seq=1,
            schema_version="1.0",
            runtime_version="ci-runtime-v1",
            created_at=utc_now(),
            nonce=secrets.token_bytes(12),
        )
        await repository.save_checkpoint(
            checkpoint_id=checkpoint_id,
            run_id=run_id,
            checkpoint_seq=1,
            schema_version="1.0",
            runtime_version="ci-runtime-v1",
            encrypted_blob=encrypted.encrypted_blob,
            compression=encrypted.compression,
            blob_size_bytes=encrypted.blob_size_bytes,
            key_id=encrypted.key_id,
            nonce=encrypted.nonce,
            state_hash=encrypted.state_hash,
            owner_id=claim.owner_id,
            fencing_token=claim.fencing_token,
            expires_at=None,
        )
        await repository.transition_run(
            run_id=run_id,
            current=RunStatus.RUNNING,
            target=RunStatus.FAILED,
            owner_id=claim.owner_id,
            fencing_token=claim.fencing_token,
            failure_category="CI_BACKUP_FIXTURE",
            failure_summary="Synthetic failure for backup verification.",
        )
        await repository.create_recovery_attempt(
            run_id=run_id,
            checkpoint_id=checkpoint_id,
            previous_owner_id=claim.owner_id,
            new_owner_id="recovery-worker",
            fencing_token=claim.fencing_token + 1,
            trigger_category="ci_backup",
        )
        await repository.write_audit(
            event_category="runtime.backup_fixture",
            result="succeeded",
            expires_at=utc_now() + timedelta(days=180),
            tenant_id="tenant-ci",
            run_id=run_id,
            correlation_id="correlation-backup-restore",
            actor_type="system",
            actor_id="seed-worker",
            detail_summary="Synthetic fixture for backup verification.",
        )
        await repository.release_owner(
            run_id=run_id,
            owner_id=claim.owner_id,
            fencing_token=claim.fencing_token,
        )
        metadata = {
            "run_id": run_id,
            "checkpoint_id": checkpoint_id,
            "checkpoint_state": state,
            "fencing_token": claim.fencing_token,
        }
        await asyncio.to_thread(
            metadata_path.parent.mkdir,
            parents=True,
            exist_ok=True,
        )
        await asyncio.to_thread(metadata_path.write_text, json.dumps(metadata), "utf-8")
    finally:
        await database.close()


async def verify(database_url: str, metadata_path: Path) -> None:
    metadata = json.loads(await asyncio.to_thread(metadata_path.read_text, "utf-8"))
    run_id = str(metadata["run_id"])
    checkpoint_id = str(metadata["checkpoint_id"])
    database = RuntimeDatabase(database_url=database_url)
    repository = MySQLRuntimeRepository(engine=database.engine)
    key = os.getenv("AI_RUNTIME_TEST_CHECKPOINT_KEY", "x" * 32).encode()
    cipher = CheckpointCipher(
        key=key,
        key_id="ci-checkpoint-v1",
        max_blob_bytes=1 << 20,
    )
    try:
        run = await repository.get_run(run_id)
        if run is None or run.status != RunStatus.FAILED:
            raise AssertionError("restored Run is missing or has an unexpected status")
        if run.owner_id is not None:
            raise AssertionError("restored Run must not trust a backup Owner Lease")
        checkpoint = await repository.load_checkpoint(
            run_id=run_id,
            checkpoint_id=checkpoint_id,
        )
        if checkpoint is None:
            raise AssertionError("restored Checkpoint is missing")
        payload = cipher.decrypt(
            checkpoint=checkpoint,
            run_id=run_id,
            supported_schema_versions=frozenset({"1.0"}),
        )
        if json.loads(payload) != metadata["checkpoint_state"]:
            raise AssertionError("restored Checkpoint state does not match")
        counts = await repository.count_run_artifacts(run_id)
        required = (
            "steps",
            "step_attempts",
            "checkpoints",
            "recovery_attempts",
            "audit_records",
            "idempotency_records",
        )
        if any(counts[name] < 1 for name in required):
            raise AssertionError(f"restored artifact count is incomplete: {counts}")
        claim = await repository.claim_owner(
            run_id=run_id,
            owner_id="restore-worker",
            allowed_statuses=frozenset({RunStatus.FAILED}),
            lease_ttl_seconds=30,
        )
        if claim.fencing_token <= int(metadata["fencing_token"]):
            raise AssertionError("restored Run did not issue a new Fencing Token")
        await repository.release_owner(
            run_id=run_id,
            owner_id=claim.owner_id,
            fencing_token=claim.fencing_token,
        )
        print(json.dumps({"status": "verified", "run_id": run_id, "counts": counts}))
    finally:
        await database.close()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=["seed", "verify"])
    parser.add_argument("--metadata", type=Path, required=True)
    args = parser.parse_args()
    if args.command == "seed":
        database_url = os.getenv("AI_RUNTIME_TEST_DATABASE_URL", "")
        if not database_url:
            raise ValueError("AI_RUNTIME_TEST_DATABASE_URL is required")
        asyncio.run(seed(database_url, args.metadata))
    else:
        database_url = os.getenv("AI_RUNTIME_RESTORE_DATABASE_URL", "")
        if not database_url:
            raise ValueError("AI_RUNTIME_RESTORE_DATABASE_URL is required")
        asyncio.run(verify(database_url, args.metadata))


if __name__ == "__main__":
    main()
