import asyncio
import logging
import signal
from contextlib import suppress
from datetime import timedelta

from mysteryverse_ai.config import get_settings
from mysteryverse_ai.logging import configure_logging
from mysteryverse_ai.persistence.cleanup import (
    RuntimeCleanupWorker,
    retention_policy_from_settings,
)
from mysteryverse_ai.persistence.database import build_runtime_database
from mysteryverse_ai.persistence.repository import MySQLRuntimeRepository


async def run_worker(
    stop_event: asyncio.Event,
    interval_seconds: float = 5.0,
    cleanup_worker: RuntimeCleanupWorker | None = None,
    cleanup_interval_seconds: float = 300.0,
) -> None:
    logger = logging.getLogger("mysteryverse_ai.worker")
    logger.info("worker started", extra={"structured": {"worker": "ai-worker"}})

    next_cleanup = 0.0
    loop = asyncio.get_running_loop()
    try:
        while not stop_event.is_set():
            if (
                cleanup_worker is not None
                and loop.time() >= next_cleanup
                and not stop_event.is_set()
            ):
                with suppress(Exception):
                    await cleanup_worker.run_once()
                next_cleanup = loop.time() + cleanup_interval_seconds
            with suppress(TimeoutError):
                timeout = min(
                    interval_seconds,
                    max(next_cleanup - loop.time(), 0.1)
                    if cleanup_worker is not None
                    else interval_seconds,
                )
                await asyncio.wait_for(stop_event.wait(), timeout=timeout)
    finally:
        logger.info("worker stopped", extra={"structured": {"worker": "ai-worker"}})


async def _main() -> None:
    settings = get_settings()
    configure_logging(settings.log_level)

    stop_event = asyncio.Event()
    loop = asyncio.get_running_loop()

    for signal_name in (signal.SIGINT, signal.SIGTERM):
        with suppress(NotImplementedError):
            loop.add_signal_handler(signal_name, stop_event.set)

    cleanup_worker = None
    database = None
    if settings.runtime_persistence_enabled:
        database = build_runtime_database(settings)
        repository = MySQLRuntimeRepository(engine=database.engine)
        cleanup_worker = RuntimeCleanupWorker(
            repository=repository,
            policy=retention_policy_from_settings(settings),
            audit_retention=timedelta(seconds=settings.runtime_audit_retention_seconds),
        )
    try:
        await run_worker(
            stop_event,
            cleanup_worker=cleanup_worker,
            cleanup_interval_seconds=float(settings.cleanup_interval_seconds),
        )
    finally:
        if database is not None:
            await database.close()


def main() -> None:
    asyncio.run(_main())


if __name__ == "__main__":
    main()
