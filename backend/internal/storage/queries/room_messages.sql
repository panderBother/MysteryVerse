-- name: GetRoomMessageByClientMessageID :one
SELECT *
FROM room_messages
WHERE room_id = ?
  AND sender_user_id = ?
  AND client_message_id = ?;

-- name: GetRoomMessageByIdempotencyKey :one
SELECT *
FROM room_messages
WHERE room_id = ?
  AND sender_user_id = ?
  AND idempotency_key_hash = ?;

-- name: CreateRoomMessage :execrows
INSERT INTO room_messages (
    id,
    event_id,
    room_id,
    chat_seq,
    sender_user_id,
    client_message_id,
    idempotency_key_hash,
    message_type,
    content_text,
    content_hash,
    moderation_status,
    schema_version,
    sent_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    'text',
    ?,
    ?,
    ?,
    ?,
    ?
);

-- name: ListRoomMessagesBefore :many
SELECT *
FROM room_messages
WHERE room_id = ?
  AND deleted_at IS NULL
  AND chat_seq < ?
ORDER BY chat_seq DESC
LIMIT ?;

-- name: CountActiveMutes :one
SELECT COUNT(*)
FROM user_sanctions
WHERE user_id = ?
  AND sanction_type = 'mute'
  AND revoked_at IS NULL
  AND starts_at <= ?
  AND (expires_at IS NULL OR expires_at > ?)
  AND (scope = 'platform' OR (scope = 'room' AND scope_ref_id = ?));
