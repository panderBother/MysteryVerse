import asyncio
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager, suppress

import uvicorn
from fastapi import FastAPI
from fastapi.exceptions import RequestValidationError
from starlette.exceptions import HTTPException as StarletteHTTPException

from mysteryverse_ai.agent_runtime.factory import build_agent_runtime
from mysteryverse_ai.agent_runtime.runtime import AgentRuntime
from mysteryverse_ai.api.dependencies import InMemoryIdempotencyCache
from mysteryverse_ai.api.request_integrity import InMemoryNonceCache
from mysteryverse_ai.api.routes.agent import router as agent_router
from mysteryverse_ai.api.routes.health import router as health_router
from mysteryverse_ai.config import Settings, get_settings
from mysteryverse_ai.errors import (
    AppError,
    app_error_handler,
    http_exception_handler,
    request_validation_error_handler,
    unhandled_error_handler,
)
from mysteryverse_ai.logging import configure_logging
from mysteryverse_ai.middleware import RequestIDMiddleware


def create_app(
    settings: Settings | None = None,
    runtime: AgentRuntime | None = None,
) -> FastAPI:
    resolved_settings = settings or get_settings()
    configure_logging(resolved_settings.log_level)
    resolved_runtime = runtime or build_agent_runtime(resolved_settings)

    @asynccontextmanager
    async def lifespan(_app: FastAPI) -> AsyncIterator[None]:
        try:
            yield
        finally:
            await resolved_runtime.aclose()

    app = FastAPI(
        title="MysteryVerse AI",
        version="0.1.0",
        docs_url=None if resolved_settings.environment == "production" else "/docs",
        redoc_url=None,
        lifespan=lifespan,
    )
    app.state.settings = resolved_settings
    app.state.agent_runtime = resolved_runtime
    app.state.idempotency_cache = InMemoryIdempotencyCache()
    app.state.nonce_cache = InMemoryNonceCache()

    app.add_middleware(RequestIDMiddleware)
    app.add_exception_handler(AppError, app_error_handler)
    app.add_exception_handler(RequestValidationError, request_validation_error_handler)
    app.add_exception_handler(StarletteHTTPException, http_exception_handler)
    app.add_exception_handler(Exception, unhandled_error_handler)
    app.include_router(health_router)
    app.include_router(agent_router)
    return app


app = create_app()


async def _run_server() -> None:
    settings: Settings = app.state.settings
    config = uvicorn.Config(
        app=app,
        host=settings.host,
        port=settings.port,
        log_config=None,
        access_log=False,
    )
    server = uvicorn.Server(config)
    await server.serve()


def main() -> None:
    with suppress(KeyboardInterrupt):
        asyncio.run(_run_server())


if __name__ == "__main__":
    main()
