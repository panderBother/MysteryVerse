from __future__ import annotations

import random
from collections.abc import Callable

from pydantic import BaseModel, ConfigDict, Field


class RetryPolicy(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    max_attempts: int = Field(default=3, ge=1, le=10)
    base_delay_ms: int = Field(default=100, ge=1)
    max_delay_ms: int = Field(default=5_000, ge=1)
    jitter_ratio: float = Field(default=0.2, ge=0, le=1)
    retryable_error_codes: frozenset[str] = frozenset({"AI_TIMEOUT", "PROVIDER_UNAVAILABLE"})


class RetryDecision(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    should_retry: bool
    delay_ms: int = Field(ge=0)
    reason: str


def decide_retry(
    *,
    policy: RetryPolicy,
    attempt_number: int,
    error_code: str,
    deadline_exceeded: bool = False,
    random_value: Callable[[], float] = random.random,
) -> RetryDecision:
    if deadline_exceeded:
        return RetryDecision(should_retry=False, delay_ms=0, reason="deadline_exceeded")
    if attempt_number >= policy.max_attempts:
        return RetryDecision(should_retry=False, delay_ms=0, reason="attempt_limit")
    if error_code not in policy.retryable_error_codes:
        return RetryDecision(should_retry=False, delay_ms=0, reason="non_retryable")
    exponential = policy.base_delay_ms * (2 ** max(attempt_number - 1, 0))
    bounded = min(exponential, policy.max_delay_ms)
    jitter = int(bounded * policy.jitter_ratio * random_value())
    return RetryDecision(should_retry=True, delay_ms=bounded + jitter, reason="retryable")
