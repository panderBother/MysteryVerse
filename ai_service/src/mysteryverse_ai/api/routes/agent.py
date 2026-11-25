from __future__ import annotations

import asyncio
from contextlib import suppress
from typing import Annotated

from fastapi import APIRouter, Depends, Request

from mysteryverse_ai.agent_runtime.runtime import AgentRuntime
from mysteryverse_ai.api.dependencies import (
    InMemoryIdempotencyCache,
    get_agent_runtime,
    get_idempotency_cache,
    request_hash,
    require_internal_service,
)
from mysteryverse_ai.errors import AppError
from mysteryverse_ai.schemas.generated.models import AgentRequest, AgentResponse
from mysteryverse_ai.security.acl import (
    AclError,
    ServicePrincipal,
    authorize_resource,
)
from mysteryverse_ai.security.audit import audit_security_event

router = APIRouter(tags=["agent-runtime"])


@router.post("/internal/v1/ai/agent-runs", response_model=AgentResponse)
async def run_agent(
    payload: AgentRequest,
    request: Request,
    principal: Annotated[ServicePrincipal, Depends(require_internal_service)],
    runtime: Annotated[AgentRuntime, Depends(get_agent_runtime)],
    idempotency: Annotated[InMemoryIdempotencyCache, Depends(get_idempotency_cache)],
) -> AgentResponse:
    run_id = str(getattr(payload.run_id, "root", payload.run_id))
    try:
        authorize_resource(
            principal,
            resource_type="run",
            resource_id=run_id,
        )
    except AclError as exc:
        audit_security_event(
            service_id=principal.service_id,
            request_id=str(getattr(request.state, "request_id", "")),
            key_id=principal.key_id,
            action=f"ai:agent:run:{run_id}",
            authentication="pass",
            authorization="fail",
            scopes=principal.scopes,
            failure_reason=exc.reason,
        )
        raise AppError(
            code=exc.reason,
            message="The internal service is not authorized for this run.",
            status_code=403,
        ) from exc

    idempotency_key = request.headers.get("Idempotency-Key", "").strip()
    if not idempotency_key:
        raise AppError(
            code="VALIDATION_ERROR",
            message="Idempotency-Key is required.",
            status_code=400,
        )
    header_request_id = request.headers.get("X-Request-Id", "")
    if header_request_id and header_request_id != payload.request_id:
        raise AppError(
            code="VALIDATION_ERROR",
            message="X-Request-Id must match request_id.",
            status_code=400,
        )
    header_trace_id = request.headers.get("X-Trace-Id", "")
    if payload.trace_id and header_trace_id and payload.trace_id != header_trace_id:
        raise AppError(
            code="VALIDATION_ERROR",
            message="X-Trace-Id must match trace_id.",
            status_code=400,
        )
    resolved_trace_id = payload.trace_id or header_trace_id or payload.request_id
    resolved_payload = payload.model_copy(update={"trace_id": resolved_trace_id})
    digest = request_hash(resolved_payload.model_dump(mode="json"))
    scope = str(getattr(resolved_payload.run_id, "root", resolved_payload.run_id))
    cached = await idempotency.begin(
        scope=scope,
        idempotency_key=idempotency_key,
        request_hash=digest,
    )
    if cached is not None:
        return cached

    cancellation_event = asyncio.Event()

    async def watch_disconnect() -> None:
        while not await request.is_disconnected():  # noqa: ASYNC110
            await asyncio.sleep(0.05)
        cancellation_event.set()

    disconnect_task = asyncio.create_task(watch_disconnect())
    try:
        response = await runtime.run(
            resolved_payload,
            cancellation_event=cancellation_event,
        )
    except Exception:
        await idempotency.fail(scope=scope, idempotency_key=idempotency_key)
        raise
    finally:
        disconnect_task.cancel()
        with suppress(asyncio.CancelledError):
            await disconnect_task

    await idempotency.complete(
        scope=scope,
        idempotency_key=idempotency_key,
        response=response,
    )
    return response
