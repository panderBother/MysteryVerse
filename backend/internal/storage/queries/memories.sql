-- name: CreateNPCMemory :execrows
INSERT INTO npc_memories (
    id,
    game_session_id,
    game_npc_id,
    owner_actor_scope,
    memory_type,
    authority_class,
    record_version,
    importance,
    confidence,
    summary_secret_payload_id,
    source_event_ids_json,
    source_type,
    occurred_at,
    valid_from,
    valid_until,
    expires_at,
    supersedes_memory_id,
    dedupe_key,
    sensitivity,
    access_scope
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
    ?
);

-- name: ListNPCMemories :many
SELECT *
FROM npc_memories
WHERE game_session_id = ?
  AND game_npc_id = ?
  AND (expires_at IS NULL OR expires_at > ?)
ORDER BY importance DESC, occurred_at DESC, id
LIMIT ?;
