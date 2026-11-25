from __future__ import annotations

import hashlib
import json
from typing import Any

from mysteryverse_ai.schemas.generated.models import (
    AgentTraceStep,
    AgentUsage,
    Status,
    StepType,
)


def hash_payload(value: Any) -> str:
    raw = json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"), default=str)
    return hashlib.sha256(raw.encode()).hexdigest()


def trace_step(
    *,
    step_index: int,
    step_type: StepType,
    name: str,
    status: Status,
    duration_ms: int,
    usage: AgentUsage | None = None,
    model_name: str | None = None,
    error_code: str | None = None,
    summary: Any | None = None,
) -> AgentTraceStep:
    return AgentTraceStep(
        step_index=step_index,
        step_type=step_type,
        name=name,
        status=status,
        duration_ms=max(duration_ms, 0),
        model_name=model_name,
        input_tokens=usage.input_tokens if usage else 0,
        output_tokens=usage.output_tokens if usage else 0,
        error_code=error_code,
        summary_hash=hash_payload(summary) if summary is not None else None,
    )
