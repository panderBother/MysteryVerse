-- name: ListScriptPhasesForGame :many
SELECT *
FROM script_phases
WHERE script_version_id = ?
ORDER BY sequence_no, id;

-- name: ListScriptCharactersForGame :many
SELECT *
FROM script_characters
WHERE script_version_id = ?
ORDER BY sort_order, id;

-- name: ListScriptCluesForGame :many
SELECT *
FROM script_clues
WHERE script_version_id = ?
ORDER BY sort_order, id;

-- name: ListScriptSearchPointsForGame :many
SELECT *
FROM script_search_points
WHERE script_version_id = ?
ORDER BY id;

-- name: ListSearchPointCluesForGame :many
SELECT spc.*, ph.phase_key
FROM script_search_point_clues spc
JOIN script_search_points sp ON sp.id = spc.search_point_id
JOIN script_phases ph ON ph.id = sp.phase_id
WHERE spc.script_version_id = ?
ORDER BY spc.search_point_id, spc.is_default DESC, spc.weight DESC, spc.clue_id;

-- name: GetRuleTemplateForGame :one
SELECT *
FROM rule_templates
WHERE id = ? AND status = 'active';

-- name: GetGameSessionByRoom :one
SELECT *
FROM game_sessions
WHERE room_id = ?;

