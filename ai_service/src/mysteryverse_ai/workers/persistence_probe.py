from __future__ import annotations

import argparse
import asyncio
import json
import os
from contextlib import suppress

from redis.asyncio import Redis

from mysteryverse_ai.agent_runtime.persistence_foundation.lifecycle import RunStatus
from mysteryverse_ai.persistence.database import RuntimeDatabase
from mysteryverse_ai.persistence.redis_lease import RedisOwnerLeaseStore
from mysteryverse_ai.persistence.repository import (
    FencingRejectedError,
    MySQLRuntimeRepository,
)


async def run_probe(args: argparse.Namespace) -> None:
    if not os.getenv("AI_RUNTIME_TEST_DATABASE_URL"):
        raise ValueError("AI_RUNTIME_TEST_DATABASE_URL is required")
    if not os.getenv("AI_RUNTIME_TEST_REDIS_URL"):
        raise ValueError("AI_RUNTIME_TEST_REDIS_URL is required")

    database = RuntimeDatabase(database_url=os.environ["AI_RUNTIME_TEST_DATABASE_URL"])
    redis = Redis.from_url(os.environ["AI_RUNTIME_TEST_REDIS_URL"])
    leases = RedisOwnerLeaseStore(client=redis)
    repository = MySQLRuntimeRepository(engine=database.engine)
    try:
        claim = await repository.claim_owner(
            run_id=args.run_id,
            owner_id=args.owner_id,
            allowed_statuses=frozenset({RunStatus.QUEUED, RunStatus.RUNNING}),
            lease_ttl_seconds=args.ttl_seconds,
        )
        lease = await leases.acquire(
            run_id=args.run_id,
            owner_id=args.owner_id,
            fencing_token=claim.fencing_token,
            ttl_seconds=args.ttl_seconds,
        )
        print(
            json.dumps(
                {
                    "event": "claimed",
                    "owner_id": args.owner_id,
                    "fencing_token": claim.fencing_token,
                }
            ),
            flush=True,
        )
        if args.stale_hold_seconds > 0:
            await asyncio.sleep(args.stale_hold_seconds)
        try:
            await repository.record_step(
                run_id=args.run_id,
                step_no=args.step_no,
                step_type="probe",
                node_key=args.mode,
                status="succeeded",
                schema_version="1",
                owner_id=args.owner_id,
                fencing_token=claim.fencing_token,
            )
            outcome = "write_succeeded"
        except FencingRejectedError:
            outcome = "stale_rejected"
        print(
            json.dumps(
                {
                    "event": "finished",
                    "owner_id": args.owner_id,
                    "fencing_token": claim.fencing_token,
                    "outcome": outcome,
                }
            ),
            flush=True,
        )
        with suppress(Exception):
            await leases.release(lease)
        if args.release_owner:
            snapshot = await repository.get_run(args.run_id)
            if snapshot is not None and snapshot.status == RunStatus.RUNNING:
                with suppress(Exception):
                    await repository.transition_run(
                        run_id=args.run_id,
                        current=RunStatus.RUNNING,
                        target=RunStatus.COMPLETED,
                        owner_id=args.owner_id,
                        fencing_token=claim.fencing_token,
                    )
            with suppress(Exception):
                await repository.release_owner(
                    run_id=args.run_id,
                    owner_id=args.owner_id,
                    fencing_token=claim.fencing_token,
                )
    finally:
        await leases.close()
        await database.close()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--run-id", required=True)
    parser.add_argument("--owner-id", required=True)
    parser.add_argument("--mode", required=True)
    parser.add_argument("--ttl-seconds", type=int, default=30)
    parser.add_argument("--step-no", type=int, required=True)
    parser.add_argument("--stale-hold-seconds", type=int, default=0)
    parser.add_argument("--release-owner", action="store_true")
    args = parser.parse_args()
    asyncio.run(run_probe(args))


if __name__ == "__main__":
    main()
