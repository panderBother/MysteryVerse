from __future__ import annotations

import asyncio
import hashlib
import json
import secrets
from dataclasses import dataclass

from fastapi import Request

from mysteryverse_ai.agent_runtime.runtime import AgentRuntime
from mysteryverse_ai.api.request_integrity import verify_request_signature
from mysteryverse_ai.config import Settings
from mysteryverse_ai.errors import AppError
from mysteryverse_ai.schemas.generated.models import AgentResponse
from mysteryverse_ai.security.acl import (
    AclError,
    ServicePrincipal,
    authorize_operation,
    authorize_tenant,
    parse_service_scopes,
    parse_tenant,
)
from mysteryverse_ai.security.audit import audit_security_event
from mysteryverse_ai.security.keys import KeyProvider
from mysteryverse_ai.security.nonce import NonceStore


@dataclass(slots=True)
class _IdempotencyRecord:
    request_hash: str
    response: AgentResponse | None = None


class InMemoryIdempotencyCache:
    def __init__(self) -> None:
        self._records: dict[str, _IdempotencyRecord] = {}
        self._lock = asyncio.Lock()

    async def begin(
        self,
        *,
        scope: str,
        idempotency_key: str,
        request_hash: str,
    ) -> AgentResponse | None:
        cache_key = scope + ":" + idempotency_key
        async with self._lock:
            existing = self._records.get(cache_key)
            if existing is None:
                self._records[cache_key] = _IdempotencyRecord(request_hash=request_hash)
                return None
            if existing.request_hash != request_hash:
                raise AppError(
                    code="IDEMPOTENCY_CONFLICT",
                    message="The idempotency key was reused with a different request.",
                    status_code=409,
                )
            if existing.response is None:
                raise AppError(
                    code="IDEMPOTENCY_CONFLICT",
                    message="The previous request is still in progress.",
                    status_code=409,
                    retryable=True,
                )
            return existing.response

    async def complete(
        self,
        *,
        scope: str,
        idempotency_key: str,
        response: AgentResponse,
    ) -> None:
        cache_key = scope + ":" + idempotency_key
        async with self._lock:
            record = self._records.get(cache_key)
            if record is not None:
                record.response = response

    async def fail(self, *, scope: str, idempotency_key: str) -> None:
        cache_key = scope + ":" + idempotency_key
        async with self._lock:
            self._records.pop(cache_key, None)


def request_hash(payload: object) -> str:
    if hasattr(payload, "model_dump"):
        payload = payload.model_dump(mode="json")  # type: ignore[union-attr]
    raw = json.dumps(payload, ensure_ascii=False, sort_keys=True, separators=(",", ":"))
    return hashlib.sha256(raw.encode()).hexdigest()


async def require_internal_service(request: Request) -> ServicePrincipal:
    settings: Settings = request.app.state.settings
    request_id = str(getattr(request.state, "request_id", ""))
    service_id = request.headers.get("X-Service-Id", "")
    key_id = request.headers.get("X-Service-Key-Id", "")
    scopes: frozenset[str] = frozenset()
    tenant_id: str | None = None
    configured_token = (
        settings.internal_service_token.get_secret_value()
        if settings.internal_service_token is not None
        else ""
    )
    if not configured_token:
        audit_security_event(
            service_id=service_id,
            request_id=request_id,
            key_id=key_id or None,
            action=request.url.path,
            authentication="not_configured",
            authorization="not_evaluated",
            scopes=scopes,
            failure_reason="SERVICE_AUTH_NOT_CONFIGURED",
        )
        raise AppError(
            code="SERVICE_UNAVAILABLE",
            message="Internal AI service authentication is not configured.",
            status_code=503,
        )
    token = request.headers.get("X-Service-Token", "")
    if not secrets.compare_digest(token, configured_token) or not secrets.compare_digest(
        service_id,
        settings.internal_service_id,
    ):
        audit_security_event(
            service_id=service_id,
            request_id=request_id,
            key_id=key_id or None,
            action=request.url.path,
            authentication="fail",
            authorization="not_evaluated",
            scopes=scopes,
            failure_reason="SERVICE_IDENTITY_INVALID",
        )
        raise AppError(
            code="FORBIDDEN_ACTION",
            message="The internal service identity is invalid.",
            status_code=403,
        )

    try:
        scopes = parse_service_scopes(request.headers.get("X-Service-Scope", ""))
        tenant_id = parse_tenant(request.headers.get("X-Service-Tenant", ""))
        nonce_store: NonceStore = request.app.state.nonce_store
        key_provider: KeyProvider = request.app.state.key_provider
        signature = await verify_request_signature(
            request,
            settings,
            nonce_store,
            key_provider,
        )
        principal = ServicePrincipal(
            service_id=service_id,
            scopes=scopes,
            key_id=signature.key_id,
            tenant_id=tenant_id,
        )
        authorize_operation(principal, settings.internal_required_agent_run_scope)
        authorize_tenant(principal)
    except AclError as exc:
        audit_security_event(
            service_id=service_id,
            request_id=request_id,
            key_id=key_id or None,
            action=request.url.path,
            authentication="pass",
            authorization="fail",
            scopes=scopes,
            failure_reason=exc.reason,
        )
        raise AppError(
            code=exc.reason,
            message="The internal service is not authorized for this operation.",
            status_code=403,
        ) from exc
    except AppError as exc:
        audit_security_event(
            service_id=service_id,
            request_id=request_id,
            key_id=key_id or None,
            action=request.url.path,
            authentication="fail",
            authorization="not_evaluated",
            scopes=scopes,
            failure_reason=exc.code,
        )
        raise

    request.state.service_principal = principal
    audit_security_event(
        service_id=service_id,
        request_id=request_id,
        key_id=principal.key_id,
        action=request.url.path,
        authentication="pass",
        authorization="pass",
        scopes=principal.scopes,
        failure_reason=None,
    )
    return principal


def get_agent_runtime(request: Request) -> AgentRuntime:
    runtime: AgentRuntime = request.app.state.agent_runtime
    return runtime


def get_idempotency_cache(request: Request) -> InMemoryIdempotencyCache:
    cache: InMemoryIdempotencyCache = request.app.state.idempotency_cache
    return cache


def get_nonce_store(request: Request) -> NonceStore:
    store: NonceStore = request.app.state.nonce_store
    return store
