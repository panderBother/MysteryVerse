from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import pytest
from jsonschema import Draft202012Validator  # type: ignore[import-untyped]
from pydantic import ValidationError

from mysteryverse_ai.schemas.generated import models

ROOT = Path(__file__).resolve().parents[2]
SCHEMA_PATH = ROOT / "packages/contracts/json-schema/mysteryverse.schema.json"
EXAMPLES_PATH = ROOT / "packages/contracts/examples"


def _load_json(path: Path) -> Any:
    return json.loads(path.read_text(encoding="utf-8"))


def _manifest() -> list[dict[str, str]]:
    return _load_json(EXAMPLES_PATH / "manifest.json")["examples"]


def _validate_envelope_rules(schema: dict[str, Any], document: dict[str, Any]) -> None:
    rules = schema["$defs"]["WebSocketEnvelope"]["x-envelope-rules"]
    stream = document["stream"]

    if stream in rules["game_id_required_streams"]:
        assert "game_id" in document

    if stream in rules["game_id_forbidden_streams"]:
        assert "game_id" not in document


def test_generated_models_cover_required_contracts() -> None:
    required = {
        "ApiSuccessResponse",
        "ApiErrorResponse",
        "CursorPage",
        "ParameterValidationError",
        "WebSocketEnvelope",
        "GameCreated",
        "PlayerJoined",
        "PlayerLeft",
        "GameStarted",
        "PhaseChanged",
        "ClueDiscovered",
        "MessageCreated",
        "VoteSubmitted",
        "GamePaused",
        "GameResumed",
        "GameEnded",
        "AgentRequest",
        "AgentResponse",
        "AgentTaskType",
        "AgentExecutionContext",
        "AgentExecutionLimits",
        "AgentExecutionStatus",
        "AgentToolCall",
        "AgentTraceStep",
        "AgentOutputValidationIssue",
        "PlayerContext",
        "MemoryCandidate",
        "ScriptGenerationRequest",
        "ScriptGenerationResult",
        "AIError",
        "Script",
        "Chapter",
        "Character",
        "Clue",
        "Timeline",
        "Relationship",
        "Ending",
        "Rule",
        "Stage",
        "ScriptAsset",
        "ScriptVersion",
        "ScriptValidationIssue",
        "ScriptValidationReport",
        "ScriptEditorContent",
    }

    assert required <= set(vars(models))


def test_all_examples_validate_and_deserialize() -> None:
    schema = _load_json(SCHEMA_PATH)
    Draft202012Validator.check_schema(schema)

    for item in _manifest():
        definition = item["definition"]
        document = _load_json(EXAMPLES_PATH / item["file"])
        validator = Draft202012Validator(
            {
                "$schema": schema["$schema"],
                "$ref": f"#/$defs/{definition}",
                "$defs": schema["$defs"],
            }
        )

        validator.validate(document)
        if definition == "WebSocketEnvelope":
            _validate_envelope_rules(schema, document)

        model_type = getattr(models, definition)
        model = model_type.model_validate(document)
        serialized = model.model_dump(mode="json", exclude_unset=True)
        validator.validate(serialized)


def test_enum_values_match_schema() -> None:
    schema = _load_json(SCHEMA_PATH)
    expected_events = schema["$defs"]["GameEventType"]["enum"]
    expected_errors = schema["$defs"]["BusinessErrorCode"]["enum"]

    assert {item.value for item in models.GameEventType} == set(expected_events)
    assert {item.value for item in models.BusinessErrorCode} == set(expected_errors)


def test_nullability_and_ulid_validation() -> None:
    phase_changed = next(item for item in _manifest() if item["definition"] == "PhaseChanged")
    phase = models.PhaseChanged.model_validate(_load_json(EXAMPLES_PATH / phase_changed["file"]))
    assert phase.deadline_at is None

    with pytest.raises(ValidationError):
        models.UserID.model_validate("too-short")


def test_websocket_envelope_scope_rules() -> None:
    schema = _load_json(SCHEMA_PATH)
    game_envelope = _load_json(EXAMPLES_PATH / "ws-game-envelope.json")
    room_envelope = _load_json(EXAMPLES_PATH / "ws-room-envelope.json")

    missing_game_id = game_envelope.copy()
    missing_game_id.pop("game_id")
    with pytest.raises(AssertionError):
        _validate_envelope_rules(schema, missing_game_id)

    fake_game_id = room_envelope.copy()
    fake_game_id["game_id"] = game_envelope["game_id"]
    with pytest.raises(AssertionError):
        _validate_envelope_rules(schema, fake_game_id)
