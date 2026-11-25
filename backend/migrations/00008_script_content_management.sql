-- +goose Up

ALTER TABLE scripts
    ADD COLUMN row_version INT UNSIGNED NOT NULL DEFAULT 1;

ALTER TABLE script_versions
    ADD COLUMN validation_report_json JSON NULL;

CREATE TABLE script_chapters (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    chapter_key VARCHAR(96) NOT NULL,
    title VARCHAR(160) NOT NULL,
    summary VARCHAR(1000) NULL,
    sequence_no INT UNSIGNED NOT NULL,
    source_refs_json JSON NOT NULL,
    row_version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_script_chapters_version
        FOREIGN KEY (script_version_id) REFERENCES script_versions (id)
        ON DELETE RESTRICT,
    CONSTRAINT uq_script_chapters_key
        UNIQUE (script_version_id, chapter_key),
    CONSTRAINT uq_script_chapters_sequence
        UNIQUE (script_version_id, sequence_no),
    INDEX ix_script_chapters_version (script_version_id, sequence_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

ALTER TABLE script_phases
    ADD COLUMN chapter_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NULL,
    ADD COLUMN row_version INT UNSIGNED NOT NULL DEFAULT 1,
    ADD CONSTRAINT fk_script_phases_chapter
        FOREIGN KEY (chapter_id) REFERENCES script_chapters (id)
        ON DELETE RESTRICT,
    ADD INDEX ix_script_phases_chapter (chapter_id);

ALTER TABLE script_characters
    ADD COLUMN row_version INT UNSIGNED NOT NULL DEFAULT 1;

ALTER TABLE script_clues
    ADD COLUMN row_version INT UNSIGNED NOT NULL DEFAULT 1;

ALTER TABLE script_search_points
    ADD COLUMN row_version INT UNSIGNED NOT NULL DEFAULT 1;

CREATE TABLE script_endings (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    ending_key VARCHAR(96) NOT NULL,
    title VARCHAR(160) NOT NULL,
    description TEXT NOT NULL,
    conditions_json JSON NOT NULL,
    is_hidden TINYINT(1) NOT NULL DEFAULT 1,
    visibility_scope VARCHAR(32) NOT NULL,
    sort_order INT UNSIGNED NOT NULL DEFAULT 0,
    row_version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_script_endings_version
        FOREIGN KEY (script_version_id) REFERENCES script_versions (id)
        ON DELETE RESTRICT,
    CONSTRAINT uq_script_endings_key
        UNIQUE (script_version_id, ending_key),
    CONSTRAINT ck_script_endings_visibility CHECK (
        visibility_scope IN ('public', 'room', 'role', 'runtime')
    ),
    INDEX ix_script_endings_version (script_version_id, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE script_assets (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    asset_key VARCHAR(96) NOT NULL,
    asset_type VARCHAR(32) NOT NULL,
    uri VARCHAR(1024) NOT NULL,
    mime_type VARCHAR(160) NULL,
    sha256 BINARY(32) NULL,
    byte_size BIGINT UNSIGNED NULL,
    visibility_scope VARCHAR(32) NOT NULL,
    sensitivity TINYINT UNSIGNED NOT NULL,
    metadata_json JSON NULL,
    sort_order INT UNSIGNED NOT NULL DEFAULT 0,
    row_version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_script_assets_version
        FOREIGN KEY (script_version_id) REFERENCES script_versions (id)
        ON DELETE RESTRICT,
    CONSTRAINT uq_script_assets_key
        UNIQUE (script_version_id, asset_key),
    CONSTRAINT ck_script_assets_sensitivity CHECK (sensitivity BETWEEN 0 AND 4),
    INDEX ix_script_assets_version (script_version_id, asset_type, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE script_publications (
    id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    script_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    script_version_id CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    channel VARCHAR(24) NOT NULL,
    status VARCHAR(24) NOT NULL,
    active_publication_key VARCHAR(96) NULL,
    submitted_by CHAR(26) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    published_at DATETIME(3) NOT NULL,
    unpublished_at DATETIME(3) NULL,
    reason_code VARCHAR(64) NULL,
    reason_text VARCHAR(1000) NULL,
    row_version INT UNSIGNED NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_script_publications_script
        FOREIGN KEY (script_id) REFERENCES scripts (id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_script_publications_version
        FOREIGN KEY (script_version_id) REFERENCES script_versions (id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_script_publications_submitter
        FOREIGN KEY (submitted_by) REFERENCES users (id)
        ON DELETE RESTRICT,
    CONSTRAINT uq_script_publications_channel
        UNIQUE (script_id, script_version_id, channel),
    CONSTRAINT uq_script_publications_active
        UNIQUE (active_publication_key),
    CONSTRAINT ck_script_publications_channel CHECK (
        channel IN ('private', 'invite', 'public')
    ),
    CONSTRAINT ck_script_publications_status CHECK (
        status IN ('active', 'unpublished', 'rejected')
    ),
    INDEX ix_script_publications_status (status, published_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down

DROP TABLE IF EXISTS script_publications;
DROP TABLE IF EXISTS script_assets;
DROP TABLE IF EXISTS script_endings;

ALTER TABLE script_search_points
    DROP COLUMN row_version;

ALTER TABLE script_clues
    DROP COLUMN row_version;

ALTER TABLE script_characters
    DROP COLUMN row_version;

ALTER TABLE script_phases
    DROP INDEX ix_script_phases_chapter,
    DROP FOREIGN KEY fk_script_phases_chapter,
    DROP COLUMN row_version,
    DROP COLUMN chapter_id;

DROP TABLE IF EXISTS script_chapters;

ALTER TABLE script_versions
    DROP COLUMN validation_report_json;

ALTER TABLE scripts
    DROP COLUMN row_version;
