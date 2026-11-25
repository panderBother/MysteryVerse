from __future__ import annotations

import hashlib
import hmac
import re
import time
from collections.abc import Callable
from dataclasses import dataclass

from fastapi import Request

from mysteryverse_ai.config import Settings
from mysteryverse_ai.errors import AppError
from mysteryverse_ai.security.keys import KeyProvider, KeyProviderError
from mysteryverse_ai.security.nonce import NonceStore

REQUEST_SIGNATURE_VERSION = "mv-hmac-sha256-v1"
_NONCE_PATTERN = re.compile(r"^[A-Za-z0-9._~-]{8,128}$")
_KEY_ID_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{2,127}$")
_SIGNATURE_PATTERN = re.compile(r"^[a-f0-9]{64}$")


@dataclass(frozen=True, slots=True)
class SignatureVerificationResult:
    key_id: str | None
    signature_version: str | None


async def verify_request_signature(
    request: Request,
    settings: Settings,
    nonce_store: NonceStore,
    key_provider: KeyProvider,
    *,
    clock: Callable[[], float] = time.time,
) -> SignatureVerificationResult:
    signing_configured = (
        settings.internal_request_signing_key is not None
        or settings.internal_request_signing_previous_key is not None
    )
    if not signing_configured:
        return SignatureVerificationResult(key_id=None, signature_version=None)

    signature_version = request.headers.get("X-Request-Signature-Version", "")
    key_id = request.headers.get("X-Service-Key-Id", "")
    timestamp_header = request.headers.get("X-Request-Timestamp", "")
    nonce = request.headers.get("X-Request-Nonce", "")
    signature = request.headers.get("X-Request-Signature", "")
    if not all(
        (
            signature_version,
            key_id,
            timestamp_header,
            nonce,
            signature,
        )
    ):
        raise integrity_error(
            "REQUEST_SIGNATURE_INVALID",
            "Signed internal requests require signature version, key ID, "
            "timestamp, nonce and signature headers.",
        )
    if signature_version != REQUEST_SIGNATURE_VERSION:
        raise integrity_error(
            "REQUEST_SIGNATURE_VERSION_UNSUPPORTED",
            "The internal request signature version is not supported.",
        )
    if not _KEY_ID_PATTERN.fullmatch(key_id):
        raise integrity_error(
            "SERVICE_KEY_INVALID",
            "X-Service-Key-Id is invalid.",
        )
    try:
        key = key_provider.key_by_id(key_id, "internal.request-signing")
    except KeyProviderError as exc:
        raise AppError(
            code="SERVICE_KEY_UNAVAILABLE",
            message="Internal signing key material is unavailable.",
            status_code=503,
            retryable=True,
        ) from exc
    if key is None:
        raise integrity_error(
            "SERVICE_KEY_INVALID",
            "The internal signing key is unknown or revoked.",
        )

    try:
        timestamp = int(timestamp_header)
    except ValueError as exc:
        raise integrity_error(
            "REQUEST_TIMESTAMP_INVALID",
            "X-Request-Timestamp must be Unix seconds.",
        ) from exc
    if abs(clock() - timestamp) > settings.internal_signature_window_seconds:
        raise integrity_error(
            "REQUEST_TIMESTAMP_INVALID",
            "The request timestamp is outside the accepted window.",
        )
    if not _NONCE_PATTERN.fullmatch(nonce):
        raise integrity_error(
            "REQUEST_SIGNATURE_INVALID",
            "X-Request-Nonce is invalid.",
        )
    if not _SIGNATURE_PATTERN.fullmatch(signature):
        raise integrity_error(
            "REQUEST_SIGNATURE_INVALID",
            "X-Request-Signature must be lowercase hex SHA-256.",
        )

    body = await request.body()
    service_id = request.headers.get("X-Service-Id", "")
    canonical = canonical_request(
        signature_version=signature_version,
        key_id=key_id,
        timestamp=timestamp_header,
        nonce=nonce,
        service_id=service_id,
        method=request.method,
        path=request.url.path,
        query=request.url.query,
        body=body,
    )
    expected = hmac.new(key.secret, canonical.encode(), hashlib.sha256).hexdigest()
    if not hmac.compare_digest(signature, expected):
        raise integrity_error(
            "REQUEST_SIGNATURE_INVALID",
            "The request signature is invalid.",
        )

    try:
        accepted = await nonce_store.accept(
            service_id + ":" + key.key_id + ":" + nonce,
            ttl_seconds=settings.internal_signature_window_seconds * 2,
        )
    except Exception as exc:
        raise AppError(
            code="NONCE_STORE_UNAVAILABLE",
            message="The internal replay-protection store is unavailable.",
            status_code=503,
            retryable=True,
        ) from exc
    if not accepted:
        raise integrity_error(
            "REQUEST_REPLAYED",
            "The request nonce was already used.",
        )
    return SignatureVerificationResult(
        key_id=key.key_id,
        signature_version=signature_version,
    )


def canonical_request(
    *,
    signature_version: str,
    key_id: str,
    timestamp: str,
    nonce: str,
    service_id: str,
    method: str,
    path: str,
    query: str,
    body: bytes,
) -> str:
    body_hash = hashlib.sha256(body).hexdigest()
    return "\n".join(
        [
            signature_version,
            key_id,
            timestamp,
            nonce,
            service_id,
            method.upper(),
            path,
            query,
            body_hash,
        ]
    )


def integrity_error(code: str, message: str) -> AppError:
    return AppError(
        code=code,
        message=message,
        status_code=401,
    )
