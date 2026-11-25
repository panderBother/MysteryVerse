-- +goose Up

CREATE TABLE script_phases (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    phase_key VARCHAR(96) NOT NULL,
    phase_type VARCHAR(24) NOT NULL,
    sequence_no INT UNSIGNED NOT NULL,
    title VARCHAR(160) NOT NULL,
    entry_condition_json JSON NULL,
    exit_condition_json JSON NULL,
    timeout_seconds INT UNSIGNED NULL,
    fallback_action VARCHAR(64) NOT NULL,
    allowed_actions_json JSON NOT NULL,
    visibility_scope VARCHAR(32) NOT NULL,
    is_terminal TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_script_phases_version
        FOREIGN KEY (script_version_id) REFERENCES script_versions (id) ON DELETE RESTRICT,
    CONSTRAINT uq_script_phases_key UNIQUE (script_version_id, phase_key),
    CONSTRAINT uq_script_phases_sequence UNIQUE (script_version_id, sequence_no),
    CONSTRAINT ck_script_phases_type CHECK (
        phase_type IN ('read', 'search', 'discussion', 'vote', 'night', 'settlement')
    ),
    INDEX ix_script_phases_type (phase_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE script_reveal_gates (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    gate_key VARCHAR(96) NOT NULL,
    phase_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    condition_json JSON NOT NULL,
    applies_to_object_type VARCHAR(32) NULL,
    applies_to_object_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    default_open TINYINT(1) NOT NULL DEFAULT 0,
    sensitivity TINYINT UNSIGNED NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_script_reveal_gates_version
        FOREIGN KEY (script_version_id) REFERENCES script_versions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_script_reveal_gates_phase
        FOREIGN KEY (phase_id) REFERENCES script_phases (id) ON DELETE RESTRICT,
    CONSTRAINT uq_script_reveal_gates_key UNIQUE (script_version_id, gate_key),
    CONSTRAINT ck_script_reveal_gates_sensitivity CHECK (sensitivity BETWEEN 0 AND 4)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE script_clues (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    clue_key VARCHAR(96) NOT NULL,
    title VARCHAR(160) NOT NULL,
    public_text TEXT NULL,
    clue_type VARCHAR(24) NOT NULL,
    is_repeatable TINYINT(1) NOT NULL DEFAULT 0,
    max_discoveries INT UNSIGNED NULL,
    visibility_scope VARCHAR(32) NOT NULL,
    sensitivity TINYINT UNSIGNED NOT NULL,
    reveal_gate_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    private_payload_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    related_fact_ids_json JSON NULL,
    source_refs_json JSON NULL,
    sort_order INT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_script_clues_version
        FOREIGN KEY (script_version_id) REFERENCES script_versions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_script_clues_reveal_gate
        FOREIGN KEY (reveal_gate_id) REFERENCES script_reveal_gates (id) ON DELETE RESTRICT,
    CONSTRAINT fk_script_clues_secret
        FOREIGN KEY (private_payload_id) REFERENCES secret_payloads (id) ON DELETE RESTRICT,
    CONSTRAINT uq_script_clues_key UNIQUE (script_version_id, clue_key),
    CONSTRAINT ck_script_clues_type CHECK (
        clue_type IN ('public', 'private', 'physical', 'testimony')
    ),
    CONSTRAINT ck_script_clues_sensitivity CHECK (sensitivity BETWEEN 0 AND 4),
    CONSTRAINT ck_script_clues_discoveries CHECK (
        max_discoveries IS NULL OR max_discoveries > 0
    ),
    INDEX ix_script_clues_visibility (script_version_id, visibility_scope)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE script_search_points (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    point_key VARCHAR(96) NOT NULL,
    phase_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    scene_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    title VARCHAR(160) NOT NULL,
    public_description VARCHAR(1000) NOT NULL,
    max_searches INT UNSIGNED NOT NULL,
    conflict_policy VARCHAR(24) NOT NULL,
    visibility_scope VARCHAR(32) NOT NULL,
    reveal_gate_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    metadata_json JSON NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_script_search_points_version
        FOREIGN KEY (script_version_id) REFERENCES script_versions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_script_search_points_phase
        FOREIGN KEY (phase_id) REFERENCES script_phases (id) ON DELETE RESTRICT,
    CONSTRAINT fk_script_search_points_scene
        FOREIGN KEY (scene_id) REFERENCES script_scenes (id) ON DELETE RESTRICT,
    CONSTRAINT fk_script_search_points_reveal_gate
        FOREIGN KEY (reveal_gate_id) REFERENCES script_reveal_gates (id) ON DELETE RESTRICT,
    CONSTRAINT uq_script_search_points_key UNIQUE (script_version_id, point_key),
    CONSTRAINT ck_script_search_points_policy CHECK (
        conflict_policy IN ('first_wins', 'repeatable', 'queue')
    ),
    INDEX ix_script_search_points_phase (phase_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE script_search_point_clues (
    script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    search_point_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    clue_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    weight INT UNSIGNED NOT NULL DEFAULT 1,
    is_default TINYINT(1) NOT NULL DEFAULT 0,
    metadata_json JSON NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (search_point_id, clue_id),
    CONSTRAINT fk_script_search_point_clues_version
        FOREIGN KEY (script_version_id) REFERENCES script_versions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_script_search_point_clues_point
        FOREIGN KEY (search_point_id) REFERENCES script_search_points (id) ON DELETE RESTRICT,
    CONSTRAINT fk_script_search_point_clues_clue
        FOREIGN KEY (clue_id) REFERENCES script_clues (id) ON DELETE RESTRICT,
    CONSTRAINT ck_script_search_point_clues_weight CHECK (weight > 0),
    INDEX ix_script_search_point_clues_clue (clue_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down

DROP TABLE IF EXISTS script_search_point_clues;
DROP TABLE IF EXISTS script_search_points;
DROP TABLE IF EXISTS script_clues;
DROP TABLE IF EXISTS script_reveal_gates;
DROP TABLE IF EXISTS script_phases;
