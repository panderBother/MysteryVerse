-- name: CreateUser :execrows
INSERT INTO users (
    id,
    public_no,
    status,
    risk_level,
    terms_version,
    privacy_version
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
);

-- name: GetUser :one
SELECT *
FROM users
WHERE id = ? AND deleted_at IS NULL;
