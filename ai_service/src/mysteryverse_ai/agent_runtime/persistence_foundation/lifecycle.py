from __future__ import annotations

from enum import StrEnum


class RunStatus(StrEnum):
    CREATED = "created"
    QUEUED = "queued"
    RUNNING = "running"
    SUSPENDED = "suspended"
    RECOVERING = "recovering"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


TERMINAL_RUN_STATUSES = frozenset(
    {
        RunStatus.COMPLETED,
        RunStatus.FAILED,
        RunStatus.CANCELLED,
    }
)

_ALLOWED_TRANSITIONS: dict[RunStatus, frozenset[RunStatus]] = {
    RunStatus.CREATED: frozenset({RunStatus.QUEUED, RunStatus.CANCELLED}),
    RunStatus.QUEUED: frozenset({RunStatus.RUNNING, RunStatus.CANCELLED}),
    RunStatus.RUNNING: frozenset(
        {
            RunStatus.SUSPENDED,
            RunStatus.RECOVERING,
            RunStatus.COMPLETED,
            RunStatus.FAILED,
            RunStatus.CANCELLED,
        }
    ),
    RunStatus.SUSPENDED: frozenset(
        {
            RunStatus.RECOVERING,
            RunStatus.FAILED,
            RunStatus.CANCELLED,
        }
    ),
    RunStatus.RECOVERING: frozenset(
        {
            RunStatus.RUNNING,
            RunStatus.FAILED,
            RunStatus.CANCELLED,
        }
    ),
    RunStatus.COMPLETED: frozenset(),
    RunStatus.FAILED: frozenset({RunStatus.RECOVERING}),
    RunStatus.CANCELLED: frozenset(),
}


class InvalidRunTransitionError(Exception):
    def __init__(self, *, current: RunStatus, target: RunStatus) -> None:
        super().__init__(f"invalid run transition: {current.value} -> {target.value}")
        self.current = current
        self.target = target


def transition_run_status(current: RunStatus, target: RunStatus) -> RunStatus:
    if current == target:
        return current
    if current == RunStatus.FAILED and target == RunStatus.RECOVERING:
        return target
    if current in TERMINAL_RUN_STATUSES:
        raise InvalidRunTransitionError(current=current, target=target)
    if target not in _ALLOWED_TRANSITIONS[current]:
        raise InvalidRunTransitionError(current=current, target=target)
    return target
