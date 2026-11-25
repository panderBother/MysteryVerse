from __future__ import annotations

from contextlib import suppress
from dataclasses import dataclass
from typing import ClassVar, Literal

from pydantic import BaseModel, ConfigDict, Field, ValidationError, model_validator

from mysteryverse_ai.schemas.generated.active_output_schemas import (
    ACTIVE_OUTPUT_SCHEMAS,
)
from mysteryverse_ai.schemas.generated.models import (
    AgentOutputValidationIssue,
    AgentTaskType,
    DemoAgentOutput,
)
from mysteryverse_ai.schemas.registry import ContractsSchemaRegistry, SchemaRegistryError


class ModelAction(BaseModel):
    model_config = ConfigDict(extra="forbid")

    action: Literal["tool", "final"]
    tool_name: str | None = None
    arguments: dict[str, object] = Field(default_factory=dict)
    output: dict[str, object] | None = None

    @model_validator(mode="after")
    def validate_action_payload(self) -> ModelAction:
        if self.action == "tool" and not self.tool_name:
            raise ValueError("tool_name is required for tool actions")
        if self.action == "final" and self.output is None:
            raise ValueError("output is required for final actions")
        return self


@dataclass(frozen=True, slots=True)
class OutputSchemaRegistration:
    task_type: AgentTaskType
    key: str
    version: int
    schema: type[BaseModel]
    schema_definition: str
    schema_status: Literal["active", "experimental"]

    @property
    def display_name(self) -> str:
        return f"{self.key}@{self.version}"


class OutputSchemaSelectionError(Exception):
    def __init__(self, *, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code
        self.message = message


class OutputSchemaRegistry:
    _GENERATED_MODELS: ClassVar[dict[str, type[BaseModel]]] = {
        "DemoAgentOutput": DemoAgentOutput,
    }

    def __init__(
        self,
        *,
        contracts_registry: ContractsSchemaRegistry | None = None,
        allow_unregistered: bool = False,
    ) -> None:
        self._registrations: dict[AgentTaskType, list[OutputSchemaRegistration]] = {}
        self._contracts_registry = contracts_registry or ContractsSchemaRegistry.load_default()
        self._allow_unregistered = allow_unregistered
        for binding in ACTIVE_OUTPUT_SCHEMAS:
            schema = self._GENERATED_MODELS.get(binding.definition)
            if schema is None:
                raise SchemaRegistryError(
                    code="SCHEMA_MODEL_NOT_GENERATED",
                    message="An active output schema has no generated model.",
                )
            self.register(
                task_type=AgentTaskType(binding.task_type),
                key=binding.schema_key,
                version=binding.schema_version,
                schema=schema,
                schema_definition=binding.definition,
            )

    def register(
        self,
        *,
        task_type: AgentTaskType,
        key: str,
        version: int,
        schema: type[BaseModel],
        schema_definition: str,
        schema_status: Literal["active", "experimental"] = "active",
    ) -> None:
        if not key or version < 1:
            raise ValueError("output schema key and positive version are required")
        if not schema_definition:
            raise ValueError("output schema definition is required")
        if schema_status == "active" and not self._allow_unregistered:
            self._contracts_registry.active(schema_definition)
        elif schema_status == "active" and self._allow_unregistered:
            with suppress(SchemaRegistryError):
                self._contracts_registry.active(schema_definition)
        registrations = self._registrations.setdefault(task_type, [])
        if any(item.key == key and item.version == version for item in registrations):
            raise ValueError(f"output schema already registered: {key}@{version}")
        registrations.append(
            OutputSchemaRegistration(
                task_type=task_type,
                key=key,
                version=version,
                schema=schema,
                schema_definition=schema_definition,
                schema_status=schema_status,
            )
        )
        registrations.sort(key=lambda item: item.version, reverse=True)

    def resolve(
        self,
        task_type: AgentTaskType,
        *,
        requested_key: str | None = None,
        requested_version: int | None = None,
    ) -> OutputSchemaRegistration:
        registrations = self._registrations.get(task_type, [])
        if not registrations:
            raise OutputSchemaSelectionError(
                code="OUTPUT_SCHEMA_NOT_FOUND",
                message=f"no output schema registered for task type {task_type.value}",
            )
        candidates = registrations
        if requested_key is not None:
            candidates = [item for item in candidates if item.key == requested_key]
        if not candidates:
            raise OutputSchemaSelectionError(
                code="OUTPUT_SCHEMA_NOT_FOUND",
                message="the requested output schema key is not registered for this task",
            )
        if requested_version is not None:
            candidates = [item for item in candidates if item.version == requested_version]
            if not candidates:
                raise OutputSchemaSelectionError(
                    code="OUTPUT_SCHEMA_VERSION_UNSUPPORTED",
                    message="the requested output schema version is not supported",
                )
        candidates = [item for item in candidates if item.schema_status == "active"]
        if not candidates:
            raise OutputSchemaSelectionError(
                code="OUTPUT_SCHEMA_NOT_ACTIVE",
                message="the requested output schema is not active",
            )
        return candidates[0]


def validation_issues(exc: ValidationError) -> list[AgentOutputValidationIssue]:
    issues: list[AgentOutputValidationIssue] = []
    for error in exc.errors(include_input=False, include_url=False):
        path = ".".join(str(item) for item in error.get("loc", []))
        issues.append(
            AgentOutputValidationIssue(
                path=path,
                message=str(error.get("msg", "Invalid output value")),
                type=str(error.get("type", "validation_error")),
            )
        )
    return issues
