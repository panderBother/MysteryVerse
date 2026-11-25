from __future__ import annotations

import re
from dataclasses import dataclass

_SCOPE_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9:._/-]{0,159}$")
_TENANT_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9:._-]{0,127}$")


class AclError(Exception):
    def __init__(self, *, reason: str) -> None:
        super().__init__(reason)
        self.reason = reason


@dataclass(frozen=True, slots=True)
class ServicePrincipal:
    service_id: str
    scopes: frozenset[str]
    key_id: str | None
    tenant_id: str | None


def parse_service_scopes(value: str) -> frozenset[str]:
    scopes = {item.strip() for item in value.split(",") if item.strip()}
    if not scopes or any(not _SCOPE_PATTERN.fullmatch(item) for item in scopes):
        raise AclError(reason="SERVICE_SCOPE_INVALID")
    return frozenset(scopes)


def parse_tenant(value: str) -> str | None:
    tenant = value.strip()
    if not tenant:
        return None
    if not _TENANT_PATTERN.fullmatch(tenant):
        raise AclError(reason="SERVICE_TENANT_INVALID")
    return tenant


def authorize_operation(principal: ServicePrincipal, required_scope: str) -> None:
    if required_scope not in principal.scopes:
        raise AclError(reason="SERVICE_SCOPE_DENIED")


def authorize_resource(
    principal: ServicePrincipal,
    *,
    resource_type: str,
    resource_id: str,
) -> None:
    if f"{resource_type}:{resource_id}" not in principal.scopes:
        raise AclError(reason="SERVICE_RESOURCE_DENIED")


def authorize_tenant(principal: ServicePrincipal) -> None:
    if principal.tenant_id is None:
        return
    if f"tenant:{principal.tenant_id}" not in principal.scopes:
        raise AclError(reason="SERVICE_TENANT_DENIED")
