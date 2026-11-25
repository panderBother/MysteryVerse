from __future__ import annotations

import fnmatch
import hashlib
import json
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Any


class RegistryValidationError(ValueError):
    pass


@dataclass(frozen=True, slots=True)
class CompatibilityFinding:
    severity: str
    path: str
    message: str

    def as_dict(self) -> dict[str, str]:
        return {
            "severity": self.severity,
            "path": self.path,
            "message": self.message,
        }


def load_json(path: Path) -> Any:
    return json.loads(path.read_text(encoding="utf-8"))


def write_json(path: Path, payload: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        json.dumps(payload, indent=2, ensure_ascii=True) + "\n",
        encoding="utf-8",
    )


def definition_digest(definition: dict[str, Any]) -> str:
    canonical = json.dumps(
        definition,
        ensure_ascii=True,
        sort_keys=True,
        separators=(",", ":"),
    )
    return hashlib.sha256(canonical.encode()).hexdigest()


def _schema_type(name: str, *, active: bool, rules: list[dict[str, Any]]) -> str:
    for rule in rules:
        patterns = rule.get("patterns", [])
        if any(fnmatch.fnmatchcase(name, pattern) for pattern in patterns):
            return str(rule["schema_type"])
    return "public" if active else "experimental"


def _schema_id(schema_type: str, name: str) -> str:
    kebab = re.sub(r"(?<!^)(?=[A-Z])", "-", name).replace("_", "-").lower()
    return f"mv.{schema_type}.{kebab}"


def _owner(name: str, definition: dict[str, Any]) -> str:
    configured = definition.get("x-schema-owner")
    if isinstance(configured, str) and configured:
        return configured
    if fnmatch.fnmatchcase(name, "Agent*") or name in {
        "AIError",
        "DemoAgentOutput",
        "RunID",
        "StepType",
        "Status",
    }:
        return "ai-runtime"
    if fnmatch.fnmatchcase(name, "*Event*") or name in {
        "GameCreated",
        "GameEnded",
        "GamePaused",
        "GameResumed",
        "GameStarted",
        "PlayerJoined",
        "PlayerLeft",
        "PhaseChanged",
        "ClueDiscovered",
        "MessageCreated",
        "VoteSubmitted",
    }:
        return "go-game"
    if name == "MemoryCandidate" or name == "MemoryID":
        return "go-core"
    if fnmatch.fnmatchcase(name, "*Script*") or name in {
        "Chapter",
        "Character",
        "Clue",
        "Ending",
        "Relationship",
        "Rule",
        "Stage",
        "Timeline",
    }:
        return "go-content"
    return "contracts"


def _status(definition: dict[str, Any]) -> tuple[str, str]:
    declared = definition.get("x-contract-status")
    if declared == "accepted":
        return "active", "accepted"
    if declared == "proposed":
        return "experimental", "proposed"
    return "experimental", "pending"


