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
from mysteryverse_ai.security.keys import KeyMaterial, KeyProvider, KeyProviderError
from mysteryverse_ai.security.nonce import InMemoryNonceStore, NonceStore

RUN_ID = "01J9Z3K4M7Q2R8T5V6W1XE0000"
ACTIVE_SIGNING_KEY = b"test-signing-key-material-32-bytes"
PREVIOUS_SIGNING_KEY = b"previous-signing-key-material-32b"


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
        "run_id": RUN_ID,
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
        "X-Service-Scope": f"ai:agent:run,run:{RUN_ID}",
        "X-Request-Id": "http-request",
        "X-Trace-Id": "http-trace",
        "Idempotency-Key": "http-idempotency-key",
    }


def _signed_headers(
    *,
    body: bytes,
    key: bytes = ACTIVE_SIGNING_KEY,
    key_id: str = "go-core-v1",
    timestamp: int | None = None,
    nonce: str = "nonce-12345678",
) -> dict[str, str]:
    resolved_timestamp = timestamp or int(time.time())
    timestamp_text = str(resolved_timestamp)
    canonical = canonical_request(
        signature_version="mv-hmac-sha256-v1",
        key_id=key_id,
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
        "X-Request-Signature-Version": "mv-hmac-sha256-v1",
        "X-Service-Key-Id": key_id,
        "X-Request-Timestamp": timestamp_text,
        "X-Request-Nonce": nonce,
        "X-Request-Signature": signature,
    }


def _signed_settings() -> Settings:
    return _settings().model_copy(
        update={
            "internal_signing_key_id": "go-core-v1",
            "internal_request_signing_key": SecretStr(ACTIVE_SIGNING_KEY.decode()),
            "internal_request_signing_previous_key_id": "go-core-previous",
            "internal_request_signing_previous_key": SecretStr(PREVIOUS_SIGNING_KEY.decode()),
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
                key=PREVIOUS_SIGNING_KEY,
                key_id="go-core-previous",
                nonce="nonce-rotation1",
            ),
            content=body,
        )
    assert response.status_code == 200


def test_internal_service_scope_and_tenant_authorization() -> None:
    app = create_app(_settings())
    with TestClient(app) as client:
        missing_resource_scope = client.post(
            "/internal/v1/ai/agent-runs",
            headers={**_headers(), "X-Service-Scope": "ai:agent:run"},
            json=_payload(),
        )
        denied_tenant = client.post(
            "/internal/v1/ai/agent-runs",
            headers={**_headers(), "X-Service-Tenant": "tenant-a"},
            json=_payload(),
        )
        allowed_tenant = client.post(
            "/internal/v1/ai/agent-runs",
            headers={
                **_headers(),
                "X-Service-Scope": f"ai:agent:run,run:{RUN_ID},tenant:tenant-a",
                "X-Service-Tenant": "tenant-a",
            },
            json=_payload(),
        )

    assert missing_resource_scope.status_code == 403
    assert missing_resource_scope.json()["error"]["code"] == "SERVICE_RESOURCE_DENIED"
    assert denied_tenant.status_code == 403
    assert denied_tenant.json()["error"]["code"] == "SERVICE_TENANT_DENIED"
    assert allowed_tenant.status_code == 200


def test_signed_request_rejects_unknown_or_revoked_key_id() -> None:
    body = json.dumps(_payload(), separators=(",", ":")).encode()
    settings = _signed_settings().model_copy(
        update={"internal_revoked_key_ids": "go-core-previous"}
    )
    app = create_app(settings)
    with TestClient(app) as client:
        unknown = client.post(
            "/internal/v1/ai/agent-runs",
            headers=_signed_headers(
                body=body,
                key_id="go-core-unknown",
                nonce="nonce-unknown01",
            ),
            content=body,
        )
        revoked = client.post(
            "/internal/v1/ai/agent-runs",
            headers=_signed_headers(
                body=body,
                key=PREVIOUS_SIGNING_KEY,
                key_id="go-core-previous",
                nonce="nonce-revoked01",
            ),
            content=body,
        )
    assert unknown.status_code == 401
    assert unknown.json()["error"]["code"] == "SERVICE_KEY_INVALID"
    assert revoked.status_code == 401
    assert revoked.json()["error"]["code"] == "SERVICE_KEY_INVALID"


class _FailingNonceStore:
    async def accept(self, key: str, *, ttl_seconds: int) -> bool:
        _ = key, ttl_seconds
        raise ConnectionError("nonce store unavailable")

    async def close(self) -> None:
        return None


class _FailingKeyProvider:
    def active_key(self, purpose: str) -> KeyMaterial:
        _ = purpose
        raise KeyProviderError("KMS unavailable")

    def key_by_id(self, key_id: str, purpose: str) -> KeyMaterial | None:
        _ = key_id, purpose
        raise KeyProviderError("KMS unavailable")


def test_signed_request_fails_closed_when_nonce_store_is_unavailable() -> None:
    body = json.dumps(_payload(), separators=(",", ":")).encode()
    failing: NonceStore = _FailingNonceStore()
    app = create_app(_signed_settings(), nonce_store=failing)
    with TestClient(app) as client:
        response = client.post(
            "/internal/v1/ai/agent-runs",
            headers=_signed_headers(body=body, nonce="nonce-storefail"),
            content=body,
        )
    assert response.status_code == 503
    assert response.json()["error"]["code"] == "NONCE_STORE_UNAVAILABLE"


def test_signed_request_fails_closed_when_key_provider_is_unavailable() -> None:
    body = json.dumps(_payload(), separators=(",", ":")).encode()
    failing: KeyProvider = _FailingKeyProvider()
    app = create_app(_signed_settings(), key_provider=failing)
    with TestClient(app) as client:
        response = client.post(
            "/internal/v1/ai/agent-runs",
            headers=_signed_headers(body=body, nonce="nonce-keyfail01"),
            content=body,
        )
    assert response.status_code == 503
    assert response.json()["error"]["code"] == "SERVICE_KEY_UNAVAILABLE"


def test_nonce_store_is_reused_across_apps() -> None:
    body = json.dumps(_payload(), separators=(",", ":")).encode()
    shared_store: NonceStore = InMemoryNonceStore()
    headers = _signed_headers(body=body, nonce="nonce-shared01")

    first_app = create_app(_signed_settings(), nonce_store=shared_store)
    with TestClient(first_app) as client:
        first = client.post(
            "/internal/v1/ai/agent-runs",
            headers=headers,
            content=body,
        )

    second_app = create_app(_signed_settings(), nonce_store=shared_store)
    with TestClient(second_app) as client:
        replay = client.post(
            "/internal/v1/ai/agent-runs",
            headers=headers,
            content=body,
        )

    assert first.status_code == 200
    assert replay.status_code == 401
    assert replay.json()["error"]["code"] == "REQUEST_REPLAYED"
