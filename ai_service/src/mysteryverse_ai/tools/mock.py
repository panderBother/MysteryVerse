from __future__ import annotations

from pydantic import BaseModel, ConfigDict, Field

from mysteryverse_ai.tools.models import ToolInvocationContext, ToolResult, ToolSpec
from mysteryverse_ai.tools.registry import ToolRegistry


class MockContextLookupArgs(BaseModel):
    model_config = ConfigDict(extra="forbid")

    topic: str = Field(min_length=1, max_length=128)


async def _lookup(
    arguments: dict[str, object],
    _context: ToolInvocationContext,
) -> ToolResult:
    topic = str(arguments["topic"])
    summary = f"Authorized demo context for topic: {topic}"
    return ToolResult(content={"summary": summary}, summary=summary)


def register_mock_tools(registry: ToolRegistry) -> None:
    registry.register(
        ToolSpec(
            name="mock_context_lookup",
            description="Return a bounded mock summary for an authorized demo topic.",
            args_model=MockContextLookupArgs,
            permissions=frozenset({"context:read"}),
            allowed_actor_types=frozenset(
                {"dm", "npc", "ai_player", "generator", "reviewer", "player_context"}
            ),
            timeout_ms=1_000,
        ),
        _lookup,
    )
