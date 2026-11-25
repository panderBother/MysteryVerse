import asyncio

from mysteryverse_ai.workers.main import run_worker


def test_worker_stops_on_event() -> None:
    async def scenario() -> None:
        stop_event = asyncio.Event()
        task = asyncio.create_task(run_worker(stop_event, interval_seconds=0.01))
        await asyncio.sleep(0)
        stop_event.set()
        await asyncio.wait_for(task, timeout=1)

    asyncio.run(scenario())
