-- name: GetIdempotencyRecord :one
SELECT *
FROM idempotency_records
WHERE scope = ?
  AND actor_scope = ?
  AND idempotency_key = ?;

-- name: CreateIdempotencyRecord :execrows
INSERT INTO idempotency_records (
    id,
    scope,
    actor_scope,
    idempotency_key,
    request_hash,
    status,
    expires_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    'pending',
    ?
);

-- name: CompleteIdempotencyRecord :execrows
UPDATE idempotency_records
SET status = 'completed',
    response_status = ?,
    response_json = ?,
    resource_type = ?,
    resource_id = ?,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?;
