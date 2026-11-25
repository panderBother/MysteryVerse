from __future__ import annotations

import hashlib
import hmac
import json
import time

from fastapi.testclient import TestClient
from pydantic import SecretStr

from mysteryverse_ai.api.request_integrity import canonical_request
from mysteryverse_ai.config import Settings
from mysteryverse_ai.main import create_app


def _settings() -> Settings:
    return Settings.for_test().model_copy(
        update={
            "environment": "test",
            "log_level": "ERROR",
            "internal_service_token": SecretStr("test-service-token"),
        }
    )


def _payload() -> dict[str, object]:
    return {
        "run_id": "01J9Z3K4M7Q2R8T5V6W1XE0000",
        "request_id": "http-request",
        "trace_id": "http-trace",
        "actor_type": "dm",
        "actor_id": "dm:1",
        "allowed_entity_ids": [],
        "requested_topics": ["demo"],
        "task_type": "demo",
        "task_goal": "Summarize the authorized demo context.",
        "context": {
            "task_context": "Use only supplied context.",
            "recent_messages": [],
            "history_summary": None,
            "user_preferences": {"language": "zh-CN"},
        },
        "allowed_tools": ["mock_context_lookup"],
        "allowed_permissions": ["context:read"],
        "execution_limits": {
            "timeout_ms": 2000,
            "max_steps": 4,
            "max_context_tokens": 1000,
        },
        "schema_version": 1,
        "budget_ms": 2000,
    }


def _headers() -> dict[str, str]:
    return {
        "X-Service-Id": "go-core",
        "X-Service-Token": "test-service-token",
        "X-Request-Id": "http-request",
        "X-Trace-Id": "http-trace",
        "Idempotency-Key": "http-idempotency-key",
    }


def _signed_headers(
    *,
    body: bytes,
    key: bytes = b"test-signing-key",
    timestamp: int | None = None,
    nonce: str = "nonce-12345678",
) -> dict[str, str]:
    resolved_timestamp = timestamp or int(time.time())
    timestamp_text = str(resolved_timestamp)
    canonical = canonical_request(
        timestamp=timestamp_text,
        nonce=nonce,
        service_id="go-core",
        method="POST",
        path="/internal/v1/ai/agent-runs",
        query="",
        body=body,
    )
    signature = hmac.new(key, canonical.encode(), hashlib.sha256).hexdigest()
    return {
        **_headers(),
        "Content-Type": "application/json",
        "X-Request-Timestamp": timestamp_text,
        "X-Request-Nonce": nonce,
        "X-Request-Signature": signature,
    }


def _signed_settings() -> Settings:
    return _settings().model_copy(
        update={
            "internal_request_signing_key": SecretStr("test-signing-key"),
            "internal_request_signing_previous_key": SecretStr("previous-signing-key"),
        }
    )


def test_internal_agent_endpoint_runs_and_is_idempotent() -> None:
    app = create_app(_settings())
    with TestClient(app) as client:
        first = client.post(
            "/internal/v1/ai/agent-runs",
            headers=_headers(),
            json=_payload(),
        )
        second = client.post(
            "/internal/v1/ai/agent-runs",
            headers=_headers(),
            json=_payload(),
        )
    assert first.status_code == 200
    assert first.json()["status"] == "succeeded"
    assert first.json()["trace_id"] == "http-trace"
    assert first.json()["tool_calls"]
    assert second.status_code == 200
    assert second.json() == first.json()


def test_internal_agent_endpoint_rejects_bad_service_identity() -> None:
    app = create_app(_settings())
    with TestClient(app) as client:
        response = client.post(
            "/internal/v1/ai/agent-runs",
            headers={**_headers(), "X-Service-Token": "wrong"},
            json=_payload(),
        )
    assert response.status_code == 403
    assert response.json()["error"]["code"] == "FORBIDDEN_ACTION"


def test_internal_agent_endpoint_uses_validation_error_envelope() -> None:
    app = create_app(_settings())
    with TestClient(app) as client:
        response = client.post(
            "/internal/v1/ai/agent-runs",
            headers=_headers(),
            json={"run_id": "not-a-ulid"},
        )
    assert response.status_code == 422
    assert response.json()["error"]["code"] == "VALIDATION_ERROR"
    assert response.json()["error"]["trace_id"] == "http-trace"
    assert "input" not in response.json()["error"]["details"][0]


def test_signed_internal_request_and_tamper_detection() -> None:
    body = json.dumps(_payload(), separators=(",", ":")).encode()
    app = create_app(_signed_settings())
    with TestClient(app) as client:
        success = client.post(
            "/internal/v1/ai/agent-runs",
            headers=_signed_headers(body=body),
            content=body,
        )
        tampered = client.post(
            "/internal/v1/ai/agent-runs",
            headers=_signed_headers(body=body, nonce="nonce-tamper01"),
            content=body + b" ",
        )
    assert success.status_code == 200
    assert tampered.status_code == 401
    assert tampered.json()["error"]["code"] == "REQUEST_SIGNATURE_INVALID"


def test_signed_request_replay_and_timestamp_window() -> None:
    body = json.dumps(_payload(), separators=(",", ":")).encode()
    app = create_app(_signed_settings())
    headers = _signed_headers(body=body, nonce="nonce-replay01")
    with TestClient(app) as client:
        first = client.post(
            "/internal/v1/ai/agent-runs",
            headers=headers,
            content=body,
        )
        replay = client.post(
            "/internal/v1/ai/agent-runs",
            headers=headers,
            content=body,
        )
        expired = client.post(
            "/internal/v1/ai/agent-runs",
            headers=_signed_headers(
                body=body,
                timestamp=int(time.time()) - 1000,
                nonce="nonce-expired01",
            ),
            content=body,
        )
    assert first.status_code == 200
    assert replay.status_code == 401
    assert replay.json()["error"]["code"] == "REQUEST_REPLAYED"
    assert expired.status_code == 401
    assert expired.json()["error"]["code"] == "REQUEST_TIMESTAMP_INVALID"


def test_previous_signing_key_supports_rotation_window() -> None:
    body = json.dumps(_payload(), separators=(",", ":")).encode()
    app = create_app(_signed_settings())
    with TestClient(app) as client:
        response = client.post(
            "/internal/v1/ai/agent-runs",
            headers=_signed_headers(
                body=body,
                key=b"previous-signing-key",
                nonce="nonce-rotation1",
            ),
            content=body,
        )
    assert response.status_code == 200