def build_registry(
    schema: dict[str, Any],
    governance: dict[str, Any],
) -> dict[str, Any]:
    definitions = schema.get("$defs")
    if not isinstance(definitions, dict) or not definitions:
        raise RegistryValidationError("schema $defs must be a non-empty object")

    rules = governance.get("classification_rules")
    if not isinstance(rules, list):
        raise RegistryValidationError("classification_rules must be an array")

    records: list[dict[str, Any]] = []
    for name in sorted(definitions):
        definition = definitions[name]
        if not isinstance(definition, dict):
            raise RegistryValidationError(f"definition {name} must be an object")
        status, decision_status = _status(definition)
        active = status == "active"
        schema_type = _schema_type(name, active=active, rules=rules)
        version = str(governance["version_policy"]["initial_version"])
        compatibility = (
            governance["default_compatibility"]
            if active
            else governance["experimental_compatibility"]
        )
        records.append(
            {
                "schema_id": _schema_id(schema_type, name),
                "name": name,
                "definition": f"#/$defs/{name}",
                "version": version,
                "schema_type": schema_type,
                "status": status,
                "decision_status": decision_status,
                "compatibility": compatibility,
                "active": active,
                "active_version": version if active else None,
                "owner": _owner(name, definition),
                "created_at": governance["generated_at"],
                "deprecated_at": None,
                "sunset_at": None,
                "definition_sha256": definition_digest(definition),
                "version_history": [
                    {
                        "version": version,
                        "status": status,
                        "compatibility": compatibility,
                        "created_at": governance["generated_at"],
                    }
                ],
            }
        )

    names = {record["name"] for record in records}
    bindings = governance.get("runtime_bindings")
    if not isinstance(bindings, dict) or not bindings:
        raise RegistryValidationError("runtime_bindings must be a non-empty object")
    normalized_bindings: dict[str, dict[str, Any]] = {}
    for key, binding in sorted(bindings.items()):
        if not isinstance(binding, dict):
            raise RegistryValidationError(f"runtime binding {key} must be an object")
        definition_name = binding.get("definition")
        if definition_name not in names:
            raise RegistryValidationError(
                f"runtime binding {key} references unknown definition {definition_name}"
            )
        record = next(item for item in records if item["name"] == definition_name)
        if record["status"] != "active" or binding.get("status") != "active":
            raise RegistryValidationError(
                f"runtime binding {key} must reference an active schema"
            )
        schema_version = binding.get("schema_version")
        if not isinstance(schema_version, int) or schema_version < 1:
            raise RegistryValidationError(
                f"runtime binding {key} needs a positive schema_version"
            )
        normalized_bindings[key] = {
            "task_type": str(binding.get("task_type", "")),
            "definition": str(definition_name),
            "schema_version": schema_version,
            "status": "active",
        }

    return {
        "registry_version": governance["registry_version"],
        "generated_at": governance["generated_at"],
        "source": governance["source"],
        "source_contract_version": schema.get("x-contract-version"),
        "version_policy": governance["version_policy"],
        "active_policy": governance["active_policy"],
        "lifecycle_policy": governance["lifecycle_policy"],
        "replay_policy": governance["replay_policy"],
        "definitions_count": len(records),
        "schemas": records,
        "runtime_bindings": normalized_bindings,
    }


def validate_registry(
    schema: dict[str, Any],
    registry: dict[str, Any],
) -> None:
    definitions = schema.get("$defs")
    schemas = registry.get("schemas")
    if not isinstance(definitions, dict) or not isinstance(schemas, list):
        raise RegistryValidationError("registry schemas must be an array")
    names = [item.get("name") for item in schemas if isinstance(item, dict)]
    if len(names) != len(set(names)):
        raise RegistryValidationError("registry contains duplicate schema names")
    if set(names) != set(definitions):
        missing = sorted(set(definitions) - set(names))
        extra = sorted(set(names) - set(definitions))
        raise RegistryValidationError(
            f"registry definition drift; missing={missing}, extra={extra}"
        )
    if registry.get("definitions_count") != len(definitions):
        raise RegistryValidationError("registry definitions_count is stale")
    for item in schemas:
        if not isinstance(item, dict):
            raise RegistryValidationError("registry schema entry must be an object")
        name = item["name"]
        expected_digest = definition_digest(definitions[name])
        if item.get("definition_sha256") != expected_digest:
            raise RegistryValidationError(f"registry digest is stale for {name}")
        history = item.get("version_history")
        if not isinstance(history, list) or not history:
            raise RegistryValidationError(f"registry history is missing for {name}")
        if item.get("active") is True and item.get("active_version") is None:
            raise RegistryValidationError(f"active schema {name} lacks active_version")
    bindings = registry.get("runtime_bindings")
    if not isinstance(bindings, dict) or not bindings:
        raise RegistryValidationError("runtime_bindings must be a non-empty object")
    for key, binding in bindings.items():
        if not isinstance(binding, dict):
            raise RegistryValidationError(f"runtime binding {key} must be an object")
        if binding.get("definition") not in definitions:
            raise RegistryValidationError(
                f"runtime binding {key} references an unknown definition"
            )


def baseline_payload(
    schema: dict[str, Any],
    *,
    contract_version: str,
    created_at: str,
) -> dict[str, Any]:
    definitions = schema.get("$defs")
    if not isinstance(definitions, dict):
        raise RegistryValidationError("schema $defs must be an object")
    return {
        "contract_version": contract_version,
        "created_at": created_at,
        "source": "json-schema/mysteryverse.schema.json",
        "definitions": definitions,
    }


