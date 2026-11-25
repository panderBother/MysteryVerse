-- name: CreateSecretPayload :execrows
INSERT INTO secret_payloads (
    id,
    scope_type,
    scope_id,
    owner_type,
    owner_id,
    purpose_code,
    version_no,
    sensitivity,
    visibility_scope,
    schema_version,
    key_id,
    nonce,
    ciphertext,
    content_hash,
    compression,
    retention_class,
    expires_at
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
    ?,
    ?,
    'none',
    ?,
    ?
);

-- name: GetSecretPayload :one
SELECT *
FROM secret_payloads
WHERE id = ?;
