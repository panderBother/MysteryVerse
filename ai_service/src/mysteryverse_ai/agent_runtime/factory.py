from __future__ import annotations

from mysteryverse_ai.agent_runtime.runtime import AgentRuntime
from mysteryverse_ai.config import Settings
from mysteryverse_ai.context.builder import ContextBuilder
from mysteryverse_ai.memory.in_memory import InMemoryMemoryStore
from mysteryverse_ai.providers.factory import build_provider
from mysteryverse_ai.tools.mock import register_mock_tools
from mysteryverse_ai.tools.registry import ToolRegistry


def build_agent_runtime(settings: Settings) -> AgentRuntime:
    tools = ToolRegistry()
    register_mock_tools(tools)
    return AgentRuntime(
        provider=build_provider(settings),
        tools=tools,
        memory=InMemoryMemoryStore(),
        context_builder=ContextBuilder(),
        settings=settings,
    )
