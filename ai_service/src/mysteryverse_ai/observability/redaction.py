from __future__ import annotations

import logging
from collections.abc import Mapping, Sequence
from typing import Any

_SENSITIVE_FRAGMENTS = {
    "api_key",
    "authorization",
    "cookie",
    "content",
    "password",
    "private",
    "prompt",
    "secret",
    "token",
}


def redact_value(value: Any, *, key: str = "", max_string: int = 256) -> Any:
    lowered = key.lower()
    if any(fragment in lowered for fragment in _SENSITIVE_FRAGMENTS):
        return "[REDACTED]"
    if isinstance(value, Mapping):
        return {
            str(item_key): redact_value(item_value, key=str(item_key), max_string=max_string)
            for item_key, item_value in value.items()
        }
    if isinstance(value, Sequence) and not isinstance(value, (str, bytes, bytearray)):
        return [redact_value(item, key=key, max_string=max_string) for item in value]
    if isinstance(value, str):
        return value if len(value) <= max_string else value[:max_string] + "...[TRUNCATED]"
    return value


def log_structured(
    logger: logging.Logger,
    message: str,
    **fields: Any,
) -> None:
    logger.info(message, extra={"structured": redact_value(fields)})
