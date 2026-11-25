from __future__ import annotations

import hashlib
import json
import os
import shutil
import subprocess
import sys
from pathlib import Path

from schema_registry import load_json, write_generated_registry

CONTRACT_VERSION = "1.0.0"
GO_GENERATOR = "github.com/atombender/go-jsonschema@v0.22.0"


def run(command: list[str], *, cwd: Path) -> None:
    print(">", " ".join(command))
    environment = os.environ.copy()
    environment.pop("VIRTUAL_ENV", None)
    completed = subprocess.run(command, cwd=cwd, check=False, env=environment)
    if completed.returncode != 0:
        raise SystemExit(f"command failed with exit code {completed.returncode}")


def write_fingerprint(root: Path, schema_path: Path) -> None:
    content = schema_path.read_bytes()
    fingerprint = {
        "contract_version": CONTRACT_VERSION,
        "sha256": hashlib.sha256(content).hexdigest(),
        "source": "json-schema/mysteryverse.schema.json",
    }
    output = root / "packages/contracts/compatibility/schema-fingerprint.json"
    output.write_text(
        json.dumps(fingerprint, indent=2, ensure_ascii=True) + "\n",
        encoding="utf-8",
    )


def postprocess_go(output: Path) -> None:
    content = output.read_text(encoding="utf-8")
    if "func (t Timestamp) MarshalJSON()" in content:
        return

    if 'import "time"' in content:
        content = content.replace(
            'import "time"',
            'import (\n\t"encoding/json"\n\t"time"\n)',
            1,
        )
    elif '"encoding/json"' not in content:
        raise SystemExit(
            "unexpected Go generator imports; cannot add Timestamp methods"
        )

    content += """

// MarshalJSON preserves RFC 3339 timestamp serialization for the generated alias.
func (t Timestamp) MarshalJSON() ([]byte, error) {
\treturn json.Marshal(time.Time(t))
}

// UnmarshalJSON parses RFC 3339 timestamps into the generated alias.
func (t *Timestamp) UnmarshalJSON(value []byte) error {
\tvar parsed time.Time
\tif err := json.Unmarshal(value, &parsed); err != nil {
\t\treturn err
\t}
\t*t = Timestamp(parsed)
\treturn nil
}
"""
    output.write_text(content, encoding="utf-8")


def main() -> int:
    root = Path(__file__).resolve().parents[3]
    schema_path = root / "packages/contracts/json-schema/mysteryverse.schema.json"
    governance_path = root / "packages/contracts/registry/schema-governance.json"
    ts_output = root / "apps/web/src/generated/contracts.ts"
    go_output = root / "backend/internal/contracts/contracts_gen.go"
    py_output = root / "ai_service/src/mysteryverse_ai/schemas/generated/models.py"
    active_output_schemas = (
        root
        / "ai_service/src/mysteryverse_ai/schemas/generated/active_output_schemas.py"
    )

    for output in (ts_output, go_output, py_output):
        output.parent.mkdir(parents=True, exist_ok=True)

    npx = shutil.which("npx.cmd") or shutil.which("npx")
    if not npx:
        raise SystemExit("npx is required")

    schema_arg = "packages/contracts/json-schema/mysteryverse.schema.json"
    ts_arg = "apps/web/src/generated/contracts.ts"
    go_arg = "backend/internal/contracts/contracts_gen.go"
    py_arg = "ai_service/src/mysteryverse_ai/schemas/generated/models.py"

    run(
        [
            npx,
            "--no-install",
            "json2ts",
            "-i",
            schema_arg,
            "-o",
            ts_arg,
            "--unreachableDefinitions",
            "--additionalProperties=false",
        ],
        cwd=root,
    )

    run(
        [
            npx,
            "--no-install",
            "prettier",
            "--write",
            ts_arg,
        ],
        cwd=root,
    )

    run(
        [
            "go",
            "run",
            GO_GENERATOR,
            "--only-models",
            "--tags",
            "json",
            "--capitalization",
            "ID",
            "--struct-name-from-title",
            "--package",
            "contracts",
            "--output",
            go_arg,
            schema_arg,
        ],
        cwd=root,
    )

    run(
        [
            "uv",
            "run",
            "--project",
            "packages/contracts/tools",
            "datamodel-codegen",
            "--input",
            schema_arg,
            "--input-file-type",
            "jsonschema",
            "--output",
            py_arg,
            "--output-model-type",
            "pydantic_v2.BaseModel",
            "--target-python-version",
            "3.12",
            "--use-title-as-name",
            "--snake-case-field",
            "--field-constraints",
            "--use-annotated",
            "--use-standard-collections",
            "--disable-timestamp",
            "--formatters",
            "builtin",
        ],
        cwd=root,
    )

    postprocess_go(go_output)
    run(["gofmt", "-w", go_arg], cwd=root)

    run(
        [
            "uv",
            "run",
            "--project",
            "ai_service",
            "ruff",
            "format",
            py_arg,
        ],
        cwd=root,
    )

    schema = load_json(schema_path)
    governance = load_json(governance_path)
    write_generated_registry(root, schema, governance)
    run(
        [
            "uv",
            "run",
            "--project",
            "ai_service",
            "ruff",
            "format",
            str(active_output_schemas.relative_to(root)),
        ],
        cwd=root,
    )

    write_fingerprint(root, schema_path)
    print("contracts generated")
    return 0


if __name__ == "__main__":
    sys.exit(main())