def write_baseline_if_missing(
    path: Path,
    schema: dict[str, Any],
    *,
    contract_version: str,
    created_at: str,
) -> None:
    if path.exists():
        return
    write_json(
        path,
        baseline_payload(
            schema,
            contract_version=contract_version,
            created_at=created_at,
        ),
    )


def _type_set(schema: dict[str, Any]) -> set[str]:
    declared = schema.get("type")
    if isinstance(declared, str):
        return {declared}
    if isinstance(declared, list):
        return {str(item) for item in declared}
    if "oneOf" in schema or "anyOf" in schema:
        return {"composed"}
    if "$ref" in schema:
        return {"ref"}
    return set()


def check_compatibility(
    previous: Any,
    current: Any,
    *,
    path: str = "$",
) -> list[CompatibilityFinding]:
    if not isinstance(previous, dict) or not isinstance(current, dict):
        if previous != current:
            return [
                CompatibilityFinding(
                    "error",
                    path,
                    "schema shape changed structurally",
                )
            ]
        return []

    findings: list[CompatibilityFinding] = []
    old_types = _type_set(previous)
    new_types = _type_set(current)
    if old_types != new_types:
        findings.append(
            CompatibilityFinding(
                "error",
                path,
                f"type changed from {sorted(old_types)} to {sorted(new_types)}",
            )
        )

    old_ref = previous.get("$ref")
    new_ref = current.get("$ref")
    if old_ref != new_ref:
        findings.append(
            CompatibilityFinding(
                "error",
                path,
                f"reference changed from {old_ref!r} to {new_ref!r}",
            )
        )

    old_enum = previous.get("enum")
    new_enum = current.get("enum")
    if isinstance(old_enum, list) or isinstance(new_enum, list):
        old_values = {json.dumps(item, sort_keys=True) for item in old_enum or []}
        new_values = {json.dumps(item, sort_keys=True) for item in new_enum or []}
        if old_values - new_values:
            findings.append(
                CompatibilityFinding(
                    "error",
                    path,
                    "enum values were removed",
                )
            )
        if new_values - old_values:
            findings.append(
                CompatibilityFinding(
                    "warning",
                    path,
                    "enum values were added; consumers must define unknown-value behavior",
                )
            )

    if previous.get("default") != current.get("default"):
        findings.append(
            CompatibilityFinding(
                "warning",
                path,
                "default value changed",
            )
        )

    old_properties = previous.get("properties")
    new_properties = current.get("properties")
    if isinstance(old_properties, dict) or isinstance(new_properties, dict):
        old_properties = old_properties if isinstance(old_properties, dict) else {}
        new_properties = new_properties if isinstance(new_properties, dict) else {}
        removed = set(old_properties) - set(new_properties)
        if removed:
            findings.append(
                CompatibilityFinding(
                    "error",
                    path,
                    f"properties were removed: {sorted(removed)}",
                )
            )
        old_required = set(previous.get("required", []))
        new_required = set(current.get("required", []))
        newly_required = new_required - old_required
        if newly_required:
            findings.append(
                CompatibilityFinding(
                    "error",
                    path,
                    f"properties became required: {sorted(newly_required)}",
                )
            )
        if old_required - new_required:
            findings.append(
                CompatibilityFinding(
                    "warning",
                    path,
                    "properties became optional",
                )
            )
        for name in sorted(set(old_properties) & set(new_properties)):
            findings.extend(
                check_compatibility(
                    old_properties[name],
                    new_properties[name],
                    path=f"{path}.properties.{name}",
                )
            )
        for name in sorted(set(new_properties) - set(old_properties)):
            if name in new_required:
                findings.append(
                    CompatibilityFinding(
                        "error",
                        path,
                        f"new required property was added: {name}",
                    )
                )
            else:
                findings.append(
                    CompatibilityFinding(
                        "warning",
                        path,
                        f"optional property was added: {name}",
                    )
                )

    old_items = previous.get("items")
    new_items = current.get("items")
    if isinstance(old_items, dict) and isinstance(new_items, dict):
        findings.extend(check_compatibility(old_items, new_items, path=f"{path}.items"))
    elif isinstance(old_items, dict) != isinstance(new_items, dict):
        findings.append(
            CompatibilityFinding(
                "error",
                path,
                "array item schema changed structurally",
            )
        )

    for keyword in ("oneOf", "anyOf", "allOf"):
        old_branch = previous.get(keyword)
        new_branch = current.get(keyword)
        if isinstance(old_branch, list) and isinstance(new_branch, list):
            if len(old_branch) != len(new_branch):
                findings.append(
                    CompatibilityFinding(
                        "error",
                        path,
                        f"{keyword} branch count changed",
                    )
                )
                continue
            for index, (old_item, new_item) in enumerate(
                zip(old_branch, new_branch, strict=True)
            ):
                findings.extend(
                    check_compatibility(
                        old_item,
                        new_item,
                        path=f"{path}.{keyword}[{index}]",
                    )
                )
        elif old_branch != new_branch:
            findings.append(
                CompatibilityFinding(
                    "error",
                    path,
                    f"{keyword} changed structurally",
                )
            )

    return findings


