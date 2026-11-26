from __future__ import annotations

import asyncio
from collections.abc import Callable
from dataclasses import dataclass
from datetime import UTC, datetime, timedelta
from typing import Protocol


@dataclass(frozen=True, slots=True)
class OwnerLease:
    run_id: str
    owner_id: str
    fencing_token: int
    expires_at: datetime

    @classmethod
    def from_redis(
        cls,
        *,
        run_id: str,
        owner_id: str,
        fencing_token: int,
        ttl_seconds: int,
    ) -> OwnerLease:
        return cls(
            run_id=run_id,
            owner_id=owner_id,
            fencing_token=fencing_token,
            expires_at=datetime.now(tz=UTC) + timedelta(seconds=ttl_seconds),
        )

    def with_ttl(self, ttl_seconds: int) -> OwnerLease:
        return OwnerLease(
            run_id=self.run_id,
            owner_id=self.owner_id,
            fencing_token=self.fencing_token,
            expires_at=datetime.now(tz=UTC) + timedelta(seconds=ttl_seconds),
        )


class LeaseConflictError(Exception):
    pass


class LeaseExpiredError(Exception):
    pass


class OwnerLeaseStore(Protocol):
    async def acquire(
        self,
        *,
        run_id: str,
        owner_id: str,
        ttl_seconds: int,
    ) -> OwnerLease: ...

    async def renew(self, lease: OwnerLease, *, ttl_seconds: int) -> OwnerLease: ...

    async def release(self, lease: OwnerLease) -> None: ...

    async def current(self, run_id: str) -> OwnerLease | None: ...

    async def validate_fencing_token(
        self,
        *,
        run_id: str,
        owner_id: str,
        fencing_token: int,
    ) -> bool: ...


class InMemoryOwnerLeaseStore:
    def __init__(self, *, clock: Callable[[], datetime] | None = None) -> None:
        self._clock = clock or (lambda: datetime.now(tz=UTC))
        self._leases: dict[str, OwnerLease] = {}
        self._fencing_tokens: dict[str, int] = {}
        self._lock = asyncio.Lock()

    async def acquire(
        self,
        *,
        run_id: str,
        owner_id: str,
        ttl_seconds: int,
    ) -> OwnerLease:
        if ttl_seconds < 1:
            raise ValueError("lease TTL must be positive")
        now = self._clock()
        async with self._lock:
            current = self._leases.get(run_id)
            if current is not None and current.expires_at > now:
                raise LeaseConflictError(f"run {run_id} already has an active owner")
            previous_token = self._fencing_tokens.get(run_id, 0)
            fencing_token = previous_token + 1
            lease = OwnerLease(
                run_id=run_id,
                owner_id=owner_id,
                fencing_token=fencing_token,
                expires_at=now + timedelta(seconds=ttl_seconds),
            )
            self._fencing_tokens[run_id] = fencing_token
            self._leases[run_id] = lease
            return lease

    async def renew(self, lease: OwnerLease, *, ttl_seconds: int) -> OwnerLease:
        if ttl_seconds < 1:
            raise ValueError("lease TTL must be positive")
        now = self._clock()
        async with self._lock:
            current = self._ensure_current_locked(
                run_id=lease.run_id,
                owner_id=lease.owner_id,
                fencing_token=lease.fencing_token,
                now=now,
            )
            renewed = OwnerLease(
                run_id=current.run_id,
                owner_id=current.owner_id,
                fencing_token=current.fencing_token,
                expires_at=now + timedelta(seconds=ttl_seconds),
            )
            self._leases[lease.run_id] = renewed
            return renewed

    async def release(self, lease: OwnerLease) -> None:
        now = self._clock()
        async with self._lock:
            self._ensure_current_locked(
                run_id=lease.run_id,
                owner_id=lease.owner_id,
                fencing_token=lease.fencing_token,
                now=now,
            )
            self._leases.pop(lease.run_id, None)

    async def current(self, run_id: str) -> OwnerLease | None:
        async with self._lock:
            lease = self._leases.get(run_id)
            if lease is None or lease.expires_at <= self._clock():
                return None
            return lease

    async def validate_fencing_token(
        self,
        *,
        run_id: str,
        owner_id: str,
        fencing_token: int,
    ) -> bool:
        current = await self.current(run_id)
        return (
            current is not None
            and current.owner_id == owner_id
            and current.fencing_token == fencing_token
        )

    def _ensure_current_locked(
        self,
        *,
        run_id: str,
        owner_id: str,
        fencing_token: int,
        now: datetime,
    ) -> OwnerLease:
        current = self._leases.get(run_id)
        if current is None or current.expires_at <= now:
            raise LeaseExpiredError(f"run {run_id} lease has expired")
        if current.owner_id != owner_id or current.fencing_token != fencing_token:
            raise LeaseConflictError(f"run {run_id} lease is owned by a different instance")
        return current
