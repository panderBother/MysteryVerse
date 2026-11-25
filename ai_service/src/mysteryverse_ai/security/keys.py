from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Literal, Protocol

from mysteryverse_ai.config import Settings

_KEY_ID_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{2,127}$")
KeyMaterialStatus = Literal["active", "previous", "revoked"]


class KeyProviderError(Exception):
    pass


@dataclass(frozen=True, slots=True)
class KeyMaterial:
    key_id: str
    secret: bytes
    status: KeyMaterialStatus
    purpose: str


class KeyProvider(Protocol):
    def active_key(self, purpose: str) -> KeyMaterial: ...

    def key_by_id(self, key_id: str, purpose: str) -> KeyMaterial | None: ...


class EnvironmentKeyProvider:
    def __init__(self, *, purpose: str, keys: tuple[KeyMaterial, ...]) -> None:
        self._purpose = purpose
        self._keys = {item.key_id: item for item in keys}
        if len(self._keys) != len(keys):
            raise KeyProviderError("duplicate signing key IDs are not allowed")
        if any(
            not _KEY_ID_PATTERN.fullmatch(item.key_id)
            or len(item.secret) < 32
            or item.purpose != purpose
            for item in keys
        ):
            raise KeyProviderError("signing key material is invalid")
        active = [item for item in keys if item.status == "active"]
        if len(active) != 1:
            raise KeyProviderError("exactly one active signing key is required")

    def active_key(self, purpose: str) -> KeyMaterial:
        if purpose != self._purpose:
            raise KeyProviderError("signing key purpose is not available")
        for item in self._keys.values():
            if item.status == "active":
                return item
        raise KeyProviderError("active signing key is unavailable")

    def key_by_id(self, key_id: str, purpose: str) -> KeyMaterial | None:
        if purpose != self._purpose:
            return None
        item = self._keys.get(key_id)
        if item is None or item.status == "revoked":
            return None
        return item


def build_key_provider(settings: Settings) -> KeyProvider:
    if settings.internal_kms_provider != "environment":
        raise KeyProviderError(
            "the configured external KMS provider is not implemented in this environment"
        )

    keys: list[KeyMaterial] = []
    active_secret = (
        settings.internal_request_signing_key.get_secret_value()
        if settings.internal_request_signing_key is not None
        else ""
    )
    if active_secret:
        if not settings.internal_signing_key_id:
            raise KeyProviderError("active signing key ID is required")
        if settings.internal_signing_key_id in settings.internal_revoked_key_ids_set:
            raise KeyProviderError("the active signing key ID is revoked")
        keys.append(
            KeyMaterial(
                key_id=settings.internal_signing_key_id,
                secret=active_secret.encode(),
                status="active",
                purpose="internal.request-signing",
            )
        )

    previous_secret = (
        settings.internal_request_signing_previous_key.get_secret_value()
        if settings.internal_request_signing_previous_key is not None
        else ""
    )
    if previous_secret:
        if not settings.internal_request_signing_previous_key_id:
            raise KeyProviderError("previous signing key ID is required")
        if (
            settings.internal_request_signing_previous_key_id
            in settings.internal_revoked_key_ids_set
        ):
            pass
        else:
            keys.append(
                KeyMaterial(
                    key_id=settings.internal_request_signing_previous_key_id,
                    secret=previous_secret.encode(),
                    status="previous",
                    purpose="internal.request-signing",
                )
            )

    if not keys:
        return _DisabledKeyProvider()
    return EnvironmentKeyProvider(purpose="internal.request-signing", keys=tuple(keys))


class _DisabledKeyProvider:
    def active_key(self, purpose: str) -> KeyMaterial:
        _ = purpose
        raise KeyProviderError("request signing is disabled")

    def key_by_id(self, key_id: str, purpose: str) -> KeyMaterial | None:
        _ = key_id, purpose
        return None
