-- name: CreateGameRoom :execrows
INSERT INTO game_rooms (
    id,
    room_code_hash,
    host_user_id,
    script_version_id,
    game_type,
    rule_template_id,
    title,
    visibility,
    password_hash,
    capacity_min,
    capacity_max,
    mode_config_json,
    mode_config_schema_version,
    privacy_policy_version,
    status,
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
    ?,
    ?
);

-- name: GetGameRoom :one
SELECT *
FROM game_rooms
WHERE id = ?;

-- name: GetGameRoomByCode :one
SELECT *
FROM game_rooms
WHERE room_code_hash = ?;

-- name: CreateRoomMember :execrows
INSERT INTO room_members (
    id,
    room_id,
    user_id,
    participant_kind,
    seat_no,
    room_role,
    member_status,
    ready_at,
    joined_at,
    connection_state
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

-- name: ListRoomMembers :many
SELECT *
FROM room_members
WHERE room_id = ?
ORDER BY seat_no IS NULL, seat_no, joined_at, id;

-- name: CountActiveRoomSeats :one
SELECT COUNT(*)
FROM room_members
WHERE room_id = ? AND seat_no IS NOT NULL AND left_at IS NULL;
