# MysteryVerse Contracts

`packages/contracts` is the single authoritative source for cross-service DTOs,
WebSocket messages, game events, AI internal payloads and script structures.

## Source Of Truth

The only schema source is:

```text
json-schema/mysteryverse.schema.json
```

It uses JSON Schema Draft 2020-12. JSON examples, generated types, Go validation
fixtures and Python models must be regenerated from this file.

Do not create a second authoritative OpenAPI model tree. OpenAPI operation
definitions will reference or generate component schemas from this source.

## Contract Inventory

| Area | Definitions |
| --- | --- |
| IDs | `UserID`, `RoomID`, `GameID`, `PlayerID`, `ScriptID`, `EventID`, `MemoryID` and supporting ULID IDs |
| API | `ApiSuccessResponse`, `ApiErrorResponse`, `CursorPage`, `ParameterValidationError`, `BusinessErrorCode` |
| Realtime | `WebSocketEnvelope`, `WebSocketStream` |
| Events | `GameEvent` and the 11 MVP event payload types |
| AI | `AgentRequest`, `AgentResponse`, `AgentExecutionContext`, `AgentExecutionLimits`, `AgentTaskType`, `AgentExecutionStatus`, `AgentTraceStep`, `AgentToolCall`, `PlayerContext`, `MemoryCandidate`, `ScriptGenerationRequest`, `ScriptGenerationResult`, `DemoAgentOutput`, `AIError` |
| Script | `Script`, `Chapter`, `Character`, `Clue`, `Timeline`, `Relationship`, `Ending`, `Rule` |
| Script Management | `Stage`, `ScriptAsset`, `ScriptVersion`, `ScriptCatalogItem`, `ScriptEditorContent`, `ScriptValidationIssue`, `ScriptValidationReport`, create/update/validate/publish requests |

## Generated Outputs

```text
apps/web/src/generated/contracts.ts
backend/internal/contracts/contracts_gen.go
ai_service/src/mysteryverse_ai/schemas/generated/models.py
ai_service/src/mysteryverse_ai/schemas/generated/active_output_schemas.py
ai_service/src/mysteryverse_ai/schemas/generated/schema-registry.json
compatibility/schema-fingerprint.json
compatibility/schema-baseline-v1.json
registry/schema-registry.json
```

Generated files must not be edited by hand.

## Generators

| Language | Generator | Version |
| --- | --- | --- |
| TypeScript | `json-schema-to-typescript` | `16.0.0` |
| Go | `github.com/atombender/go-jsonschema` | `v0.22.0` |
| Python | `datamodel-code-generator` | `0.99.0` |

Python output uses Pydantic v2 `BaseModel` and constrained `RootModel` ID types.

## Commands

Generate all outputs:

```powershell
./scripts/contracts.ps1 generate
```

Validate schemas and examples:

```powershell
./scripts/contracts.ps1 validate
```

Run drift detection and contract tests:

```powershell
./scripts/contracts.ps1 check
```

Root command equivalents:

```powershell
./scripts/mv.ps1 contracts-generate
./scripts/mv.ps1 contracts-check
```

## Schema Registry

`registry/schema-governance.json` contains version, Active, lifecycle, compatibility
and runtime binding policy. The generator reads the policy and the authoritative
JSON Schema definitions to produce `registry/schema-registry.json`.

Runtime code only binds schemas marked `active`. The current formal output binding
is `demo_result -> DemoAgentOutput@1`. Proposed business schemas remain
`experimental` and cannot be treated as trusted Agent output.

Detailed policy:

```text
../docs/architecture/contracts-registry.md
```

## Change Process

1. Update `json-schema/mysteryverse.schema.json`.
2. Update `registry/schema-governance.json` when version, status or runtime
   binding changes.
3. Add or update a fixture in `examples/` and register it in
   `examples/manifest.json`.
4. Run `./scripts/contracts.ps1 generate`.
5. Run `./scripts/contracts.ps1 check`.
6. Review generated code, compatibility findings, Registry and fingerprint diffs.
7. Do not manually patch generated models or Registry artifacts.

## Compatibility

- Additive optional fields are compatible minor changes.
- Removing or renaming fields, changing semantics or changing required fields
  requires a major contract version.
- Enum additions are allowed only when clients explicitly ignore unknown values.
- WebSocket payload changes require a new schema version and compatible
  `mvws.v1` behavior.
- ID values remain pure 26-character ULIDs.
- Breaking changes fail against `compatibility/schema-baseline-v1.json` until a
  reviewed baseline or migration path is approved.

## Decision Gate

ADR-003, ADR-009 and ADR-010 are `Accepted`.

DQ-18 Contracts Registry and DQ-19 internal service security are `Approved`.
WebSocket, event versioning, Memory, PlayerContext, script editor, stage DSL,
ending conditions and voting policy contracts still depend on `Proposed` ADRs.
They are generated and tested as candidates, but they must not be presented as
approved production contracts. The Task 7 APIs actively reject unsupported
non-empty condition DSL and public publish; the Agent Runtime only enables the
`demo` output schema and one formal Active binding.

The game-scoped `game_id` rule uses the schema extension
`x-envelope-rules`; the contract validator and language tests enforce this rule
in addition to the generated optional field.
