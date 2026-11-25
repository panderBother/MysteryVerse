-- +goose Up

CREATE TABLE moderation_cases (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    object_type VARCHAR(32) NOT NULL,
    object_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    source VARCHAR(24) NOT NULL,
    case_type VARCHAR(32) NOT NULL,
    severity TINYINT UNSIGNED NOT NULL,
    status VARCHAR(20) NOT NULL,
    decision VARCHAR(64) NULL,
    reason VARCHAR(2000) NULL,
    reviewer_user_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    second_reviewer_user_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    closed_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_moderation_cases_version
        FOREIGN KEY (script_version_id) REFERENCES script_versions (id) ON DELETE RESTRICT,
    CONSTRAINT fk_moderation_cases_reviewer
        FOREIGN KEY (reviewer_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT fk_moderation_cases_second_reviewer
        FOREIGN KEY (second_reviewer_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT ck_moderation_cases_source CHECK (
        source IN ('upload', 'generation', 'report', 'appeal')
    ),
    CONSTRAINT ck_moderation_cases_type CHECK (
        case_type IN ('content_rights', 'safety', 'abuse', 'quality')
    ),
    CONSTRAINT ck_moderation_cases_severity CHECK (severity BETWEEN 0 AND 4),
    CONSTRAINT ck_moderation_cases_status CHECK (
        status IN ('open', 'reviewing', 'approved', 'rejected', 'appealed', 'closed')
    ),
    INDEX ix_moderation_cases_object (object_type, object_id, status),
    INDEX ix_moderation_cases_status (status, severity, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE reports (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    reporter_user_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    object_type VARCHAR(32) NOT NULL,
    object_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    reason_code VARCHAR(48) NOT NULL,
    details VARCHAR(2000) NULL,
    evidence_json JSON NULL,
    status VARCHAR(20) NOT NULL,
    assigned_to CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_reports_reporter
        FOREIGN KEY (reporter_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT ck_reports_status CHECK (
        status IN ('open', 'triaged', 'resolved', 'rejected')
    ),
    INDEX ix_reports_object (object_type, object_id),
    INDEX ix_reports_reason_status (reason_code, status),
    INDEX ix_reports_queue (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE audit_logs (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    actor_type VARCHAR(24) NOT NULL,
    actor_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    action VARCHAR(96) NOT NULL,
    object_type VARCHAR(32) NOT NULL,
    object_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    scope_json JSON NULL,
    request_id VARCHAR(64) NULL,
    trace_id VARCHAR(64) NULL,
    before_digest BINARY(32) NULL,
    after_digest BINARY(32) NULL,
    reason VARCHAR(1000) NULL,
    ip_hash BINARY(32) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT ck_audit_logs_actor CHECK (
        actor_type IN ('user', 'service', 'admin', 'system')
    ),
    INDEX ix_audit_logs_actor (actor_type, actor_id, created_at),
    INDEX ix_audit_logs_action (action, created_at),
    INDEX ix_audit_logs_object (object_type, object_id, created_at),
    INDEX ix_audit_logs_trace (trace_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE user_sanctions (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    user_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    sanction_type VARCHAR(20) NOT NULL,
    scope VARCHAR(32) NOT NULL,
    scope_ref_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    reason_code VARCHAR(64) NOT NULL,
    reason_text VARCHAR(500) NOT NULL,
    created_by CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    starts_at DATETIME(3) NOT NULL,
    expires_at DATETIME(3) NULL,
    revoked_at DATETIME(3) NULL,
    revoked_by CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_user_sanctions_user
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT fk_user_sanctions_creator
        FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT fk_user_sanctions_revoker
        FOREIGN KEY (revoked_by) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT ck_user_sanctions_type CHECK (sanction_type IN ('mute', 'suspend')),
    CONSTRAINT ck_user_sanctions_time CHECK (
        expires_at IS NULL OR expires_at > starts_at
    ),
    CONSTRAINT ck_user_sanctions_revoke CHECK (
        revoked_at IS NULL OR revoked_by IS NOT NULL
    ),
    INDEX ix_user_sanctions_user (user_id),
    INDEX ix_user_sanctions_scope (scope),
    INDEX ix_user_sanctions_reason (reason_code),
    INDEX ix_user_sanctions_expiry (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE incidents (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    incident_type VARCHAR(48) NOT NULL,
    room_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    game_session_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    generation_job_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    severity TINYINT UNSIGNED NOT NULL,
    status VARCHAR(20) NOT NULL,
    trigger_summary VARCHAR(1000) NOT NULL,
    assigned_to CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    frozen_at DATETIME(3) NULL,
    resolution VARCHAR(2000) NULL,
    closed_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_incidents_room
        FOREIGN KEY (room_id) REFERENCES game_rooms (id) ON DELETE RESTRICT,
    CONSTRAINT fk_incidents_game
        FOREIGN KEY (game_session_id) REFERENCES game_sessions (id) ON DELETE RESTRICT,
    CONSTRAINT ck_incidents_severity CHECK (severity BETWEEN 0 AND 4),
    CONSTRAINT ck_incidents_status CHECK (
        status IN ('open', 'frozen', 'diagnosing', 'recovered', 'terminated')
    ),
    INDEX ix_incidents_type (incident_type),
    INDEX ix_incidents_room (room_id),
    INDEX ix_incidents_game (game_session_id),
    INDEX ix_incidents_queue (status, severity, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down

DROP TABLE IF EXISTS incidents;
DROP TABLE IF EXISTS user_sanctions;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS moderation_cases;
