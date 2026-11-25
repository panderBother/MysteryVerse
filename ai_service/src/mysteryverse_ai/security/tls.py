from __future__ import annotations

import ssl
from dataclasses import dataclass
from pathlib import Path

from mysteryverse_ai.config import Settings


class TLSConfigurationError(Exception):
    pass


@dataclass(frozen=True, slots=True)
class ServerTLSOptions:
    certfile: str
    keyfile: str
    ca_certs: str
    cert_reqs: int


def server_tls_options(settings: Settings) -> ServerTLSOptions | None:
    if not settings.internal_mtls_enabled:
        return None
    if not settings.internal_tls_cert_file or not settings.internal_tls_key_file:
        raise TLSConfigurationError("server certificate and private key are required")
    if settings.internal_tls_require_client_cert and not settings.internal_tls_ca_file:
        raise TLSConfigurationError("client CA is required when mTLS is enabled")
    required_paths = [
        settings.internal_tls_cert_file,
        settings.internal_tls_key_file,
    ]
    if settings.internal_tls_require_client_cert:
        required_paths.append(settings.internal_tls_ca_file or "")
    if any(not Path(path).is_file() for path in required_paths):
        raise TLSConfigurationError("configured TLS files are not readable")
    return ServerTLSOptions(
        certfile=settings.internal_tls_cert_file,
        keyfile=settings.internal_tls_key_file,
        ca_certs=settings.internal_tls_ca_file or "",
        cert_reqs=(
            ssl.CERT_REQUIRED if settings.internal_tls_require_client_cert else ssl.CERT_OPTIONAL
        ),
    )
