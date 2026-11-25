from __future__ import annotations

import asyncio
import hashlib
import hmac
import re
import time
from collections.abc import Callable

from fastapi import Request

from mysteryverse_ai.config import Settings
from mysteryverse_ai.errors import AppError

_NONCE_PATTERN = re.compile(r"^[A-Za-z0-9._~-]{8,128}$")
_SIGNATURE_PATTERN = re.compile(r"^[a-f0-9]{64}$")


class InMemoryNonceCache:
    def __init__(self, *, clock: Callable[[], float] = time.time) -> None:
        self._clock = clock
        self._entries: dict[str, float] = {}
        self._lock = asyncio.Lock()

    async def accept(self, key: str, *, ttl_seconds: int) -> bool:
        now = self._clock()
        async with self._lock:
            expired = [item for item, expiry in self._entries.items() if expiry <= now]
            for item in expired:
                self._entries.pop(item, None)
            if key in self._entries:
                return False
            self._entries[key] = now + ttl_seconds
            return True


async def verify_request_signature(
    request: Request,
    settings: Settings,
    nonce_cache: InMemoryNonceCache,
    *,
    clock: Callable[[], float] = time.time,
) -> None:
    keys = _configured_keys(settings)
    if not keys:
        return
    timestamp_header = request.headers.get("X-Request-Timestamp", "")
    nonce = request.headers.get("X-Request-Nonce", "")
    signature = request.headers.get("X-Request-Signature", "")
    if not timestamp_header or not nonce or not signature:
        raise integrity_error(
            "REQUEST_SIGNATURE_INVALID",
            "Signed internal requests require timestamp, nonce and signature headers.",
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
        timestamp=timestamp_header,
        nonce=nonce,
        service_id=service_id,
        method=request.method,
        path=request.url.path,
        query=request.url.query,
        body=body,
    )
    if not any(
        hmac.compare_digest(
            signature,
            hmac.new(key, canonical.encode(), hashlib.sha256).hexdigest(),
        )
        for key in keys
    ):
        raise integrity_error(
            "REQUEST_SIGNATURE_INVALID",
            "The request signature is invalid.",
        )
    accepted = await nonce_cache.accept(
        service_id + ":" + nonce,
        ttl_seconds=settings.internal_signature_window_seconds * 2,
    )
    if not accepted:
        raise integrity_error(
            "REQUEST_REPLAYED",
            "The request nonce was already used.",
        )


def canonical_request(
    *,
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
            timestamp,
            nonce,
            service_id,
            method.upper(),
            path,
            query,
            body_hash,
        ]
    )


def _configured_keys(settings: Settings) -> list[bytes]:
    keys = []
    if settings.internal_request_signing_key is not None:
        value = settings.internal_request_signing_key.get_secret_value()
        if value:
            keys.append(value.encode())
    if settings.internal_request_signing_previous_key is not None:
        value = settings.internal_request_signing_previous_key.get_secret_value()
        if value:
            keys.append(value.encode())
    return keys


def integrity_error(code: str, message: str) -> AppError:
    return AppError(
        code=code,
        message=message,
        status_code=401,
    )
