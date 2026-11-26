from __future__ import annotations

import asyncio
import json
import os
import subprocess
import sys
import time
from pathlib import Path

import pytest
from sqlalchemy.ext.asyncio import create_async_engine

from mysteryverse_ai.persistence.ids import new_ulid
from mysteryverse_ai.persistence.repository import MySQLRuntimeRepository

ROOT = Path(__file__).resolve().parents[2]


def _database_url() -> str:
    value = os.getenv("AI_RUNTIME_TEST_DATABASE_URL", "")
    if value.startswith("mysql://"):
        return "mysql+asyncmy://" + value.removeprefix("mysql://")
    return value


@pytest.mark.integration
def test_two_worker_failover_and_stale_fence_rejection() -> None:
    database_url = _database_url()
    redis_url = os.getenv("AI_RUNTIME_TEST_REDIS_URL", "")
    if not database_url or not redis_url:
        pytest.skip("AI_RUNTIME_TEST_DATABASE_URL and AI_RUNTIME_TEST_REDIS_URL are required")

    run_id = new_ulid()

    async def prepare() -> None:
        engine = create_async_engine(database_url)
        repository = MySQLRuntimeRepository(engine=engine)
        try:
            await repository.create_run(
                run_id=run_id,
                tenant_id="tenant-test",
                task_type="multiworker",
                actor_type="system",
                actor_id=None,
                trace_id="trace-multiworker",
                correlation_id="correlation-multiworker",
                schema_version="1",
                idempotency_scope=f"multiworker:{run_id}",
                idempotency_key="multiworker",
                request_hash=b"c" * 32,
                runtime_deadline_seconds=120,
                idempotency_ttl_seconds=60,
            )
        finally:
            await engine.dispose()

    asyncio.run(prepare())
    env = {
        **os.environ,
        "AI_RUNTIME_TEST_DATABASE_URL": database_url,
        "AI_RUNTIME_TEST_REDIS_URL": redis_url,
    }
    worker_module = "mysteryverse_ai.workers.persistence_probe"
    worker_a = subprocess.Popen(
        [
            sys.executable,
            "-m",
            worker_module,
            "--run-id",
            run_id,
            "--owner-id",
            "worker-a",
            "--mode",
            "stale-owner",
            "--ttl-seconds",
            "2",
            "--step-no",
            "1",
            "--stale-hold-seconds",
            "5",
        ],
        cwd=ROOT,
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    assert worker_a.stdout is not None
    claimed = json.loads(worker_a.stdout.readline())
    assert claimed["event"] == "claimed"
    assert claimed["fencing_token"] == 1

    time.sleep(3)
    worker_b = subprocess.Popen(
        [
            sys.executable,
            "-m",
            worker_module,
            "--run-id",
            run_id,
            "--owner-id",
            "worker-b",
            "--mode",
            "takeover",
            "--ttl-seconds",
            "10",
            "--step-no",
            "2",
            "--release-owner",
        ],
        cwd=ROOT,
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    worker_b_output, worker_b_error = worker_b.communicate(timeout=30)
    assert worker_b.returncode == 0, worker_b_error
    assert '"fencing_token": 2' in worker_b_output
    assert '"outcome": "write_succeeded"' in worker_b_output

    worker_a_output, worker_a_error = worker_a.communicate(timeout=30)
    assert worker_a.returncode == 0, worker_a_error
    assert '"outcome": "stale_rejected"' in worker_a_output

    async def cleanup() -> None:
        engine = create_async_engine(database_url)
        repository = MySQLRuntimeRepository(engine=engine)
        try:
            await repository.cleanup_run(run_id=run_id)
        finally:
            await engine.dispose()

    asyncio.run(cleanup())
