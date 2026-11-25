from __future__ import annotations

import asyncio
import io
import logging
import time
from pathlib import Path
from typing import cast

import pytest
from redis.asyncio import Redis

from mysteryverse_ai.logging import JsonFormatter
from mysteryverse_ai.security.acl import (
    AclError,
    ServicePrincipal,
    authorize_operation,
    authorize_resource,
    authorize_tenant,
    parse_service_scopes,
    parse_tenant,
)
from mysteryverse_ai.security.audit import audit_security_event
from mysteryverse_ai.security.keys import (
    EnvironmentKeyProvider,
    KeyMaterial,
    KeyProviderError,
)
from mysteryverse_ai.security.nonce import RedisNonceStore
from mysteryverse_ai.security.tls import server_tls_options


class _FakeRedis:
    def __init__(self) -> None:
        self.values: dict[str, float] = {}
        self.lock = asyncio.Lock()
        self.closed = False

    async def set(
        self,
        key: str,
        value: str,
        *,
        ex: int,
        nx: bool,
    ) -> bool | None:
        _ = value, nx
        now = time.monotonic()
        async with self.lock:
            expired = [name for name, expiry in self.values.items() if expiry <= now]
            for name in expired:
                self.values.pop(name, None)
            if key in self.values:
                return None
            self.values[key] = now + ex
            return True

    async def aclose(self) -> None:
        self.closed = True


def test_environment_key_provider_rotation_and_revocation() -> None:
    provider = EnvironmentKeyProvider(
        purpose="internal.request-signing",
        keys=(
            KeyMaterial(
                key_id="go-core-v2",
                secret=b"a" * 32,
                status="active",
                purpose="internal.request-signing",
            ),
            KeyMaterial(
                key_id="go-core-v1",
                secret=b"b" * 32,
                status="previous",
                purpose="internal.request-signing",
            ),
        ),
    )

    assert provider.active_key("internal.request-signing").key_id == "go-core-v2"
    assert provider.key_by_id("go-core-v1", "internal.request-signing") is not None
    assert provider.key_by_id("unknown", "internal.request-signing") is None
    assert provider.key_by_id("go-core-v1", "other-purpose") is None

    with pytest.raises(KeyProviderError):
        EnvironmentKeyProvider(
            purpose="internal.request-signing",
            keys=(KeyMaterial("short", b"short", "active", "internal.request-signing"),),
        )


def test_service_scope_and_tenant_acl() -> None:
    scopes = parse_service_scopes("ai:agent:run,run:run-1,tenant:tenant-a")
    principal = ServicePrincipal(
        service_id="go-core",
        scopes=scopes,
        key_id="go-core-v1",
        tenant_id=parse_tenant("tenant-a"),
    )

    authorize_operation(principal, "ai:agent:run")
    authorize_resource(principal, resource_type="run", resource_id="run-1")
    authorize_tenant(principal)

    with pytest.raises(AclError) as denied:
        authorize_resource(principal, resource_type="run", resource_id="run-2")
    assert denied.value.reason == "SERVICE_RESOURCE_DENIED"

    cross_tenant = ServicePrincipal(
        service_id="go-core",
        scopes=scopes,
        key_id="go-core-v1",
        tenant_id="tenant-b",
    )
    with pytest.raises(AclError) as tenant_denied:
        authorize_tenant(cross_tenant)
    assert tenant_denied.value.reason == "SERVICE_TENANT_DENIED"

    with pytest.raises(AclError):
        parse_service_scopes("")
    with pytest.raises(AclError):
        parse_service_scopes("ai:agent:run,bad scope")


def test_redis_nonce_store_is_atomic_and_shared() -> None:
    fake = _FakeRedis()
    store = RedisNonceStore(
        client=cast(Redis, fake),
        key_prefix="mv:test:nonce",
    )

    async def scenario() -> None:
        results = await asyncio.gather(
            *[store.accept("go-core:key-1:nonce-1", ttl_seconds=30) for _ in range(20)]
        )
        assert results.count(True) == 1
        assert results.count(False) == 19
        assert await store.accept("go-core:key-1:nonce-2", ttl_seconds=30)
        await store.close()

    asyncio.run(scenario())
    assert fake.closed is True


def test_security_audit_does_not_log_secrets() -> None:
    stream = io.StringIO()
    handler = logging.StreamHandler(stream)
    handler.setFormatter(JsonFormatter())
    logger = logging.getLogger("mysteryverse_ai.security.audit")
    previous_level = logger.level
    logger.setLevel(logging.INFO)
    logger.addHandler(handler)
    try:
        audit_security_event(
            service_id="go-core",
            request_id="request-1",
            key_id="go-core-v1",
            action="/internal/v1/ai/agent-runs",
            authentication="pass",
            authorization="pass",
            scopes=frozenset({"ai:agent:run"}),
            failure_reason=None,
        )
    finally:
        logger.removeHandler(handler)
        logger.setLevel(previous_level)

    output = stream.getvalue()
    assert "go-core-v1" in output
    assert '"authz_result": "pass"' in output
    assert "test-service-token" not in output
    assert "signing-key-material" not in output
    assert "private script" not in output


def test_mtls_options_require_readable_certificate_files(
    tmp_path: Path,
) -> None:
    from mysteryverse_ai.config import Settings

    cert = tmp_path / "server.crt"
    key = tmp_path / "server.key"
    ca = tmp_path / "ca.crt"
    for path in (cert, key, ca):
        path.write_text("test certificate material", encoding="utf-8")

    settings = Settings.for_test().model_copy(
        update={
            "internal_mtls_enabled": True,
            "internal_tls_cert_file": str(cert),
            "internal_tls_key_file": str(key),
            "internal_tls_ca_file": str(ca),
            "internal_tls_require_client_cert": True,
        }
    )
    options = server_tls_options(settings)
    assert options is not None
    assert options.certfile == str(cert)
    assert options.ca_certs == str(ca)

    missing = settings.model_copy(update={"internal_tls_ca_file": None})
    with pytest.raises(Exception, match="client CA is required"):
        server_tls_options(missing)
