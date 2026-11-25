-- name: GetScriptVersionForRoom :one
SELECT
    sv.id AS script_version_id,
    sv.script_id,
    sv.status AS version_status,
    sv.schema_version,
    sv.rule_template_id,
    s.owner_user_id,
    s.title,
    s.game_type,
    s.status AS script_status,
    s.min_players,
    s.max_players
FROM script_versions sv
JOIN scripts s ON s.id = sv.script_id
WHERE sv.id = ?
  AND s.deleted_at IS NULL;

-- name: GetGameRoomForUpdate :one
SELECT *
FROM game_rooms
WHERE id = ?
FOR UPDATE;

-- name: UpdateGameRoomMutation :execrows
UPDATE game_rooms
SET status = ?,
    current_event_seq = ?,
    started_at = ?,
    closed_at = ?,
    row_version = row_version + 1
WHERE id = ?;

-- name: UpdateGameRoomHost :execrows
UPDATE game_rooms
SET host_user_id = ?,
    row_version = row_version + 1
WHERE id = ?
  AND host_user_id = ?;

-- name: CreateRoomEvent :execrows
INSERT INTO room_events (
    id,
    room_id,
    seq,
    event_type,
    actor_type,
    actor_id,
    visibility_scope,
    visibility_json,
    payload_json,
    payload_hash,
    occurred_at
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
    ?
);

-- name: ListRoomEventsAfter :many
SELECT *
FROM room_events
WHERE room_id = ? AND seq > ?
ORDER BY seq
LIMIT ?;

-- name: GetActiveRoomMemberByUser :one
SELECT *
FROM room_members
WHERE room_id = ?
  AND user_id = ?
  AND left_at IS NULL
LIMIT 1;

-- name: GetLatestRoomMemberByUser :one
SELECT *
FROM room_members
WHERE room_id = ?
  AND user_id = ?
ORDER BY joined_at DESC, id DESC
LIMIT 1;

-- name: GetRoomMemberForUpdate :one
SELECT *
FROM room_members
WHERE id = ?
FOR UPDATE;

-- name: ListActiveRoomMembers :many
SELECT
    rm.*,
    u.public_no,
    p.nickname,
    p.avatar_uri
FROM room_members rm
JOIN users u ON u.id = rm.user_id
JOIN user_profiles p ON p.user_id = u.id
WHERE rm.room_id = ?
  AND rm.left_at IS NULL
ORDER BY rm.seat_no IS NULL, rm.seat_no, rm.joined_at, rm.id;

-- name: UpdateRoomMemberReady :execrows
UPDATE room_members
SET member_status = ?,
    ready_at = ?,
    row_version = row_version + 1
WHERE id = ?
  AND left_at IS NULL;

-- name: UpdateRoomMemberLeave :execrows
UPDATE room_members
SET member_status = 'left',
    left_at = ?,
    connection_state = 'offline',
    row_version = row_version + 1
WHERE id = ?
  AND left_at IS NULL;

-- name: UpdateRoomMemberRole :execrows
UPDATE room_members
SET room_role = ?,
    row_version = row_version + 1
WHERE id = ?
  AND left_at IS NULL;

-- name: CountActiveRoomMembers :one
SELECT COUNT(*)
FROM room_members
WHERE room_id = ? AND left_at IS NULL;

-- name: CountReadyRoomMembers :one
SELECT COUNT(*)
FROM room_members
WHERE room_id = ?
  AND left_at IS NULL
  AND member_status = 'ready';

-- name: CreateInvitationToken :execrows
INSERT INTO invitation_tokens (
    id,
    room_id,
    created_by,
    token_hash,
    purpose,
    max_uses,
    use_count,
    expires_at,
    invitee_rule_json
) VALUES (
    ?,
    ?,
    ?,
    ?,
    'room_invite',
    ?,
    0,
    ?,
    ?
);

-- name: GetInvitationForUpdate :one
SELECT *
FROM invitation_tokens
WHERE token_hash = ?
FOR UPDATE;

-- name: IncrementInvitationUse :execrows
UPDATE invitation_tokens
SET use_count = use_count + 1,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?
  AND revoked_at IS NULL
  AND use_count < max_uses;

-- name: ListPublicRooms :many
SELECT
    r.*,
    s.title AS script_title,
    p.nickname AS host_nickname,
    (
        SELECT COUNT(*)
        FROM room_members rm
        WHERE rm.room_id = r.id AND rm.left_at IS NULL
    ) AS active_members
FROM game_rooms r
JOIN script_versions sv ON sv.id = r.script_version_id
JOIN scripts s ON s.id = sv.script_id
JOIN user_profiles p ON p.user_id = r.host_user_id
WHERE r.visibility = 'public'
  AND r.status = 'waiting'
  AND (r.expires_at IS NULL OR r.expires_at > ?)
ORDER BY r.created_at DESC, r.id DESC
LIMIT ?;

-- name: CloseExpiredRooms :execrows
UPDATE game_rooms
SET status = 'cancelled',
    closed_at = ?,
    current_event_seq = current_event_seq + 1,
    row_version = row_version + 1
WHERE status = 'waiting'
  AND expires_at IS NOT NULL
  AND expires_at <= ?;

-- name: ListExpiredWaitingRoomIDs :many
SELECT id
FROM game_rooms
WHERE status = 'waiting'
  AND expires_at IS NOT NULL
  AND expires_at <= ?
ORDER BY expires_at
LIMIT ?;
