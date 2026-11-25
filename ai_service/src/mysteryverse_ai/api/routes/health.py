from datetime import UTC, datetime
from typing import Literal

from fastapi import APIRouter, Request
from pydantic import BaseModel

from mysteryverse_ai import __version__

router = APIRouter(tags=["health"])


class HealthData(BaseModel):
    service: str
    status: Literal["live", "ready"]
    timestamp: datetime
    version: str


class HealthMeta(BaseModel):
    request_id: str
    trace_id: str


class HealthResponse(BaseModel):
    data: HealthData
    meta: HealthMeta


def _payload(request: Request, status: Literal["live", "ready"]) -> HealthResponse:
    return HealthResponse(
        data=HealthData(
            service="ai-api",
            status=status,
            timestamp=datetime.now(tz=UTC),
            version=__version__,
        ),
        meta=HealthMeta(
            request_id=getattr(request.state, "request_id", ""),
            trace_id=getattr(request.state, "trace_id", ""),
        ),
    )


@router.get("/health/live", response_model=HealthResponse)
async def live(request: Request) -> HealthResponse:
    return _payload(request, "live")


@router.get("/health/ready", response_model=HealthResponse)
async def ready(request: Request) -> HealthResponse:
    return _payload(request, "ready")


@router.get("/internal/v1/health", response_model=HealthResponse)
async def internal_health(request: Request) -> HealthResponse:
    return _payload(request, "ready")
