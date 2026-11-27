from __future__ import annotations

from dataclasses import dataclass
from typing import Final


@dataclass(frozen=True, slots=True)
class GeneratedOutputSchemaBinding:
    task_type: str
    schema_key: str
    schema_version: int
    definition: str


ACTIVE_OUTPUT_SCHEMAS: Final[tuple[GeneratedOutputSchemaBinding, ...]] = (
    GeneratedOutputSchemaBinding(
        task_type="demo",
        schema_key="demo_result",
        schema_version=1,
        definition="DemoAgentOutput",
    ),
)
