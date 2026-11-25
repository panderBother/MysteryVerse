from __future__ import annotations

from pydantic import BaseModel, ConfigDict, Field

from mysteryverse_ai.schemas.generated.models import Sensitivity


class MemoryMessage(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    role: str
    content: str
    sensitivity: Sensitivity = Sensitivity.d0


class AgentMemoryView(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    recent_messages: list[MemoryMessage] = Field(default_factory=list)
    history_summary: str | None = None
    user_preferences: dict[str, object] = Field(default_factory=dict)


class MemoryScope(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    actor_type: str
    actor_id: str
    game_id: str | None = None
    run_id: str
