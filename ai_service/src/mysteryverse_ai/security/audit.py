from __future__ import annotations

import logging
from typing import Literal

from mysteryverse_ai.observability.redaction import log_structured


def audit_security_event(
    *,
    service_id: str,
    request_id: str,
    key_id: str | None,
    action: str,
    authentication: Literal["pass", "fail", "not_configured"],
    authorization: Literal["pass", "fail", "not_evaluated"],
    scopes: frozenset[str],
    failure_reason: str | None,
) -> None:
    log_structured(
        logging.getLogger("mysteryverse_ai.security.audit"),
        "internal service security decision",
        service_id=service_id or "[MISSING]",
        request_id=request_id or "[MISSING]",
        key_id=key_id,
        action=action,
        authn_result=authentication,
        authz_result=authorization,
        scopes=sorted(scopes),
        failure_reason=failure_reason,
    )
