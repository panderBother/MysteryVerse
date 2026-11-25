from __future__ import annotations

from mysteryverse_ai.observability.redaction import redact_value


def test_sensitive_fields_and_content_are_redacted() -> None:
    value = {
        "api_key": "secret",
        "authorization": "Bearer token",
        "nested": {
            "private_script": "do not log",
            "run_id": "run-1",
        },
        "prompt": "full prompt",
    }
    redacted = redact_value(value)
    assert redacted["api_key"] == "[REDACTED]"
    assert redacted["authorization"] == "[REDACTED]"
    assert redacted["nested"]["private_script"] == "[REDACTED]"
    assert redacted["nested"]["run_id"] == "run-1"
    assert redacted["prompt"] == "[REDACTED]"
