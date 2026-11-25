-- name: CreateReport :execrows
INSERT INTO reports (
    id,
    reporter_user_id,
    object_type,
    object_id,
    reason_code,
    details,
    evidence_json,
    status,
    assigned_to
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
);

-- name: GetReport :one
SELECT *
FROM reports
WHERE id = ?;

-- name: CreateAuditLog :execrows
INSERT INTO audit_logs (
    id,
    actor_type,
    actor_id,
    action,
    object_type,
    object_id,
    scope_json,
    request_id,
    trace_id,
    before_digest,
    after_digest,
    reason,
    ip_hash
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
);

-- name: ListAuditLogsByObject :many
SELECT *
FROM audit_logs
WHERE object_type = ? AND object_id = ?
ORDER BY created_at DESC, id DESC
LIMIT ?;
