-- +goose Up

CREATE TABLE users (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    public_no CHAR(12) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    status VARCHAR(20) NOT NULL,
    risk_level TINYINT UNSIGNED NOT NULL DEFAULT 0,
    terms_version VARCHAR(32) NOT NULL,
    privacy_version VARCHAR(32) NOT NULL,
    last_login_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT uq_users_public_no UNIQUE (public_no),
    CONSTRAINT ck_users_status CHECK (status IN ('pending', 'active', 'suspended', 'disabled', 'deleted')),
    INDEX ix_users_status_created (status, created_at),
    INDEX ix_users_last_login (last_login_at),
    INDEX ix_users_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE auth_identities (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    user_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    provider VARCHAR(32) NOT NULL,
    provider_subject VARCHAR(191) NOT NULL,
    identifier_hash BINARY(32) NOT NULL,
    identifier_hint VARCHAR(64) NOT NULL,
    verified_at DATETIME(3) NULL,
    is_primary TINYINT(1) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_auth_identities_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT uq_auth_identities_subject UNIQUE (provider, provider_subject),
    CONSTRAINT uq_auth_identities_hash UNIQUE (provider, identifier_hash),
    CONSTRAINT ck_auth_identities_status CHECK (status IN ('active', 'revoked', 'locked')),
    INDEX ix_auth_identities_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE user_sessions (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    user_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    session_token_hash BINARY(32) NOT NULL,
    csrf_token_hash BINARY(32) NOT NULL,
    device_label VARCHAR(128) NULL,
    ip_hash BINARY(32) NULL,
    user_agent_hash BINARY(32) NULL,
    expires_at DATETIME(3) NOT NULL,
    last_seen_at DATETIME(3) NOT NULL,
    revoked_at DATETIME(3) NULL,
    revoke_reason VARCHAR(64) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_user_sessions_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT uq_user_sessions_token UNIQUE (session_token_hash),
    INDEX ix_user_sessions_user_expires (user_id, expires_at),
    INDEX ix_user_sessions_expires (expires_at),
    INDEX ix_user_sessions_revoked (revoked_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE user_profiles (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    user_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    nickname VARCHAR(64) NOT NULL,
    avatar_uri VARCHAR(512) NULL,
    bio VARCHAR(500) NULL,
    locale VARCHAR(16) NOT NULL,
    timezone VARCHAR(64) NOT NULL,
    preferences_json JSON NOT NULL,
    privacy_json JSON NOT NULL,
    moderation_status VARCHAR(20) NOT NULL,
    row_version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_user_profiles_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT uq_user_profiles_user UNIQUE (user_id),
    CONSTRAINT ck_user_profiles_moderation CHECK (moderation_status IN ('pending', 'approved', 'rejected')),
    INDEX ix_user_profiles_nickname (nickname)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE user_statistics (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    user_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    games_started INT UNSIGNED NOT NULL DEFAULT 0,
    games_completed INT UNSIGNED NOT NULL DEFAULT 0,
    games_won INT UNSIGNED NOT NULL DEFAULT 0,
    script_murder_games INT UNSIGNED NOT NULL DEFAULT 0,
    werewolf_games INT UNSIGNED NOT NULL DEFAULT 0,
    rpg_games INT UNSIGNED NOT NULL DEFAULT 0,
    experience BIGINT UNSIGNED NOT NULL DEFAULT 0,
    level INT UNSIGNED NOT NULL DEFAULT 1,
    stats_json JSON NOT NULL,
    calculated_at DATETIME(3) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_user_statistics_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT uq_user_statistics_user UNIQUE (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE secret_payloads (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    scope_type VARCHAR(32) NOT NULL,
    scope_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    owner_type VARCHAR(32) NOT NULL,
    owner_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    purpose_code VARCHAR(48) NOT NULL,
    version_no INT UNSIGNED NOT NULL,
    sensitivity TINYINT UNSIGNED NOT NULL,
    visibility_scope VARCHAR(32) NOT NULL,
    schema_version VARCHAR(32) NOT NULL,
    key_id VARCHAR(128) NOT NULL,
    nonce VARBINARY(24) NOT NULL,
    ciphertext MEDIUMBLOB NOT NULL,
    content_hash BINARY(32) NOT NULL,
    compression VARCHAR(16) NOT NULL DEFAULT 'none',
    retention_class VARCHAR(32) NOT NULL,
    expires_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT uq_secret_payloads_scope UNIQUE (
        scope_type, scope_id, owner_type, owner_id, purpose_code, version_no
    ),
    CONSTRAINT ck_secret_payloads_sensitivity CHECK (sensitivity BETWEEN 2 AND 4),
    CONSTRAINT ck_secret_payloads_compression CHECK (compression IN ('none', 'gzip')),
    INDEX ix_secret_payloads_scope (scope_type, scope_id),
    INDEX ix_secret_payloads_owner (owner_type, owner_id),
    INDEX ix_secret_payloads_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE rule_templates (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    template_key VARCHAR(64) NOT NULL,
    version_no INT UNSIGNED NOT NULL,
    game_type VARCHAR(24) NOT NULL,
    name VARCHAR(160) NOT NULL,
    status VARCHAR(20) NOT NULL,
    schema_version VARCHAR(32) NOT NULL,
    engine_version VARCHAR(32) NOT NULL,
    config_json JSON NOT NULL,
    owner_user_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    content_hash BINARY(32) NOT NULL,
    published_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_rule_templates_owner FOREIGN KEY (owner_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT uq_rule_templates_version UNIQUE (template_key, version_no),
    CONSTRAINT uq_rule_templates_hash UNIQUE (content_hash),
    CONSTRAINT ck_rule_templates_status CHECK (status IN ('draft', 'active', 'retired')),
    INDEX ix_rule_templates_game_status (game_type, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE scripts (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    owner_user_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    public_slug VARCHAR(96) NULL,
    title VARCHAR(160) NOT NULL,
    summary VARCHAR(1000) NULL,
    game_type VARCHAR(24) NOT NULL,
    source_kind VARCHAR(24) NOT NULL,
    rights_type VARCHAR(24) NOT NULL,
    status VARCHAR(24) NOT NULL,
    cover_asset_uri VARCHAR(1024) NULL,
    language VARCHAR(16) NOT NULL,
    content_rating VARCHAR(16) NOT NULL,
    min_players TINYINT UNSIGNED NOT NULL,
    max_players TINYINT UNSIGNED NOT NULL,
    duration_minutes SMALLINT UNSIGNED NOT NULL,
    difficulty VARCHAR(16) NOT NULL,
    ai_assisted_ratio_bp SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    active_draft_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    published_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    deleted_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_scripts_owner FOREIGN KEY (owner_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT uq_scripts_public_slug UNIQUE (public_slug),
    CONSTRAINT ck_scripts_game_type CHECK (game_type IN ('script_murder', 'werewolf', 'rpg')),
    CONSTRAINT ck_scripts_status CHECK (status IN ('draft', 'validating', 'review', 'published', 'unlisted', 'soft_deleted')),
    CONSTRAINT ck_scripts_players CHECK (min_players <= max_players),
    CONSTRAINT ck_scripts_ai_ratio CHECK (ai_assisted_ratio_bp BETWEEN 0 AND 10000),
    INDEX ix_scripts_title (title),
    INDEX ix_scripts_game_status (game_type, status),
    INDEX ix_scripts_published_version (published_version_id),
    INDEX ix_scripts_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE script_versions (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    script_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    version_no INT UNSIGNED NOT NULL,
    version_kind VARCHAR(20) NOT NULL,
    status VARCHAR(24) NOT NULL,
    schema_version VARCHAR(32) NOT NULL,
    workflow_version VARCHAR(32) NOT NULL,
    base_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    rule_template_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    blueprint_json JSON NOT NULL,
    package_uri VARCHAR(1024) NULL,
    package_key_hash BINARY(32) NULL,
    package_hash BINARY(32) NULL,
    manifest_json JSON NULL,
    validation_report_uri VARCHAR(1024) NULL,
    validation_status VARCHAR(24) NOT NULL,
    blocking_issue_count INT UNSIGNED NOT NULL DEFAULT 0,
    created_by CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    frozen_at DATETIME(3) NULL,
    published_at DATETIME(3) NULL,
    row_version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_script_versions_script FOREIGN KEY (script_id) REFERENCES scripts (id) ON DELETE RESTRICT,
    CONSTRAINT fk_script_versions_base FOREIGN KEY (base_version_id) REFERENCES script_versions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_script_versions_rule FOREIGN KEY (rule_template_id) REFERENCES rule_templates (id) ON DELETE RESTRICT,
    CONSTRAINT fk_script_versions_creator FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT uq_script_versions_number UNIQUE (script_id, version_no),
    CONSTRAINT uq_script_versions_key_hash UNIQUE (package_key_hash),
    CONSTRAINT ck_script_versions_kind CHECK (version_kind IN ('draft', 'release', 'rollback')),
    CONSTRAINT ck_script_versions_status CHECK (
        status IN ('draft', 'validating', 'review', 'published_private', 'published_public', 'unlisted', 'archived')
    ),
    CONSTRAINT ck_script_versions_validation CHECK (validation_status IN ('pending', 'passed', 'warning', 'blocked')),
    INDEX ix_script_versions_script_status (script_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE script_characters (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    character_key VARCHAR(96) NOT NULL,
    display_name VARCHAR(96) NOT NULL,
    role_class VARCHAR(24) NOT NULL,
    public_profile TEXT NOT NULL,
    public_goals_json JSON NOT NULL,
    attributes_json JSON NOT NULL,
    private_payload_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    visibility_scope VARCHAR(32) NOT NULL,
    sensitivity TINYINT UNSIGNED NOT NULL,
    source_refs_json JSON NULL,
    generated_origin VARCHAR(24) NOT NULL,
    locked_fields_json JSON NULL,
    sort_order INT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_script_characters_version FOREIGN KEY (script_version_id) REFERENCES script_versions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_script_characters_secret FOREIGN KEY (private_payload_id) REFERENCES secret_payloads (id) ON DELETE RESTRICT,
    CONSTRAINT uq_script_characters_key UNIQUE (script_version_id, character_key),
    CONSTRAINT ck_script_characters_role_class CHECK (role_class IN ('player_role', 'npc', 'faction_leader')),
    CONSTRAINT ck_script_characters_sensitivity CHECK (sensitivity BETWEEN 0 AND 4),
    INDEX ix_script_characters_display (script_version_id, display_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE script_scenes (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    scene_key VARCHAR(96) NOT NULL,
    scene_type VARCHAR(24) NOT NULL,
    title VARCHAR(160) NOT NULL,
    public_description TEXT NOT NULL,
    entry_conditions_json JSON NULL,
    visibility_scope VARCHAR(32) NOT NULL,
    sensitivity TINYINT UNSIGNED NOT NULL,
    private_payload_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    assets_json JSON NULL,
    metadata_json JSON NULL,
    is_start TINYINT(1) NOT NULL DEFAULT 0,
    sort_order INT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_script_scenes_version FOREIGN KEY (script_version_id) REFERENCES script_versions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_script_scenes_secret FOREIGN KEY (private_payload_id) REFERENCES secret_payloads (id) ON DELETE RESTRICT,
    CONSTRAINT uq_script_scenes_key UNIQUE (script_version_id, scene_key),
    CONSTRAINT ck_script_scenes_sensitivity CHECK (sensitivity BETWEEN 0 AND 4),
    INDEX ix_script_scenes_type (scene_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE npc_profiles (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    npc_key VARCHAR(96) NOT NULL,
    display_name VARCHAR(96) NOT NULL,
    public_profile TEXT NOT NULL,
    persona_json JSON NOT NULL,
    goals_json JSON NOT NULL,
    knowledge_scope_json JSON NOT NULL,
    initial_relationship_json JSON NULL,
    visibility_scope VARCHAR(32) NOT NULL,
    sensitivity TINYINT UNSIGNED NOT NULL,
    secret_payload_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    tool_policy_json JSON NOT NULL,
    sort_order INT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_npc_profiles_version FOREIGN KEY (script_version_id) REFERENCES script_versions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_npc_profiles_secret FOREIGN KEY (secret_payload_id) REFERENCES secret_payloads (id) ON DELETE RESTRICT,
    CONSTRAINT uq_npc_profiles_key UNIQUE (script_version_id, npc_key),
    CONSTRAINT ck_npc_profiles_sensitivity CHECK (sensitivity BETWEEN 0 AND 4)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down

DROP TABLE IF EXISTS npc_profiles;
DROP TABLE IF EXISTS script_scenes;
DROP TABLE IF EXISTS script_characters;
DROP TABLE IF EXISTS script_versions;
DROP TABLE IF EXISTS scripts;
DROP TABLE IF EXISTS rule_templates;
DROP TABLE IF EXISTS secret_payloads;
DROP TABLE IF EXISTS user_statistics;
DROP TABLE IF EXISTS user_profiles;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS auth_identities;
DROP TABLE IF EXISTS users;
