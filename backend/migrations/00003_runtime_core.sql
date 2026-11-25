-- +goose Up

CREATE TABLE game_rooms (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    room_code_hash BINARY(32) NOT NULL,
    host_user_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    game_type VARCHAR(24) NOT NULL,
    rule_template_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    title VARCHAR(120) NOT NULL,
    visibility VARCHAR(20) NOT NULL,
    password_hash VARBINARY(255) NULL,
    capacity_min TINYINT UNSIGNED NOT NULL,
    capacity_max TINYINT UNSIGNED NOT NULL,
    mode_config_json JSON NOT NULL,
    mode_config_schema_version VARCHAR(32) NOT NULL,
    privacy_policy_version VARCHAR(32) NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'waiting',
    current_event_seq BIGINT UNSIGNED NOT NULL DEFAULT 0,
    started_at DATETIME(3) NULL,
    closed_at DATETIME(3) NULL,
    expires_at DATETIME(3) NULL,
    row_version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_game_rooms_host
        FOREIGN KEY (host_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_rooms_script_version
        FOREIGN KEY (script_version_id) REFERENCES script_versions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_rooms_rule_template
        FOREIGN KEY (rule_template_id) REFERENCES rule_templates (id) ON DELETE RESTRICT,
    CONSTRAINT uq_game_rooms_code UNIQUE (room_code_hash),
    CONSTRAINT ck_game_rooms_game_type CHECK (
        game_type IN ('script_murder', 'werewolf', 'rpg')
    ),
    CONSTRAINT ck_game_rooms_visibility CHECK (
        visibility IN ('private', 'public', 'password', 'invite_only')
    ),
    CONSTRAINT ck_game_rooms_capacity CHECK (
        capacity_min > 0 AND capacity_min <= capacity_max
    ),
    CONSTRAINT ck_game_rooms_status CHECK (
        status IN (
            'waiting', 'starting', 'in_game', 'paused', 'settling',
            'result_ready', 'closed', 'cancelled', 'aborted'
        )
    ),
    INDEX ix_game_rooms_lobby (visibility, game_type, status, expires_at),
    INDEX ix_game_rooms_host (host_user_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE invitation_tokens (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    room_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    created_by CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    token_hash BINARY(32) NOT NULL,
    purpose VARCHAR(24) NOT NULL,
    max_uses INT UNSIGNED NOT NULL DEFAULT 1,
    use_count INT UNSIGNED NOT NULL DEFAULT 0,
    expires_at DATETIME(3) NOT NULL,
    revoked_at DATETIME(3) NULL,
    invitee_rule_json JSON NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_invitation_tokens_room
        FOREIGN KEY (room_id) REFERENCES game_rooms (id) ON DELETE RESTRICT,
    CONSTRAINT fk_invitation_tokens_creator
        FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT uq_invitation_tokens_hash UNIQUE (token_hash),
    CONSTRAINT ck_invitation_tokens_purpose CHECK (purpose = 'room_invite'),
    CONSTRAINT ck_invitation_tokens_uses CHECK (use_count <= max_uses),
    INDEX ix_invitation_tokens_room_expiry (room_id, expires_at),
    INDEX ix_invitation_tokens_creator (created_by, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE room_members (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    room_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    user_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    participant_kind VARCHAR(16) NOT NULL,
    seat_no TINYINT UNSIGNED NULL,
    room_role VARCHAR(16) NOT NULL,
    member_status VARCHAR(20) NOT NULL,
    ready_at DATETIME(3) NULL,
    joined_at DATETIME(3) NOT NULL,
    left_at DATETIME(3) NULL,
    connection_state VARCHAR(20) NULL,
    disconnected_at DATETIME(3) NULL,
    ai_actor_ref VARCHAR(96) NULL,
    active_user_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin
        GENERATED ALWAYS AS (
            CASE
                WHEN left_at IS NULL AND user_id IS NOT NULL THEN user_id
                ELSE NULL
            END
        ) STORED,
    row_version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_room_members_room
        FOREIGN KEY (room_id) REFERENCES game_rooms (id) ON DELETE RESTRICT,
    CONSTRAINT fk_room_members_user
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT uq_room_members_seat UNIQUE (room_id, seat_no),
    CONSTRAINT uq_room_members_active_user UNIQUE (room_id, active_user_id),
    CONSTRAINT ck_room_members_kind CHECK (
        participant_kind IN ('human', 'ai', 'spectator')
    ),
    CONSTRAINT ck_room_members_role CHECK (
        room_role IN ('host', 'player', 'spectator')
    ),
    CONSTRAINT ck_room_members_status CHECK (
        member_status IN ('joined', 'ready', 'disconnected', 'left', 'kicked')
    ),
    CONSTRAINT ck_room_members_human CHECK (
        participant_kind <> 'human' OR user_id IS NOT NULL
    ),
    INDEX ix_room_members_status (room_id, member_status),
    INDEX ix_room_members_user (user_id, joined_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE room_events (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    room_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    seq BIGINT UNSIGNED NOT NULL,
    event_type VARCHAR(96) NOT NULL,
    actor_type VARCHAR(20) NULL,
    actor_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    visibility_scope VARCHAR(24) NOT NULL,
    visibility_json JSON NULL,
    payload_json JSON NOT NULL,
    payload_hash BINARY(32) NOT NULL,
    occurred_at DATETIME(3) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_room_events_room
        FOREIGN KEY (room_id) REFERENCES game_rooms (id) ON DELETE RESTRICT,
    CONSTRAINT uq_room_events_seq UNIQUE (room_id, seq),
    CONSTRAINT ck_room_events_visibility CHECK (
        visibility_scope IN ('room', 'player', 'runtime')
    ),
    INDEX ix_room_events_type (room_id, event_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE room_messages (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    room_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    chat_seq BIGINT UNSIGNED NOT NULL,
    sender_user_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    client_message_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    message_type VARCHAR(16) NOT NULL,
    content_text TEXT NOT NULL,
    content_hash BINARY(32) NOT NULL,
    moderation_status VARCHAR(20) NOT NULL,
    deleted_at DATETIME(3) NULL,
    sent_at DATETIME(3) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_room_messages_room
        FOREIGN KEY (room_id) REFERENCES game_rooms (id) ON DELETE RESTRICT,
    CONSTRAINT fk_room_messages_sender
        FOREIGN KEY (sender_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT uq_room_messages_seq UNIQUE (room_id, chat_seq),
    CONSTRAINT uq_room_messages_client UNIQUE (room_id, sender_user_id, client_message_id),
    CONSTRAINT ck_room_messages_type CHECK (message_type = 'text'),
    CONSTRAINT ck_room_messages_content CHECK (CHAR_LENGTH(content_text) <= 4000),
    CONSTRAINT ck_room_messages_moderation CHECK (
        moderation_status IN ('pending', 'allowed', 'rejected', 'redacted')
    ),
    INDEX ix_room_messages_sender (sender_user_id),
    INDEX ix_room_messages_sent (room_id, sent_at),
    INDEX ix_room_messages_moderation (moderation_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_sessions (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    room_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    game_type VARCHAR(24) NOT NULL,
    script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    rule_template_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    engine_key VARCHAR(64) NOT NULL,
    engine_version VARCHAR(32) NOT NULL,
    package_uri VARCHAR(1024) NOT NULL,
    package_hash BINARY(32) NOT NULL,
    schema_version VARCHAR(32) NOT NULL,
    status VARCHAR(24) NOT NULL,
    state_version BIGINT UNSIGNED NOT NULL DEFAULT 0,
    current_event_seq BIGINT UNSIGNED NOT NULL DEFAULT 0,
    state_hash BINARY(32) NOT NULL,
    owner_fencing_token BIGINT UNSIGNED NOT NULL DEFAULT 0,
    started_at DATETIME(3) NOT NULL,
    ended_at DATETIME(3) NULL,
    settle_reason VARCHAR(64) NULL,
    row_version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_game_sessions_room
        FOREIGN KEY (room_id) REFERENCES game_rooms (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_sessions_script_version
        FOREIGN KEY (script_version_id) REFERENCES script_versions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_sessions_rule_template
        FOREIGN KEY (rule_template_id) REFERENCES rule_templates (id) ON DELETE RESTRICT,
    CONSTRAINT uq_game_sessions_room UNIQUE (room_id),
    CONSTRAINT ck_game_sessions_game_type CHECK (
        game_type IN ('script_murder', 'werewolf', 'rpg')
    ),
    CONSTRAINT ck_game_sessions_status CHECK (
        status IN ('starting', 'running', 'paused', 'settling', 'completed', 'aborted')
    ),
    INDEX ix_game_sessions_status (status, updated_at),
    INDEX ix_game_sessions_engine (engine_version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_players (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    room_member_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    user_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    seat_no TINYINT UNSIGNED NOT NULL,
    participant_kind VARCHAR(16) NOT NULL,
    display_name_snapshot VARCHAR(96) NOT NULL,
    runtime_status VARCHAR(20) NOT NULL,
    connection_state VARCHAR(20) NULL,
    joined_at DATETIME(3) NOT NULL,
    left_at DATETIME(3) NULL,
    result_summary_json JSON NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_game_players_session
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_players_member
        FOREIGN KEY (room_member_id) REFERENCES room_members (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_players_user
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT uq_game_players_seat UNIQUE (game_session_id, seat_no),
    CONSTRAINT uq_game_players_user UNIQUE (game_session_id, user_id),
    CONSTRAINT uq_game_players_member UNIQUE (room_member_id),
    CONSTRAINT uq_game_players_identity UNIQUE (game_session_id, id),
    CONSTRAINT ck_game_players_kind CHECK (
        participant_kind IN ('human', 'ai_player', 'npc')
    ),
    INDEX ix_game_players_user_time (user_id, joined_at),
    INDEX ix_game_players_runtime (game_session_id, runtime_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_roles (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    role_instance_key VARCHAR(96) NOT NULL,
    player_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    role_source_type VARCHAR(24) NOT NULL,
    source_character_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    faction_key VARCHAR(64) NOT NULL,
    team_key VARCHAR(64) NULL,
    assignment_status VARCHAR(20) NOT NULL,
    secret_payload_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    public_reveal_state VARCHAR(24) NOT NULL DEFAULT 'hidden',
    assigned_at DATETIME(3) NOT NULL,
    revealed_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_game_roles_session
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_roles_player
        FOREIGN KEY (game_session_id, player_id)
        REFERENCES game_players (game_session_id, id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_roles_character
        FOREIGN KEY (source_character_id) REFERENCES script_characters (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_roles_secret
        FOREIGN KEY (secret_payload_id) REFERENCES secret_payloads (id) ON DELETE RESTRICT,
    CONSTRAINT uq_game_roles_instance UNIQUE (game_session_id, role_instance_key),
    CONSTRAINT uq_game_roles_player UNIQUE (game_session_id, player_id),
    CONSTRAINT ck_game_roles_source CHECK (
        role_source_type IN ('script_character', 'werewolf_template', 'rpg_character')
    ),
    CONSTRAINT ck_game_roles_assignment CHECK (
        assignment_status IN ('assigned', 'revealed', 'revoked')
    ),
    CONSTRAINT ck_game_roles_reveal CHECK (
        public_reveal_state IN ('hidden', 'role_revealed', 'full_revealed')
    ),
    INDEX ix_game_roles_faction (game_session_id, faction_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_states (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    public_state_json JSON NOT NULL,
    secret_payload_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    schema_version VARCHAR(32) NOT NULL,
    phase_key VARCHAR(96) NOT NULL,
    phase_started_at DATETIME(3) NULL,
    phase_deadline_at DATETIME(3) NULL,
    state_hash BINARY(32) NOT NULL,
    last_event_seq BIGINT UNSIGNED NOT NULL,
    materializer_version VARCHAR(32) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_game_states_session
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_states_secret
        FOREIGN KEY (secret_payload_id) REFERENCES secret_payloads (id) ON DELETE RESTRICT,
    CONSTRAINT uq_game_states_session UNIQUE (game_session_id),
    INDEX ix_game_states_deadline (phase_deadline_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_intents (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    actor_type VARCHAR(20) NOT NULL,
    actor_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    intent_type VARCHAR(64) NOT NULL,
    client_request_id VARCHAR(128) NULL,
    payload_json JSON NULL,
    secret_payload_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    payload_hash BINARY(32) NOT NULL,
    expected_version BIGINT UNSIGNED NOT NULL,
    status VARCHAR(20) NOT NULL,
    result_code VARCHAR(64) NULL,
    result_summary_json JSON NULL,
    response_secret_payload_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    event_start_seq BIGINT UNSIGNED NULL,
    event_end_seq BIGINT UNSIGNED NULL,
    trace_id VARCHAR(64) NOT NULL,
    finished_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_game_intents_session
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_intents_secret
        FOREIGN KEY (secret_payload_id) REFERENCES secret_payloads (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_intents_response_secret
        FOREIGN KEY (response_secret_payload_id) REFERENCES secret_payloads (id) ON DELETE RESTRICT,
    CONSTRAINT uq_game_intents_idempotency UNIQUE (game_session_id, idempotency_key),
    CONSTRAINT uq_game_intents_request UNIQUE (game_session_id, client_request_id),
    CONSTRAINT ck_game_intents_payload CHECK (
        payload_json IS NOT NULL OR secret_payload_id IS NOT NULL
    ),
    CONSTRAINT ck_game_intents_actor CHECK (
        actor_type IN ('player', 'npc', 'dm', 'system')
    ),
    CONSTRAINT ck_game_intents_status CHECK (
        status IN ('accepted', 'rejected', 'retryable_conflict', 'executed')
    ),
    CONSTRAINT ck_game_intents_event_range CHECK (
        event_start_seq IS NULL OR event_end_seq IS NULL OR event_start_seq <= event_end_seq
    ),
    INDEX ix_game_intents_actor (actor_type, actor_id, created_at),
    INDEX ix_game_intents_status (game_session_id, status),
    INDEX ix_game_intents_trace (trace_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_events (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    seq BIGINT UNSIGNED NOT NULL,
    event_type VARCHAR(96) NOT NULL,
    event_category VARCHAR(24) NOT NULL,
    actor_type VARCHAR(20) NULL,
    actor_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    intent_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    caused_by_event_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    visibility_scope VARCHAR(24) NOT NULL,
    visibility_json JSON NULL,
    sensitivity TINYINT UNSIGNED NOT NULL,
    reveal_gate_key VARCHAR(96) NULL,
    payload_json JSON NULL,
    secret_payload_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    payload_hash BINARY(32) NOT NULL,
    schema_version VARCHAR(32) NOT NULL,
    state_version_before BIGINT UNSIGNED NULL,
    state_version_after BIGINT UNSIGNED NULL,
    occurred_at DATETIME(3) NOT NULL,
    recorded_at DATETIME(3) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_game_events_session
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_events_intent
        FOREIGN KEY (intent_id) REFERENCES game_intents (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_events_cause
        FOREIGN KEY (caused_by_event_id) REFERENCES game_events (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_events_secret
        FOREIGN KEY (secret_payload_id) REFERENCES secret_payloads (id) ON DELETE RESTRICT,
    CONSTRAINT uq_game_events_seq UNIQUE (game_session_id, seq),
    CONSTRAINT ck_game_events_payload CHECK (
        payload_json IS NOT NULL OR secret_payload_id IS NOT NULL
    ),
    CONSTRAINT ck_game_events_sensitivity CHECK (sensitivity BETWEEN 0 AND 4),
    CONSTRAINT ck_game_events_category CHECK (
        event_category IN ('room', 'phase', 'action', 'adjudication', 'narrative', 'incident')
    ),
    CONSTRAINT ck_game_events_visibility CHECK (
        visibility_scope IN ('public', 'room', 'role', 'role_group', 'player', 'runtime', 'audit')
    ),
    CONSTRAINT ck_game_events_state_versions CHECK (
        (state_version_before IS NULL AND state_version_after IS NULL)
        OR (
            state_version_before IS NOT NULL
            AND state_version_after = state_version_before + 1
        )
    ),
    INDEX ix_game_events_type (game_session_id, event_type),
    INDEX ix_game_events_time (game_session_id, occurred_at),
    INDEX ix_game_events_visibility (game_session_id, visibility_scope)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_votes (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    round_key VARCHAR(96) NOT NULL,
    voter_player_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    target_player_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    target_role_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    ballot_type VARCHAR(32) NOT NULL,
    is_abstain TINYINT(1) NOT NULL DEFAULT 0,
    effective_status VARCHAR(20) NOT NULL,
    intent_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    event_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    cast_at DATETIME(3) NOT NULL,
    row_version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_game_votes_session
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_votes_voter
        FOREIGN KEY (voter_player_id) REFERENCES game_players (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_votes_target_player
        FOREIGN KEY (target_player_id) REFERENCES game_players (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_votes_target_role
        FOREIGN KEY (target_role_id) REFERENCES game_roles (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_votes_intent
        FOREIGN KEY (intent_id) REFERENCES game_intents (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_votes_event
        FOREIGN KEY (event_id) REFERENCES game_events (id) ON DELETE RESTRICT,
    CONSTRAINT uq_game_votes_ballot
        UNIQUE (game_session_id, round_key, voter_player_id, ballot_type),
    CONSTRAINT ck_game_votes_target CHECK (
        is_abstain = 1 OR target_player_id IS NOT NULL OR target_role_id IS NOT NULL
    ),
    CONSTRAINT ck_game_votes_status CHECK (
        effective_status IN ('pending', 'valid', 'superseded', 'invalid')
    ),
    INDEX ix_game_votes_tally (game_session_id, round_key, effective_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_search_claims (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    search_point_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    clue_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    claim_slot INT UNSIGNED NOT NULL,
    claimant_player_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    status VARCHAR(20) NOT NULL,
    intent_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    event_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    claimed_at DATETIME(3) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_search_claims_session
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_search_claims_point
        FOREIGN KEY (search_point_id) REFERENCES script_search_points (id) ON DELETE RESTRICT,
    CONSTRAINT fk_search_claims_clue
        FOREIGN KEY (clue_id) REFERENCES script_clues (id) ON DELETE RESTRICT,
    CONSTRAINT fk_search_claims_player
        FOREIGN KEY (claimant_player_id) REFERENCES game_players (id) ON DELETE RESTRICT,
    CONSTRAINT fk_search_claims_intent
        FOREIGN KEY (intent_id) REFERENCES game_intents (id) ON DELETE RESTRICT,
    CONSTRAINT fk_search_claims_event
        FOREIGN KEY (event_id) REFERENCES game_events (id) ON DELETE RESTRICT,
    CONSTRAINT uq_search_claims_slot UNIQUE (game_session_id, clue_id, claim_slot),
    CONSTRAINT uq_search_claims_intent UNIQUE (intent_id),
    CONSTRAINT ck_search_claims_slot CHECK (claim_slot > 0),
    CONSTRAINT ck_search_claims_status CHECK (
        status IN ('claimed', 'revoked', 'compensated')
    ),
    INDEX ix_search_claims_point (game_session_id, search_point_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_messages (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    room_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    client_message_id VARCHAR(128) NULL,
    sender_type VARCHAR(20) NOT NULL,
    sender_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    message_type VARCHAR(24) NOT NULL,
    visibility_scope VARCHAR(24) NOT NULL,
    recipient_json JSON NULL,
    content_text TEXT NULL,
    secret_payload_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    content_hash BINARY(32) NOT NULL,
    room_event_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    event_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    moderation_status VARCHAR(20) NOT NULL,
    sent_at DATETIME(3) NOT NULL,
    deleted_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_game_messages_room
        FOREIGN KEY (room_id) REFERENCES game_rooms (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_messages_session
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_messages_secret
        FOREIGN KEY (secret_payload_id) REFERENCES secret_payloads (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_messages_room_event
        FOREIGN KEY (room_event_id) REFERENCES room_events (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_messages_event
        FOREIGN KEY (event_id) REFERENCES game_events (id) ON DELETE RESTRICT,
    CONSTRAINT uq_game_messages_client
        UNIQUE (room_id, sender_type, sender_id, client_message_id),
    CONSTRAINT ck_game_messages_sender CHECK (
        sender_type IN ('user', 'ai_player', 'npc', 'dm', 'system')
    ),
    CONSTRAINT ck_game_messages_type CHECK (
        message_type IN ('public', 'private', 'system', 'npc_dialog', 'dm')
    ),
    CONSTRAINT ck_game_messages_body CHECK (
        NOT (content_text IS NOT NULL AND secret_payload_id IS NOT NULL)
    ),
    CONSTRAINT ck_game_messages_moderation CHECK (
        moderation_status IN ('pending', 'allowed', 'blocked', 'redacted')
    ),
    INDEX ix_game_messages_game_time (game_session_id, sent_at, id),
    INDEX ix_game_messages_room_time (room_id, sent_at, id),
    INDEX ix_game_messages_type (game_session_id, message_type),
    INDEX ix_game_messages_visibility (visibility_scope)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_timers (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    timer_key VARCHAR(96) NOT NULL,
    phase_key VARCHAR(96) NOT NULL,
    deadline_at DATETIME(3) NOT NULL,
    paused_at DATETIME(3) NULL,
    remaining_ms_at_pause BIGINT UNSIGNED NULL,
    status VARCHAR(20) NOT NULL,
    expected_state_version BIGINT UNSIGNED NOT NULL,
    fired_event_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    row_version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_game_timers_session
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_timers_event
        FOREIGN KEY (fired_event_id) REFERENCES game_events (id) ON DELETE RESTRICT,
    CONSTRAINT uq_game_timers_key UNIQUE (game_session_id, timer_key),
    CONSTRAINT ck_game_timers_status CHECK (
        status IN ('scheduled', 'paused', 'fired', 'cancelled')
    ),
    INDEX ix_game_timers_phase (phase_key),
    INDEX ix_game_timers_deadline (deadline_at),
    INDEX ix_game_timers_status (status, deadline_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_snapshots (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    event_seq BIGINT UNSIGNED NOT NULL,
    state_version BIGINT UNSIGNED NOT NULL,
    snapshot_type VARCHAR(24) NOT NULL,
    schema_version VARCHAR(32) NOT NULL,
    engine_version VARCHAR(32) NOT NULL,
    state_hash BINARY(32) NOT NULL,
    snapshot_hash BINARY(32) NOT NULL,
    storage_uri VARCHAR(1024) NOT NULL,
    storage_key_hash BINARY(32) NOT NULL,
    inline_state_json JSON NULL,
    secret_payload_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    is_valid TINYINT(1) NOT NULL DEFAULT 1,
    created_by VARCHAR(32) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_game_snapshots_session
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_snapshots_secret
        FOREIGN KEY (secret_payload_id) REFERENCES secret_payloads (id) ON DELETE RESTRICT,
    CONSTRAINT uq_game_snapshots_seq UNIQUE (game_session_id, event_seq),
    CONSTRAINT uq_game_snapshots_storage UNIQUE (storage_key_hash),
    CONSTRAINT ck_game_snapshots_type CHECK (
        snapshot_type IN (
            'critical_phase', 'periodic', 'pause', 'owner_transfer', 'settlement'
        )
    ),
    INDEX ix_game_snapshots_latest (game_session_id, event_seq DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_viewer_manifests (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    viewer_type VARCHAR(20) NOT NULL,
    viewer_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    player_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    role_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    manifest_version INT UNSIGNED NOT NULL,
    manifest_uri VARCHAR(1024) NOT NULL,
    manifest_hash BINARY(32) NOT NULL,
    scope_summary_json JSON NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    expires_at DATETIME(3) NULL,
    CONSTRAINT fk_game_viewer_manifests_session
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_viewer_manifests_player
        FOREIGN KEY (player_id) REFERENCES game_players (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_viewer_manifests_role
        FOREIGN KEY (role_id) REFERENCES game_roles (id) ON DELETE RESTRICT,
    CONSTRAINT uq_game_viewer_manifests_version
        UNIQUE (game_session_id, viewer_type, viewer_id, manifest_version),
    CONSTRAINT ck_game_viewer_manifests_type CHECK (
        viewer_type IN ('player', 'npc', 'dm', 'auditor')
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_results (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    final_state_version BIGINT UNSIGNED NOT NULL,
    final_event_seq BIGINT UNSIGNED NOT NULL,
    result_code VARCHAR(64) NOT NULL,
    winner_team_json JSON NOT NULL,
    summary_json JSON NOT NULL,
    details_uri VARCHAR(1024) NULL,
    details_hash BINARY(32) NULL,
    result_hash BINARY(32) NOT NULL,
    settled_at DATETIME(3) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_game_results_session
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT uq_game_results_session UNIQUE (game_session_id),
    CONSTRAINT uq_game_results_hash UNIQUE (result_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE replay_reports (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    result_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    report_type VARCHAR(24) NOT NULL,
    status VARCHAR(20) NOT NULL,
    timeline_uri VARCHAR(1024) NULL,
    timeline_hash BINARY(32) NULL,
    summary_text TEXT NULL,
    ai_run_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    visibility_scope VARCHAR(24) NOT NULL,
    error_code VARCHAR(64) NULL,
    finished_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_replay_reports_session
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_replay_reports_result
        FOREIGN KEY (result_id) REFERENCES game_results (id) ON DELETE RESTRICT,
    CONSTRAINT uq_replay_reports_result UNIQUE (result_id),
    CONSTRAINT ck_replay_reports_type CHECK (
        report_type IN ('structured', 'ai_assisted', 'incident')
    ),
    CONSTRAINT ck_replay_reports_status CHECK (
        status IN ('queued', 'generating', 'ready', 'failed', 'degraded')
    ),
    INDEX ix_replay_reports_game (game_session_id),
    INDEX ix_replay_reports_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE outbox_events (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    aggregate_type VARCHAR(32) NOT NULL,
    aggregate_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    event_type VARCHAR(96) NOT NULL,
    dedupe_key VARCHAR(160) NOT NULL,
    payload_json JSON NOT NULL,
    status VARCHAR(20) NOT NULL,
    attempts INT UNSIGNED NOT NULL DEFAULT 0,
    available_at DATETIME(3) NOT NULL,
    locked_by VARCHAR(64) NULL,
    locked_until DATETIME(3) NULL,
    published_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT uq_outbox_events_dedupe UNIQUE (dedupe_key),
    CONSTRAINT ck_outbox_events_aggregate CHECK (
        aggregate_type IN ('game', 'room', 'generation_job')
    ),
    CONSTRAINT ck_outbox_events_status CHECK (
        status IN ('pending', 'published', 'failed')
    ),
    INDEX ix_outbox_events_aggregate (aggregate_type, aggregate_id),
    INDEX ix_outbox_events_dispatch (status, available_at),
    INDEX ix_outbox_events_type_status (event_type, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE game_npcs (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    npc_instance_key VARCHAR(96) NOT NULL,
    script_npc_profile_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    current_scene_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    display_name_snapshot VARCHAR(96) NOT NULL,
    current_goal VARCHAR(500) NULL,
    emotion_state VARCHAR(64) NULL,
    public_state_json JSON NULL,
    secret_payload_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    visibility_scope VARCHAR(32) NOT NULL,
    runtime_status VARCHAR(20) NOT NULL,
    last_event_seq BIGINT UNSIGNED NOT NULL,
    row_version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_game_npcs_session
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_npcs_profile
        FOREIGN KEY (script_npc_profile_id) REFERENCES npc_profiles (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_npcs_scene
        FOREIGN KEY (current_scene_id) REFERENCES script_scenes (id) ON DELETE RESTRICT,
    CONSTRAINT fk_game_npcs_secret
        FOREIGN KEY (secret_payload_id) REFERENCES secret_payloads (id) ON DELETE RESTRICT,
    CONSTRAINT uq_game_npcs_instance UNIQUE (game_session_id, npc_instance_key),
    CONSTRAINT uq_game_npcs_identity UNIQUE (id, game_session_id),
    CONSTRAINT ck_game_npcs_runtime CHECK (
        runtime_status IN ('active', 'away', 'disabled')
    ),
    INDEX ix_game_npcs_scene (game_session_id, current_scene_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE npc_relationships (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    game_npc_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    subject_type VARCHAR(20) NOT NULL,
    subject_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    relationship_label VARCHAR(64) NULL,
    attitude SMALLINT NOT NULL,
    trust SMALLINT NOT NULL,
    fear SMALLINT NOT NULL,
    source_event_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    visibility_scope VARCHAR(24) NOT NULL,
    last_event_seq BIGINT UNSIGNED NOT NULL,
    row_version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_npc_relationships_npc
        FOREIGN KEY (game_npc_id) REFERENCES game_npcs (id) ON DELETE RESTRICT,
    CONSTRAINT fk_npc_relationships_event
        FOREIGN KEY (source_event_id) REFERENCES game_events (id) ON DELETE RESTRICT,
    CONSTRAINT uq_npc_relationships_subject
        UNIQUE (game_npc_id, subject_type, subject_id),
    CONSTRAINT ck_npc_relationships_subject CHECK (
        subject_type IN ('player', 'character', 'faction', 'npc')
    ),
    CONSTRAINT ck_npc_relationships_attitude CHECK (attitude BETWEEN -100 AND 100),
    CONSTRAINT ck_npc_relationships_trust CHECK (trust BETWEEN -100 AND 100),
    CONSTRAINT ck_npc_relationships_fear CHECK (fear BETWEEN -100 AND 100),
    INDEX ix_npc_relationships_subject (subject_id),
    INDEX ix_npc_relationships_event (source_event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE npc_memories (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    game_npc_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    owner_actor_scope VARCHAR(96) NOT NULL,
    memory_type VARCHAR(32) NOT NULL,
    authority_class VARCHAR(24) NOT NULL,
    record_version INT UNSIGNED NOT NULL DEFAULT 1,
    importance TINYINT UNSIGNED NOT NULL,
    confidence DECIMAL(5,4) NOT NULL,
    summary_secret_payload_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    source_event_ids_json JSON NOT NULL,
    source_type VARCHAR(24) NOT NULL,
    occurred_at DATETIME(3) NOT NULL,
    valid_from DATETIME(3) NOT NULL,
    valid_until DATETIME(3) NULL,
    expires_at DATETIME(3) NULL,
    supersedes_memory_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    dedupe_key VARCHAR(160) NOT NULL,
    sensitivity TINYINT UNSIGNED NOT NULL,
    access_scope VARCHAR(32) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_npc_memories_game
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_npc_memories_npc_scope
        FOREIGN KEY (game_npc_id, game_session_id)
        REFERENCES game_npcs (id, game_session_id) ON DELETE RESTRICT,
    CONSTRAINT fk_npc_memories_secret
        FOREIGN KEY (summary_secret_payload_id) REFERENCES secret_payloads (id) ON DELETE RESTRICT,
    CONSTRAINT fk_npc_memories_supersedes
        FOREIGN KEY (supersedes_memory_id) REFERENCES npc_memories (id) ON DELETE RESTRICT,
    CONSTRAINT uq_npc_memories_dedupe
        UNIQUE (game_session_id, game_npc_id, dedupe_key),
    CONSTRAINT uq_npc_memories_secret UNIQUE (summary_secret_payload_id),
    CONSTRAINT ck_npc_memories_type CHECK (
        memory_type IN ('fact', 'dialog', 'inference', 'rumor', 'promise', 'emotion')
    ),
    CONSTRAINT ck_npc_memories_authority CHECK (
        authority_class IN ('fact', 'player_claim', 'inference', 'rumor')
    ),
    CONSTRAINT ck_npc_memories_importance CHECK (importance BETWEEN 0 AND 100),
    CONSTRAINT ck_npc_memories_confidence CHECK (confidence BETWEEN 0 AND 1),
    CONSTRAINT ck_npc_memories_sensitivity CHECK (sensitivity BETWEEN 0 AND 4),
    CONSTRAINT ck_npc_memories_validity CHECK (
        valid_until IS NULL OR valid_until >= valid_from
    ),
    INDEX ix_npc_memories_retrieval
        (game_session_id, game_npc_id, importance, occurred_at),
    INDEX ix_npc_memories_expiry (game_session_id, game_npc_id, expires_at),
    INDEX ix_npc_memories_owner (owner_actor_scope),
    INDEX ix_npc_memories_type (memory_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down

DROP TABLE IF EXISTS npc_memories;
DROP TABLE IF EXISTS npc_relationships;
DROP TABLE IF EXISTS game_npcs;
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS replay_reports;
DROP TABLE IF EXISTS game_results;
DROP TABLE IF EXISTS game_viewer_manifests;
DROP TABLE IF EXISTS game_snapshots;
DROP TABLE IF EXISTS game_timers;
DROP TABLE IF EXISTS game_messages;
DROP TABLE IF EXISTS game_search_claims;
DROP TABLE IF EXISTS game_votes;
DROP TABLE IF EXISTS game_events;
DROP TABLE IF EXISTS game_intents;
DROP TABLE IF EXISTS game_states;
DROP TABLE IF EXISTS game_roles;
DROP TABLE IF EXISTS game_players;
DROP TABLE IF EXISTS game_sessions;
DROP TABLE IF EXISTS room_messages;
DROP TABLE IF EXISTS room_events;
DROP TABLE IF EXISTS room_members;
DROP TABLE IF EXISTS invitation_tokens;
DROP TABLE IF EXISTS game_rooms;
