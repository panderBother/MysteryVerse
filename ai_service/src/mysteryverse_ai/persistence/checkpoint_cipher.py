from __future__ import annotations

import hashlib
import zlib
from dataclasses import dataclass
from datetime import datetime

from cryptography.exceptions import InvalidTag
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

from mysteryverse_ai.agent_runtime.persistence_foundation.checkpoint import (
    CheckpointEnvelope,
    CheckpointIntegrityError,
    encode_checkpoint_state,
    validate_checkpoint_blob_size,
)
from mysteryverse_ai.config import Settings
from mysteryverse_ai.persistence.repository import StoredCheckpoint


@dataclass(frozen=True, slots=True)
class EncryptedCheckpoint:
    checkpoint_id: str
    checkpoint_seq: int
    schema_version: str
    runtime_version: str
    encrypted_blob: bytes
    compression: str
    blob_size_bytes: int
    key_id: str
    nonce: bytes
    state_hash: bytes
    envelope: CheckpointEnvelope


class CheckpointDecryptionError(Exception):
    pass


class CheckpointCipher:
    def __init__(
        self,
        *,
        key: bytes,
        key_id: str,
        max_blob_bytes: int,
        compression_level: int = 9,
    ) -> None:
        if len(key) != 32:
            raise ValueError("checkpoint encryption key must be exactly 32 bytes")
        if max_blob_bytes < 1:
            raise ValueError("checkpoint size limit must be positive")
        self._key = key
        self._key_id = key_id
        self._max_blob_bytes = max_blob_bytes
        self._compression_level = compression_level

    @classmethod
    def from_settings(cls, settings: Settings) -> CheckpointCipher:
        if settings.checkpoint_encryption_key is None:
            raise ValueError("checkpoint encryption key is not configured")
        return cls(
            key=settings.checkpoint_encryption_key.get_secret_value().encode(),
            key_id=settings.checkpoint_encryption_key_id,
            max_blob_bytes=settings.checkpoint_max_blob_bytes,
            compression_level=settings.checkpoint_compression_level,
        )

    def encrypt(
        self,
        *,
        state: dict[str, object],
        checkpoint_id: str,
        run_id: str,
        checkpoint_seq: int,
        schema_version: str,
        runtime_version: str,
        created_at: datetime,
        nonce: bytes,
    ) -> EncryptedCheckpoint:
        if len(nonce) != 12:
            raise ValueError("checkpoint AES-GCM nonce must be 12 bytes")
        serialized, _ = encode_checkpoint_state(state)
        compressed = zlib.compress(serialized, level=self._compression_level)
        validate_checkpoint_blob_size(compressed, max_bytes=self._max_blob_bytes)
        state_hash = hashlib.sha256(compressed).digest()
        aad = self._aad(
            run_id=run_id,
            checkpoint_seq=checkpoint_seq,
            schema_version=schema_version,
        )
        encrypted_blob = AESGCM(self._key).encrypt(nonce, compressed, aad)
        envelope = CheckpointEnvelope(
            checkpoint_id=checkpoint_id,
            run_id=run_id,
            checkpoint_seq=checkpoint_seq,
            schema_version=schema_version,
            runtime_version=runtime_version,
            payload_hash=state_hash.hex(),
            key_id=self._key_id,
            created_at=created_at,
        )
        return EncryptedCheckpoint(
            checkpoint_id=checkpoint_id,
            checkpoint_seq=checkpoint_seq,
            schema_version=schema_version,
            runtime_version=runtime_version,
            encrypted_blob=encrypted_blob,
            compression="zlib",
            blob_size_bytes=len(compressed),
            key_id=self._key_id,
            nonce=nonce,
            state_hash=state_hash,
            envelope=envelope,
        )

    def decrypt(
        self,
        *,
        checkpoint: StoredCheckpoint,
        run_id: str,
        supported_schema_versions: frozenset[str],
    ) -> bytes:
        if checkpoint.key_id != self._key_id:
            raise CheckpointDecryptionError("checkpoint encryption key ID is unavailable")
        if checkpoint.compression != "zlib":
            raise CheckpointDecryptionError("checkpoint compression is unsupported")
        aad = self._aad(
            run_id=run_id,
            checkpoint_seq=checkpoint.checkpoint_seq,
            schema_version=checkpoint.schema_version,
        )
        try:
            compressed = AESGCM(self._key).decrypt(
                checkpoint.nonce,
                checkpoint.encrypted_blob,
                aad,
            )
        except InvalidTag as exc:
            raise CheckpointDecryptionError("checkpoint authentication failed") from exc
        if hashlib.sha256(compressed).digest() != checkpoint.state_hash:
            raise CheckpointIntegrityError("checkpoint state hash mismatch")
        if checkpoint.schema_version not in supported_schema_versions:
            raise CheckpointDecryptionError("checkpoint schema version is unsupported")
        return zlib.decompress(compressed)

    @staticmethod
    def _aad(
        *,
        run_id: str,
        checkpoint_seq: int,
        schema_version: str,
    ) -> bytes:
        return f"mv-checkpoint-v1\n{run_id}\n{checkpoint_seq}\n{schema_version}".encode()
