from __future__ import annotations

import argparse
import hashlib
import json
import sys
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator

REQUIRED_DEFINITIONS = {
    "UserID",
    "RoomID",
    "GameID",
    "PlayerID",
    "ScriptID",
    "EventID",
    "MemoryID",
    "ApiSuccessResponse",
    "ApiErrorResponse",
    "CursorPage",
    "ParameterValidationError",
    "BusinessErrorCode",
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

ID_DEFINITIONS = {
    "UserID",
    "RoomID",
    "GameID",
    "PlayerID",
    "ScriptID",
    "EventID",
    "MemoryID",
}


def load_json(path: Path) -> Any:
    return json.loads(path.read_text(encoding="utf-8"))


def validate_schema(schema: dict[str, Any]) -> None:
    Draft202012Validator.check_schema(schema)
    definitions = schema.get("$defs", {})
    missing = REQUIRED_DEFINITIONS - definitions.keys()
    if missing:
        raise ValueError(f"missing contract definitions: {sorted(missing)}")

    for name in ID_DEFINITIONS:
        definition = definitions[name]
        if definition.get("type") != "string":
            raise ValueError(f"{name} must be a string")
        if definition.get("pattern") != "^[0-9A-HJKMNP-TV-Z]{26}$":
            raise ValueError(f"{name} does not enforce a 26-character ULID")

    event_types = definitions["GameEventType"]["enum"]
    expected_events = {
        "game.created",
        "room.player_joined",
        "room.player_left",
        "game.started",
        "game.phase_changed",
        "game.clue_unlocked",
        "game.message_received",
        "game.vote_submitted",
        "game.paused",
        "game.resumed",
        "game.ended",
    }
    if set(event_types) != expected_events:
        raise ValueError("GameEventType enum is incomplete")

    error_codes = definitions["BusinessErrorCode"]["enum"]
    if len(error_codes) != len(set(error_codes)):
        raise ValueError("BusinessErrorCode contains duplicate values")


def validate_examples(root: Path, schema: dict[str, Any]) -> None:
    manifest_path = root / "packages/contracts/examples/manifest.json"
    manifest = load_json(manifest_path)

    for example in manifest["examples"]:
        definition = example["definition"]
        if definition not in schema["$defs"]:
            raise ValueError(f"example references missing definition {definition}")

        validator_schema = {
            "$schema": schema["$schema"],
            "$ref": f"#/$defs/{definition}",
            "$defs": schema["$defs"],
        }
        validator = Draft202012Validator(validator_schema)
        document = load_json(root / "packages/contracts/examples" / example["file"])
        if definition == "WebSocketEnvelope":
            validate_envelope_rules(schema, document)

        errors = sorted(validator.iter_errors(document), key=lambda error: list(error.path))

        if errors:
            messages = "; ".join(error.message for error in errors)
            raise ValueError(f"{example['file']} failed {definition}: {messages}")


def validate_envelope_rules(schema: dict[str, Any], document: dict[str, Any]) -> None:
    rules = schema["$defs"]["WebSocketEnvelope"]["x-envelope-rules"]
    stream = document.get("stream")
    has_game_id = "game_id" in document

    if stream in rules["game_id_required_streams"] and not has_game_id:
        raise ValueError(f"stream {stream} requires game_id")

    if stream in rules["game_id_forbidden_streams"] and has_game_id:
        raise ValueError(f"stream {stream} must not contain game_id")


def verify_fingerprint(root: Path, schema_path: Path) -> None:
    fingerprint_path = root / "packages/contracts/compatibility/schema-fingerprint.json"
    expected = load_json(fingerprint_path)
    actual = {
        "contract_version": "1.0.0",
        "sha256": hashlib.sha256(schema_path.read_bytes()).hexdigest(),
        "source": "json-schema/mysteryverse.schema.json",
    }
    if expected != actual:
        raise ValueError(
            "schema fingerprint is stale; run packages/contracts/scripts/generate.py"
        )


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--skip-fingerprint", action="store_true")
    args = parser.parse_args()

    root = Path(__file__).resolve().parents[3]
    schema_path = root / "packages/contracts/json-schema/mysteryverse.schema.json"
    schema = load_json(schema_path)

    validate_schema(schema)
    validate_examples(root, schema)
    if not args.skip_fingerprint:
        verify_fingerprint(root, schema_path)

    print(
        "contracts validation passed:",
        f"{len(schema['$defs'])} definitions",
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
