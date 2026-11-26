from __future__ import annotations

import hashlib
import json
from datetime import datetime
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field


class CheckpointIntegrityError(Exception):
    pass


class CheckpointCompatibilityError(Exception):
    pass


class CheckpointSizeError(Exception):
    pass


class CheckpointEnvelope(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    checkpoint_id: str = Field(min_length=1)
    run_id: str = Field(min_length=1)
    checkpoint_seq: int = Field(ge=1)
    schema_version: str = Field(min_length=1)
    runtime_version: str = Field(min_length=1)
    status: Literal["valid", "superseded", "incompatible"] = "valid"
    storage_ref: str | None = None
    payload_hash: str = Field(pattern="^[a-f0-9]{64}$")
    key_id: str | None = None
    created_at: datetime


def encode_checkpoint_state(state: dict[str, Any]) -> tuple[bytes, str]:
    payload = json.dumps(
        state,
        ensure_ascii=True,
        sort_keys=True,
        separators=(",", ":"),
    ).encode()
    return payload, hashlib.sha256(payload).hexdigest()


def verify_checkpoint(
    envelope: CheckpointEnvelope,
    payload: bytes,
    *,
    supported_schema_versions: frozenset[str],
) -> None:
    if envelope.status != "valid":
        raise CheckpointCompatibilityError("checkpoint is not valid")
    if envelope.schema_version not in supported_schema_versions:
        raise CheckpointCompatibilityError("checkpoint schema version is unsupported")
    actual_hash = hashlib.sha256(payload).hexdigest()
    if actual_hash != envelope.payload_hash:
        raise CheckpointIntegrityError("checkpoint payload hash does not match")


def validate_checkpoint_blob_size(payload: bytes, *, max_bytes: int) -> None:
    if max_bytes < 1:
        raise ValueError("checkpoint size limit must be positive")
    if len(payload) > max_bytes:
        raise CheckpointSizeError("checkpoint payload exceeds the configured size limit")
