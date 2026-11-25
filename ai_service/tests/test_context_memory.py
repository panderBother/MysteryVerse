from __future__ import annotations

import asyncio

from mysteryverse_ai.agent_runtime.outputs import DemoAgentOutput
from mysteryverse_ai.context.builder import ContextBuilder
from mysteryverse_ai.memory.in_memory import InMemoryMemoryStore
from mysteryverse_ai.memory.models import AgentMemoryView, MemoryMessage, MemoryScope
from mysteryverse_ai.schemas.generated.models import AgentRequest
from mysteryverse_ai.tools.mock import register_mock_tools
from mysteryverse_ai.tools.registry import ToolRegistry


def _request() -> AgentRequest:
    return AgentRequest.model_validate(
        {
            "run_id": "01J9Z3K4M7Q2R8T5V6W1XE0000",
            "request_id": "request",
            "actor_type": "dm",
            "actor_id": "dm:1",
            "allowed_entity_ids": [],
            "requested_topics": [],
            "task_type": "demo",
            "task_goal": "Summarize the authorized context.",
            "context": {
                "task_context": "x" * 2000,
                "recent_messages": [
                    {
                        "role": "user",
                        "content": "public message",
                        "sensitivity": "D2",
                    },
                    {
                        "role": "user",
                        "content": "private role book",
                        "sensitivity": "D3",
                    },
                ],
                "history_summary": "summary",
                "user_preferences": {"language": "zh-CN"},
            },
            "allowed_tools": ["mock_context_lookup"],
            "allowed_permissions": ["context:read"],
            "execution_limits": {
                "timeout_ms": 1000,
                "max_steps": 4,
                "max_context_tokens": 128,
            },
            "schema_version": 1,
            "budget_ms": 1000,
        }
    )


def test_context_builder_respects_budget_and_hashes_sections() -> None:
    tools = ToolRegistry()
    register_mock_tools(tools)
    context = ContextBuilder().build(
        request=_request(),
        memory=AgentMemoryView(),
        max_tokens=128,
        tools=tools.describe({"mock_context_lookup"}),
        output_schema=DemoAgentOutput,
    )
    assert context.estimated_tokens <= 128
    assert any(section.truncated for section in context.sections)
    assert all(len(section.sha256) == 64 for section in context.sections)
    assert context.restricted_messages_omitted == 1

    allowed = ContextBuilder().build(
        request=_request(),
        memory=AgentMemoryView(),
        max_tokens=512,
        tools=tools.describe({"mock_context_lookup"}),
        output_schema=DemoAgentOutput,
        allow_restricted_sensitivity=True,
    )
    assert allowed.restricted_messages_omitted == 0


def test_in_memory_memory_store_is_bounded() -> None:
    async def scenario() -> None:
        store = InMemoryMemoryStore()
        scope = MemoryScope(actor_type="dm", actor_id="dm:1", run_id="run")
        await store.append_short_term(
            scope,
            [
                MemoryMessage(role="user", content="one"),
                MemoryMessage(role="assistant", content="two"),
                MemoryMessage(role="user", content="three"),
            ],
            max_messages=2,
        )
        view = await store.load(scope, max_messages=2)
        assert [message.content for message in view.recent_messages] == ["two", "three"]

    asyncio.run(scenario())