def check_baseline_compatibility(
    baseline: dict[str, Any],
    schema: dict[str, Any],
) -> list[CompatibilityFinding]:
    old_definitions = baseline.get("definitions")
    new_definitions = schema.get("$defs")
    if not isinstance(old_definitions, dict) or not isinstance(new_definitions, dict):
        raise RegistryValidationError("compatibility inputs must contain definitions")
    findings: list[CompatibilityFinding] = []
    removed = set(old_definitions) - set(new_definitions)
    if removed:
        findings.append(
            CompatibilityFinding(
                "error",
                "$defs",
                f"definitions were removed: {sorted(removed)}",
            )
        )
    for name in sorted(set(old_definitions) & set(new_definitions)):
        findings.extend(
            check_compatibility(
                old_definitions[name],
                new_definitions[name],
                path=f"$defs.{name}",
            )
        )
    return findings


def active_output_schema_module(registry: dict[str, Any]) -> str:
    bindings = registry["runtime_bindings"]
    lines = [
        "from __future__ import annotations",
        "",
        "from dataclasses import dataclass",
        "from typing import Final",
        "",
        "",
        "@dataclass(frozen=True, slots=True)",
        "class GeneratedOutputSchemaBinding:",
        "    task_type: str",
        "    schema_key: str",
        "    schema_version: int",
        "    definition: str",
        "",
        "",
        "ACTIVE_OUTPUT_SCHEMAS: Final[tuple[GeneratedOutputSchemaBinding, ...]] = (",
    ]
    for key, binding in sorted(bindings.items()):
        lines.extend(
            [
                "    GeneratedOutputSchemaBinding(",
                f'        task_type="{binding["task_type"]}",',
                f'        schema_key="{key}",',
                f"        schema_version={binding['schema_version']},",
                f'        definition="{binding["definition"]}",',
                "    ),",
            ]
        )
    lines.extend([")", ""])
    return "\n".join(lines)


def write_generated_registry(
    root: Path,
    schema: dict[str, Any],
    governance: dict[str, Any],
) -> dict[str, Any]:
    registry = build_registry(schema, governance)
    validate_registry(schema, registry)
    registry_path = root / "packages/contracts/registry/schema-registry.json"
    write_json(registry_path, registry)

    ai_registry_path = (
        root / "ai_service/src/mysteryverse_ai/schemas/generated/schema-registry.json"
    )
    write_json(ai_registry_path, registry)

    module_path = (
        root
        / "ai_service/src/mysteryverse_ai/schemas/generated/active_output_schemas.py"
    )
    module_path.parent.mkdir(parents=True, exist_ok=True)
    module_path.write_text(active_output_schema_module(registry), encoding="utf-8")

    write_baseline_if_missing(
        root / "packages/contracts/compatibility/schema-baseline-v1.json",
        schema,
        contract_version=str(schema["x-contract-version"]),
        created_at=str(governance["generated_at"]),
    )
    return registry
