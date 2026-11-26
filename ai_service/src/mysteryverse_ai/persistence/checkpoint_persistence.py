from __future__ import annotations

import json
import secrets

from mysteryverse_ai.persistence.checkpoint_cipher import CheckpointCipher
from mysteryverse_ai.persistence.ids import new_ulid
from mysteryverse_ai.persistence.repository import (
    MySQLRuntimeRepository,
    OwnerClaim,
    utc_now,
)


class RuntimeCheckpointPersistence:
    def __init__(
        self,
        *,
        repository: MySQLRuntimeRepository,
        cipher: CheckpointCipher,
    ) -> None:
        self._repository = repository
        self._cipher = cipher

    async def save_state(
        self,
        *,
        run_id: str,
        checkpoint_seq: int,
        schema_version: str,
        runtime_version: str,
        state: dict[str, object],
        claim: OwnerClaim,
        thread_id: str | None = None,
    ) -> str:
        checkpoint_id = new_ulid()
        now = utc_now()
        encrypted = self._cipher.encrypt(
            state=state,
            checkpoint_id=checkpoint_id,
            run_id=run_id,
            checkpoint_seq=checkpoint_seq,
            schema_version=schema_version,
            runtime_version=runtime_version,
            created_at=now,
            nonce=secrets.token_bytes(12),
        )
        await self._repository.save_checkpoint(
            checkpoint_id=checkpoint_id,
            run_id=run_id,
            checkpoint_seq=checkpoint_seq,
            schema_version=schema_version,
            runtime_version=runtime_version,
            encrypted_blob=encrypted.encrypted_blob,
            compression=encrypted.compression,
            blob_size_bytes=encrypted.blob_size_bytes,
            key_id=encrypted.key_id,
            nonce=encrypted.nonce,
            state_hash=encrypted.state_hash,
            owner_id=claim.owner_id,
            fencing_token=claim.fencing_token,
            expires_at=None,
            thread_id=thread_id,
        )
        return checkpoint_id

    async def load_state(
        self,
        *,
        run_id: str,
        checkpoint_id: str | None,
        supported_schema_versions: frozenset[str],
    ) -> dict[str, object] | None:
        checkpoint = await self._repository.load_checkpoint(
            run_id=run_id,
            checkpoint_id=checkpoint_id,
        )
        if checkpoint is None:
            return None
        payload = self._cipher.decrypt(
            checkpoint=checkpoint,
            run_id=run_id,
            supported_schema_versions=supported_schema_versions,
        )
        decoded = json.loads(payload)
        if not isinstance(decoded, dict):
            raise ValueError("checkpoint state must be a JSON object")
        return decoded
