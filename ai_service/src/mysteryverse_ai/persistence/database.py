from __future__ import annotations

from sqlalchemy.ext.asyncio import AsyncEngine, create_async_engine

from mysteryverse_ai.config import Settings


def normalize_async_url(database_url: str) -> str:
    if database_url.startswith("mysql://"):
        return "mysql+asyncmy://" + database_url.removeprefix("mysql://")
    if database_url.startswith("mysql+asyncmy://"):
        return database_url
    raise ValueError("AI_DATABASE_URL must use mysql:// or mysql+asyncmy://")


class RuntimeDatabase:
    def __init__(self, *, database_url: str, pool_size: int = 10) -> None:
        self.engine: AsyncEngine = create_async_engine(
            normalize_async_url(database_url),
            pool_pre_ping=True,
            pool_size=pool_size,
            max_overflow=max(pool_size // 2, 1),
        )

    async def close(self) -> None:
        await self.engine.dispose()


def build_runtime_database(settings: Settings) -> RuntimeDatabase:
    if not settings.database_url:
        raise ValueError("AI_DATABASE_URL is required")
    return RuntimeDatabase(database_url=settings.database_url)