-- name: CreateGameState :execrows
INSERT INTO game_states (
    id,
    game_session_id,
    public_state_json,
    secret_payload_id,
    schema_version,
    phase_key,
    phase_started_at,
    phase_deadline_at,
    state_hash,
    last_event_seq,
    materializer_version
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

-- name: GetGameState :one
SELECT *
FROM game_states
WHERE game_session_id = ?;

-- name: GetGameStateForUpdate :one
SELECT *
FROM game_states
WHERE game_session_id = ?
FOR UPDATE;

-- name: UpdateGameState :execrows
UPDATE game_states
SET public_state_json = ?,
    secret_payload_id = ?,
    schema_version = ?,
    phase_key = ?,
    phase_started_at = ?,
    phase_deadline_at = ?,
    state_hash = ?,
    last_event_seq = ?,
    materializer_version = ?
WHERE game_session_id = ?;

-- name: CreateGameRole :execrows
INSERT INTO game_roles (
    id,
    game_session_id,
    role_instance_key,
    player_id,
    role_source_type,
    source_character_id,
    faction_key,
    team_key,
    assignment_status,
    secret_payload_id,
    public_reveal_state,
    assigned_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    'script_character',
    ?,
    ?,
    ?,
    'assigned',
    ?,
    'hidden',
    ?
);

-- name: GetGamePlayerByUser :one
SELECT *
FROM game_players
WHERE game_session_id = ? AND user_id = ?;

-- name: GetGamePlayerByID :one
SELECT *
FROM game_players
WHERE id = ?;

-- name: GetGameRoleByPlayer :one
SELECT *
FROM game_roles
WHERE game_session_id = ? AND player_id = ?;

-- name: ListGameRoles :many
SELECT *
FROM game_roles
WHERE game_session_id = ?
ORDER BY id;

-- name: UpdateGamePlayerRuntimeStatus :execrows
UPDATE game_players
SET runtime_status = ?,
    left_at = ?,
    connection_state = ?
WHERE game_session_id = ?
  AND id = ?;

-- name: UpdateGameSessionCommand :execrows
UPDATE game_sessions
SET status = ?,
    state_version = ?,
    current_event_seq = ?,
    state_hash = ?,
    ended_at = ?,
    settle_reason = ?,
    row_version = row_version + 1
WHERE id = ?
  AND state_version = ?;

-- name: CreateGameIntent :execrows
INSERT INTO game_intents (
    id,
    game_session_id,
    idempotency_key,
    actor_type,
    actor_id,
    intent_type,
    client_request_id,
    payload_json,
    payload_hash,
    expected_version,
    status,
    trace_id
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
    'accepted',
    ?
);

-- name: GetGameIntentByIdempotency :one
SELECT *
FROM game_intents
WHERE game_session_id = ?
  AND idempotency_key = ?;

-- name: CompleteGameIntent :execrows
UPDATE game_intents
SET status = ?,
    result_code = ?,
    result_summary_json = ?,
    event_start_seq = ?,
    event_end_seq = ?,
    finished_at = ?
WHERE id = ?;

-- name: UpsertGameVote :execrows
INSERT INTO game_votes (
    id,
    game_session_id,
    round_key,
    voter_player_id,
    target_player_id,
    target_role_id,
    ballot_type,
    is_abstain,
    effective_status,
    intent_id,
    event_id,
    cast_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    'murder_final',
    ?,
    'valid',
    ?,
    ?,
    ?
)
ON DUPLICATE KEY UPDATE
    target_player_id = VALUES(target_player_id),
    target_role_id = VALUES(target_role_id),
    is_abstain = VALUES(is_abstain),
    effective_status = VALUES(effective_status),
    intent_id = VALUES(intent_id),
    event_id = VALUES(event_id),
    cast_at = VALUES(cast_at),
    row_version = row_version + 1;

-- name: ListGameVotesForRound :many
SELECT *
FROM game_votes
WHERE game_session_id = ?
  AND round_key = ?
  AND effective_status = 'valid'
ORDER BY voter_player_id;

-- name: CreateGameSearchClaim :execrows
INSERT INTO game_search_claims (
    id,
    game_session_id,
    search_point_id,
    clue_id,
    claim_slot,
    claimant_player_id,
    status,
    intent_id,
    event_id,
    claimed_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    'claimed',
    ?,
    ?,
    ?
);

-- name: CountGameSearchClaimsForPoint :one
SELECT COUNT(*)
FROM game_search_claims
WHERE game_session_id = ?
  AND search_point_id = ?
  AND status = 'claimed';

-- name: CountGameSearchClaimsForPlayerPhase :one
SELECT COUNT(*)
FROM game_search_claims c
JOIN script_search_points p ON p.id = c.search_point_id
JOIN script_phases ph ON ph.id = p.phase_id
WHERE c.game_session_id = ?
  AND c.claimant_player_id = ?
  AND ph.phase_key = ?
  AND c.status = 'claimed';

-- name: CreateGameResult :execrows
INSERT INTO game_results (
    id,
    game_session_id,
    final_state_version,
    final_event_seq,
    result_code,
    winner_team_json,
    summary_json,
    details_uri,
    details_hash,
    result_hash,
    settled_at
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

-- name: GetGameResultBySession :one
SELECT *
FROM game_results
WHERE game_session_id = ?;

-- name: RevealGameRoles :execrows
UPDATE game_roles
SET assignment_status = 'revealed',
    public_reveal_state = 'full_revealed',
    revealed_at = ?,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE game_session_id = ?
  AND assignment_status <> 'revealed';

-- name: CreateGameTimer :execrows
INSERT INTO game_timers (
    id,
    game_session_id,
    timer_key,
    phase_key,
    deadline_at,
    status,
    expected_state_version
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    'scheduled',
    ?
);

-- name: GetLatestGameTimerForUpdate :one
SELECT *
FROM game_timers
WHERE game_session_id = ?
ORDER BY created_at DESC, id DESC
LIMIT 1
FOR UPDATE;

-- name: UpdateGameTimer :execrows
UPDATE game_timers
SET deadline_at = ?,
    paused_at = ?,
    remaining_ms_at_pause = ?,
    status = ?,
    expected_state_version = ?,
    row_version = row_version + 1
WHERE id = ?;

-- name: ListExpiredRunningGameIDs :many
SELECT s.id
FROM game_sessions s
JOIN game_states st ON st.game_session_id = s.id
WHERE s.status = 'running'
  AND st.phase_deadline_at IS NOT NULL
  AND st.phase_deadline_at <= ?
ORDER BY st.phase_deadline_at
LIMIT ?;
