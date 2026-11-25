-- name: GetAuthIdentityBySubject :one
SELECT *
FROM auth_identities
WHERE provider = ? AND provider_subject = ?
LIMIT 1;

-- name: CreateAuthIdentity :execrows
INSERT INTO auth_identities (
    id,
    user_id,
    provider,
    provider_subject,
    identifier_hash,
    identifier_hint,
    password_hash,
    verified_at,
    is_primary,
    status
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
    ?
);

-- name: CreateUserProfile :execrows
INSERT INTO user_profiles (
    id,
    user_id,
    nickname,
    avatar_uri,
    bio,
    locale,
    timezone,
    preferences_json,
    privacy_json,
    moderation_status,
    row_version
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
    1
);

-- name: GetUserProfile :one
SELECT *
FROM user_profiles
WHERE user_id = ?;

-- name: UpdateUserProfile :execrows
UPDATE user_profiles
SET nickname = ?,
    avatar_uri = ?,
    bio = ?,
    locale = ?,
    timezone = ?,
    preferences_json = ?,
    privacy_json = ?,
    moderation_status = ?,
    row_version = row_version + 1
WHERE user_id = ?
  AND row_version = ?;

-- name: CreateUserStatistics :execrows
INSERT INTO user_statistics (
    id,
    user_id,
    stats_json,
    calculated_at
) VALUES (
    ?,
    ?,
    ?,
    ?
);

-- name: GetUserStatistics :one
SELECT *
FROM user_statistics
WHERE user_id = ?;

-- name: CreateUserSession :execrows
INSERT INTO user_sessions (
    id,
    user_id,
    session_token_hash,
    csrf_token_hash,
    device_label,
    ip_hash,
    user_agent_hash,
    expires_at,
    last_seen_at
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

-- name: GetSessionUserByToken :one
SELECT
    s.id AS session_id,
    s.user_id,
    s.csrf_token_hash,
    s.expires_at AS session_expires_at,
    s.revoked_at,
    u.public_no,
    u.status AS user_status,
    u.deleted_at,
    p.nickname,
    p.avatar_uri
FROM user_sessions s
JOIN users u ON u.id = s.user_id
JOIN user_profiles p ON p.user_id = u.id
WHERE s.session_token_hash = ?
LIMIT 1;

-- name: RevokeSession :execrows
UPDATE user_sessions
SET revoked_at = ?,
    revoke_reason = ?,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?
  AND revoked_at IS NULL;

-- name: RevokeAllUserSessions :execrows
UPDATE user_sessions
SET revoked_at = ?,
    revoke_reason = ?,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE user_id = ?
  AND revoked_at IS NULL;

-- name: UpdateSessionLastSeen :exec
UPDATE user_sessions
SET last_seen_at = ?
WHERE id = ?
  AND revoked_at IS NULL;

-- name: UpdateUserLastLogin :exec
UPDATE users
SET last_login_at = ?
WHERE id = ?
  AND deleted_at IS NULL;
