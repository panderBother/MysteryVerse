-- +goose Up

ALTER TABLE auth_identities
    ADD COLUMN password_hash VARCHAR(255) NULL AFTER identifier_hint;

ALTER TABLE room_members
    DROP INDEX uq_room_members_seat,
    ADD COLUMN active_seat_no TINYINT UNSIGNED
        GENERATED ALWAYS AS (
            CASE
                WHEN left_at IS NULL THEN seat_no
                ELSE NULL
            END
        ) STORED,
    ADD CONSTRAINT uq_room_members_active_seat UNIQUE (room_id, active_seat_no),
    ADD INDEX ix_room_members_seat_history (room_id, seat_no);

ALTER TABLE room_messages
    ADD COLUMN event_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL AFTER id,
    ADD COLUMN idempotency_key_hash BINARY(32) NOT NULL AFTER client_message_id,
    ADD COLUMN schema_version SMALLINT UNSIGNED NOT NULL DEFAULT 1 AFTER moderation_status,
    ADD CONSTRAINT uq_room_messages_idempotency
        UNIQUE (room_id, sender_user_id, idempotency_key_hash),
    ADD CONSTRAINT uq_room_messages_event UNIQUE (event_id);

CREATE TABLE idempotency_records (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    scope VARCHAR(64) NOT NULL,
    actor_scope VARCHAR(96) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    request_hash BINARY(32) NOT NULL,
    status VARCHAR(16) NOT NULL,
    response_status SMALLINT UNSIGNED NULL,
    response_json JSON NULL,
    resource_type VARCHAR(64) NULL,
    resource_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    expires_at DATETIME(3) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT uq_idempotency_records_key
        UNIQUE (scope, actor_scope, idempotency_key),
    CONSTRAINT ck_idempotency_records_status CHECK (
        status IN ('pending', 'completed', 'failed')
    ),
    INDEX ix_idempotency_records_expiry (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down

DROP TABLE IF EXISTS idempotency_records;

ALTER TABLE room_messages
    DROP INDEX uq_room_messages_event,
    DROP INDEX uq_room_messages_idempotency,
    DROP COLUMN schema_version,
    DROP COLUMN idempotency_key_hash,
    DROP COLUMN event_id;

ALTER TABLE room_members
    DROP INDEX ix_room_members_seat_history,
    DROP INDEX uq_room_members_active_seat,
    DROP COLUMN active_seat_no,
    ADD CONSTRAINT uq_room_members_seat UNIQUE (room_id, seat_no);

ALTER TABLE auth_identities
    DROP COLUMN password_hash;
