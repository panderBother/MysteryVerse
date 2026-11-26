from __future__ import annotations

from redis.asyncio import Redis

from mysteryverse_ai.agent_runtime.persistence_foundation.lease import (
    LeaseConflictError,
    LeaseExpiredError,
    OwnerLease,
)

_ACQUIRE_SCRIPT = """
if redis.call('EXISTS', KEYS[1]) == 0 then
  redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[2])
  return 1
end
return 0
"""

_RENEW_SCRIPT = """
if redis.call('GET', KEYS[1]) == ARGV[1] then
  redis.call('PEXPIRE', KEYS[1], ARGV[2])
  return 1
end
return 0
"""

_RELEASE_SCRIPT = """
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0
"""


class RedisOwnerLeaseStore:
    def __init__(self, *, client: Redis, key_prefix: str = "mv:ai:lease") -> None:
        self._client = client
        self._key_prefix = key_prefix.rstrip(":")

    async def acquire(
        self,
        *,
        run_id: str,
        owner_id: str,
        fencing_token: int,
        ttl_seconds: int,
    ) -> OwnerLease:
        if fencing_token < 1 or ttl_seconds < 1:
            raise ValueError("fencing token and lease TTL must be positive")
        value = self._lease_value(owner_id, fencing_token)
        accepted = await self._client.eval(
            _ACQUIRE_SCRIPT,
            1,
            self._key(run_id),
            value,
            ttl_seconds * 1000,
        )
        if int(accepted) != 1:
            raise LeaseConflictError(f"run {run_id} already has an active owner")
        return OwnerLease.from_redis(
            run_id=run_id,
            owner_id=owner_id,
            fencing_token=fencing_token,
            ttl_seconds=ttl_seconds,
        )

    async def renew(
        self,
        lease: OwnerLease,
        *,
        ttl_seconds: int,
    ) -> OwnerLease:
        if ttl_seconds < 1:
            raise ValueError("lease TTL must be positive")
        renewed = await self._client.eval(
            _RENEW_SCRIPT,
            1,
            self._key(lease.run_id),
            self._lease_value(lease.owner_id, lease.fencing_token),
            ttl_seconds * 1000,
        )
        if int(renewed) != 1:
            raise LeaseExpiredError(f"run {lease.run_id} lease is no longer owned")
        return lease.with_ttl(ttl_seconds)

    async def release(self, lease: OwnerLease) -> None:
        released = await self._client.eval(
            _RELEASE_SCRIPT,
            1,
            self._key(lease.run_id),
            self._lease_value(lease.owner_id, lease.fencing_token),
        )
        if int(released) != 1:
            raise LeaseExpiredError(f"run {lease.run_id} lease is no longer owned")

    async def validate_fencing_token(
        self,
        *,
        run_id: str,
        owner_id: str,
        fencing_token: int,
    ) -> bool:
        value = await self._client.get(self._key(run_id))
        if value is None:
            return False
        decoded = value.decode() if isinstance(value, bytes) else str(value)
        return decoded == self._lease_value(owner_id, fencing_token)

    async def close(self) -> None:
        await self._client.aclose()

    def _key(self, run_id: str) -> str:
        return f"{self._key_prefix}:{run_id}"

    @staticmethod
    def _lease_value(owner_id: str, fencing_token: int) -> str:
        return f"{owner_id}:{fencing_token}"
