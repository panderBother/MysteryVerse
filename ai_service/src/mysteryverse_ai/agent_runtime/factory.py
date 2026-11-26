from __future__ import annotations

from redis.asyncio import Redis

from mysteryverse_ai.agent_runtime.run_store import AgentRunStore
from mysteryverse_ai.agent_runtime.runtime import AgentRuntime
from mysteryverse_ai.config import Settings
from mysteryverse_ai.context.builder import ContextBuilder
from mysteryverse_ai.memory.in_memory import InMemoryMemoryStore
from mysteryverse_ai.persistence.database import build_runtime_database
from mysteryverse_ai.persistence.mysql_run_store import MySQLAgentRunStore
from mysteryverse_ai.persistence.redis_lease import RedisOwnerLeaseStore
from mysteryverse_ai.persistence.repository import MySQLRuntimeRepository
from mysteryverse_ai.providers.factory import build_provider
from mysteryverse_ai.tools.mock import register_mock_tools
from mysteryverse_ai.tools.registry import ToolRegistry


def build_agent_runtime(settings: Settings) -> AgentRuntime:
    tools = ToolRegistry()
    register_mock_tools(tools)
    run_store: AgentRunStore | None = None
    if settings.runtime_persistence_enabled:
        if not settings.redis_url:
            raise ValueError("runtime persistence requires AI_REDIS_URL")
        database = build_runtime_database(settings)
        redis_client = Redis.from_url(
            settings.redis_url,
            encoding="utf-8",
            decode_responses=False,
        )
        lease_store = RedisOwnerLeaseStore(client=redis_client)
        repository = MySQLRuntimeRepository(engine=database.engine)
        run_store = MySQLAgentRunStore(
            database=database,
            repository=repository,
            lease_store=lease_store,
            owner_id=settings.runtime_instance_id,
            runtime_deadline_seconds=settings.run_runtime_deadline_seconds,
            idempotency_ttl_seconds=settings.idempotency_retention_seconds,
            lease_ttl_seconds=settings.owner_lease_ttl_seconds,
        )
    return AgentRuntime(
        provider=build_provider(settings),
        tools=tools,
        memory=InMemoryMemoryStore(),
        context_builder=ContextBuilder(),
        settings=settings,
        run_store=run_store,
    )
