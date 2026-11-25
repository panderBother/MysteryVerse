import asyncio
import logging
import signal
from contextlib import suppress

from mysteryverse_ai.config import get_settings
from mysteryverse_ai.logging import configure_logging


async def run_worker(stop_event: asyncio.Event, interval_seconds: float = 5.0) -> None:
    logger = logging.getLogger("mysteryverse_ai.worker")
    logger.info("worker started", extra={"structured": {"worker": "ai-worker"}})

    try:
        while not stop_event.is_set():
            with suppress(TimeoutError):
                await asyncio.wait_for(stop_event.wait(), timeout=interval_seconds)
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

    await run_worker(stop_event)


def main() -> None:
    asyncio.run(_main())


if __name__ == "__main__":
    main()
