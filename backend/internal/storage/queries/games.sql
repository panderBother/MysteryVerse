-- name: CreateGameSession :execrows
INSERT INTO game_sessions (
    id,
    room_id,
    game_type,
    script_version_id,
    rule_template_id,
    engine_key,
    engine_version,
    package_uri,
    package_hash,
    schema_version,
    status,
    state_version,
    current_event_seq,
    state_hash,
    owner_fencing_token,
    started_at
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

-- name: GetGameSession :one
SELECT *
FROM game_sessions
WHERE id = ?;

-- name: GetGameSessionForUpdate :one
SELECT *
FROM game_sessions
WHERE id = ?
FOR UPDATE;

-- name: UpdateGameSessionVersion :execrows
UPDATE game_sessions
SET state_version = ?,
    current_event_seq = ?,
    state_hash = ?,
    row_version = row_version + 1
WHERE id = ?
  AND state_version = ?;

-- name: CreateGamePlayer :execrows
INSERT INTO game_players (
    id,
    game_session_id,
    room_member_id,
    user_id,
    seat_no,
    participant_kind,
    display_name_snapshot,
    runtime_status,
    connection_state,
    joined_at
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

-- name: ListGamePlayers :many
SELECT *
FROM game_players
WHERE game_session_id = ?
ORDER BY seat_no, id;

-- name: CreateGameEvent :execrows
INSERT INTO game_events (
    id,
    game_session_id,
    seq,
    event_type,
    event_category,
    actor_type,
    actor_id,
    intent_id,
    caused_by_event_id,
    visibility_scope,
    visibility_json,
    sensitivity,
    reveal_gate_key,
    payload_json,
    secret_payload_id,
    payload_hash,
    schema_version,
    state_version_before,
    state_version_after,
    occurred_at,
    recorded_at
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
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
);

-- name: ListGameEventsAfter :many
SELECT *
FROM game_events
WHERE game_session_id = ? AND seq > ?
ORDER BY seq
LIMIT ?;
