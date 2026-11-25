from __future__ import annotations

from copy import deepcopy
from pathlib import Path

from scripts.schema_registry import (
    build_registry,
    check_baseline_compatibility,
    check_compatibility,
    load_json,
    validate_registry,
)

ROOT = Path(__file__).resolve().parents[3]
SCHEMA_PATH = ROOT / "packages/contracts/json-schema/mysteryverse.schema.json"
GOVERNANCE_PATH = ROOT / "packages/contracts/registry/schema-governance.json"
REGISTRY_PATH = ROOT / "packages/contracts/registry/schema-registry.json"
BASELINE_PATH = ROOT / "packages/contracts/compatibility/schema-baseline-v1.json"


def test_registry_covers_every_contract_definition() -> None:
    schema = load_json(SCHEMA_PATH)
    governance = load_json(GOVERNANCE_PATH)
    registry = build_registry(schema, governance)

    validate_registry(schema, registry)
    assert registry == load_json(REGISTRY_PATH)
    assert registry["definitions_count"] == len(schema["$defs"])
    assert all(item["version_history"] for item in registry["schemas"])


def test_active_runtime_binding_is_explicit_and_versioned() -> None:
    registry = load_json(REGISTRY_PATH)
    binding = registry["runtime_bindings"]["demo_result"]
    record = next(
        item for item in registry["schemas"] if item["name"] == binding["definition"]
    )

    assert binding == {
        "task_type": "demo",
        "definition": "DemoAgentOutput",
        "schema_version": 1,
        "status": "active",
    }
    assert record["status"] == "active"
    assert record["active_version"] == "1.0.0"
    assert record["decision_status"] == "accepted"


def test_compatibility_detects_required_field_and_enum_removal() -> None:
    previous = {
        "type": "object",
        "additionalProperties": False,
        "properties": {
            "mode": {"type": "string", "enum": ["safe", "legacy"]},
            "answer": {"type": "string"},
        },
        "required": ["mode", "answer"],
    }
    current = {
        "type": "object",
        "additionalProperties": False,
        "properties": {
            "mode": {"type": "string", "enum": ["safe"]},
            "answer": {"type": "string"},
            "new_required": {"type": "boolean"},
        },
        "required": ["mode", "answer", "new_required"],
    }

    findings = check_compatibility(previous, current)
    errors = [item for item in findings if item.severity == "error"]
    warnings = [item for item in findings if item.severity == "warning"]

    assert any("enum values were removed" in item.message for item in errors)
    assert any("new required property" in item.message for item in errors)
    assert any("properties became required" in item.message for item in errors)
    assert warnings == []


def test_compatible_optional_addition_is_warning_only() -> None:
    previous = {
        "type": "object",
        "properties": {"answer": {"type": "string"}},
        "required": ["answer"],
    }
    current = deepcopy(previous)
    current["properties"]["confidence"] = {"type": "number"}

    findings = check_compatibility(previous, current)
    assert findings
    assert all(item.severity == "warning" for item in findings)


def test_current_schema_is_compatible_with_persisted_baseline() -> None:
    schema = load_json(SCHEMA_PATH)
    baseline = load_json(BASELINE_PATH)

    findings = check_baseline_compatibility(baseline, schema)
    assert not [item for item in findings if item.severity == "error"]
