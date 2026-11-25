from __future__ import annotations

import json
from importlib.resources import files
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field


class SchemaRegistryError(Exception):
    def __init__(self, *, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code
        self.message = message


class SchemaVersionHistoryEntry(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    version: str
    status: Literal["draft", "experimental", "active", "deprecated", "retired"]
    compatibility: Literal["none", "backward", "forward", "full"]
    created_at: str


class SchemaRecord(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    schema_id: str
    name: str
    definition: str
    version: str
    schema_type: Literal["public", "internal", "runtime", "persistence", "experimental"]
    status: Literal["draft", "experimental", "active", "deprecated", "retired"]
    decision_status: Literal["accepted", "proposed", "pending"]
    compatibility: Literal["none", "backward", "forward", "full"]
    active: bool
    active_version: str | None
    owner: str
    created_at: str
    deprecated_at: str | None
    sunset_at: str | None
    definition_sha256: str
    version_history: list[SchemaVersionHistoryEntry]


class RuntimeSchemaBinding(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    task_type: str = Field(min_length=1)
    definition: str = Field(min_length=1)
    schema_version: int = Field(ge=1)
    status: Literal["active"]


class SchemaRegistryDocument(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    registry_version: str
    generated_at: str
    source: str
    source_contract_version: str
    version_policy: dict[str, Any]
    active_policy: dict[str, Any]
    lifecycle_policy: dict[str, Any]
    replay_policy: dict[str, Any]
    definitions_count: int = Field(ge=1)
    schemas: list[SchemaRecord]
    runtime_bindings: dict[str, RuntimeSchemaBinding]


class ContractsSchemaRegistry:
    def __init__(self, document: SchemaRegistryDocument) -> None:
        self._document = document
        self._by_name = {item.name: item for item in document.schemas}
        self._by_id = {item.schema_id: item for item in document.schemas}
        if len(self._by_name) != len(document.schemas):
            raise SchemaRegistryError(
                code="SCHEMA_REGISTRY_INVALID",
                message="Schema registry contains duplicate names.",
            )
        if len(self._by_id) != len(document.schemas):
            raise SchemaRegistryError(
                code="SCHEMA_REGISTRY_INVALID",
                message="Schema registry contains duplicate IDs.",
            )

    @classmethod
    def load_default(cls) -> ContractsSchemaRegistry:
        payload = (
            files("mysteryverse_ai.schemas.generated")
            .joinpath("schema-registry.json")
            .read_text(encoding="utf-8")
        )
        return cls(SchemaRegistryDocument.model_validate(json.loads(payload)))

    @property
    def document(self) -> SchemaRegistryDocument:
        return self._document

    def by_name(self, name: str) -> SchemaRecord:
        record = self._by_name.get(name)
        if record is None:
            raise SchemaRegistryError(
                code="SCHEMA_NOT_REGISTERED",
                message="The requested schema is not registered.",
            )
        return record

    def active(self, name: str) -> SchemaRecord:
        record = self.by_name(name)
        if record.status != "active" or not record.active:
            raise SchemaRegistryError(
                code="SCHEMA_NOT_ACTIVE",
                message="The requested schema is not active.",
            )
        return record

    def runtime_binding(self, schema_key: str) -> RuntimeSchemaBinding:
        binding = self._document.runtime_bindings.get(schema_key)
        if binding is None:
            raise SchemaRegistryError(
                code="SCHEMA_BINDING_NOT_FOUND",
                message="The runtime schema binding is not registered.",
            )
        record = self.active(binding.definition)
        if record.active_version is None:
            raise SchemaRegistryError(
                code="SCHEMA_REGISTRY_INVALID",
                message="The active schema has no active version.",
            )
        return binding
