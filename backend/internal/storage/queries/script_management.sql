-- name: GetScriptForUpdate :one
SELECT *
FROM scripts
WHERE id = ?
  AND deleted_at IS NULL
FOR UPDATE;

-- name: GetScriptAny :one
SELECT *
FROM scripts
WHERE id = ?;

-- name: ListPublicScripts :many
SELECT
    s.id AS script_id,
    s.owner_user_id,
    s.title,
    s.summary,
    s.game_type,
    s.cover_asset_uri,
    s.language,
    s.content_rating,
    s.min_players,
    s.max_players,
    s.duration_minutes,
    s.difficulty,
    sv.id AS script_version_id,
    sv.version_no,
    sv.schema_version,
    s.created_at,
    s.updated_at
FROM scripts s
JOIN script_versions sv ON sv.id = s.published_version_id
JOIN script_publications p
  ON p.script_id = s.id
 AND p.script_version_id = sv.id
 AND p.channel = 'public'
 AND p.status = 'active'
WHERE s.deleted_at IS NULL
  AND s.status = 'published'
  AND sv.status = 'published_public'
  AND sv.validation_status IN ('passed', 'warning')
  AND (? = '' OR s.title LIKE CONCAT('%', ?, '%'))
  AND (? = '' OR s.game_type = ?)
  AND (? = 0 OR s.min_players >= ?)
  AND (? = 0 OR s.max_players <= ?)
  AND (? = 0 OR s.duration_minutes <= ?)
  AND (? = '' OR s.difficulty = ?)
  AND (? = '' OR s.id < ?)
ORDER BY s.id DESC
LIMIT ?;

-- name: GetPublicScriptDetail :one
SELECT
    s.id AS script_id,
    s.owner_user_id,
    s.title,
    s.summary,
    s.game_type,
    s.status AS script_status,
    s.cover_asset_uri,
    s.language,
    s.content_rating,
    s.min_players,
    s.max_players,
    s.duration_minutes,
    s.difficulty,
    s.published_version_id,
    sv.id AS script_version_id,
    sv.version_no,
    sv.version_kind,
    sv.status AS version_status,
    sv.schema_version,
    sv.validation_status,
    sv.blocking_issue_count,
    sv.frozen_at,
    sv.published_at,
    sv.row_version AS version_row_version,
    s.created_at,
    s.updated_at
FROM scripts s
JOIN script_versions sv ON sv.id = s.published_version_id
JOIN script_publications p
  ON p.script_id = s.id
 AND p.script_version_id = sv.id
 AND p.channel = 'public'
 AND p.status = 'active'
WHERE s.id = ?
  AND s.deleted_at IS NULL
  AND s.status = 'published'
  AND sv.status = 'published_public';

-- name: UpdateScriptMetadata :execrows
UPDATE scripts
SET title = ?,
    summary = ?,
    cover_asset_uri = ?,
    language = ?,
    content_rating = ?,
    min_players = ?,
    max_players = ?,
    duration_minutes = ?,
    difficulty = ?,
    row_version = row_version + 1
WHERE id = ?
  AND owner_user_id = ?
  AND row_version = ?
  AND deleted_at IS NULL;

-- name: SoftDeleteScript :execrows
UPDATE scripts
SET status = 'soft_deleted',
    active_draft_version_id = NULL,
    published_version_id = NULL,
    deleted_at = ?,
    row_version = row_version + 1
WHERE id = ?
  AND owner_user_id = ?
  AND row_version = ?
  AND deleted_at IS NULL
  AND published_version_id IS NULL;

-- name: SetScriptPublishedVersion :execrows
UPDATE scripts
SET status = 'published',
    published_version_id = ?,
    row_version = row_version + 1
WHERE id = ?
  AND row_version = ?;

-- name: SetScriptActiveDraftVersion :execrows
UPDATE scripts
SET active_draft_version_id = ?,
    row_version = row_version + 1
WHERE id = ?
  AND row_version = ?;

-- name: SetScriptUnlisted :execrows
UPDATE scripts
SET status = 'unlisted',
    published_version_id = NULL,
    row_version = row_version + 1
WHERE id = ?
  AND published_version_id = ?
  AND row_version = ?;

-- name: GetScriptVersion :one
SELECT *
FROM script_versions
WHERE id = ?;

-- name: GetScriptVersionForUpdate :one
SELECT *
FROM script_versions
WHERE id = ?
FOR UPDATE;

-- name: ListScriptVersions :many
SELECT *
FROM script_versions
WHERE script_id = ?
  AND (? = '' OR status = ?)
ORDER BY version_no DESC
LIMIT ?;

-- name: GetNextScriptVersionNo :one
SELECT COALESCE(MAX(version_no), 0) + 1
FROM script_versions
WHERE script_id = ?;

-- name: CreateScriptVersion :execrows
INSERT INTO script_versions (
    id,
    script_id,
    version_no,
    version_kind,
    status,
    schema_version,
    workflow_version,
    base_version_id,
    rule_template_id,
    blueprint_json,
    validation_status,
    blocking_issue_count,
    created_by
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
    'pending',
    0,
    ?
);

-- name: UpdateScriptDraftValidation :execrows
UPDATE script_versions
SET validation_status = ?,
    blocking_issue_count = ?,
    validation_report_json = ?,
    row_version = row_version + 1
WHERE id = ?
  AND status IN ('draft', 'validating', 'review')
  AND row_version = ?;

-- name: BumpScriptDraftRowVersion :execrows
UPDATE script_versions
SET validation_status = 'pending',
    blocking_issue_count = 0,
    validation_report_json = NULL,
    row_version = row_version + 1
WHERE id = ?
  AND script_id = ?
  AND status = 'draft'
  AND row_version = ?;

-- name: PublishScriptVersion :execrows
UPDATE script_versions
SET status = ?,
    version_kind = 'release',
    validation_status = ?,
    blocking_issue_count = 0,
    validation_report_json = ?,
    frozen_at = ?,
    published_at = ?,
    row_version = row_version + 1
WHERE id = ?
  AND script_id = ?
  AND status IN ('draft', 'validating', 'review')
  AND row_version = ?;

-- name: MarkScriptVersionUnlisted :execrows
UPDATE script_versions
SET status = 'unlisted',
    row_version = row_version + 1
WHERE id = ?
  AND status IN ('published_private', 'published_public');

-- name: GetRuleTemplateByID :one
SELECT *
FROM rule_templates
WHERE id = ?;

-- name: GetActiveRuleTemplateForScript :one
SELECT *
FROM rule_templates
WHERE id = ?
  AND status = 'active'
  AND game_type = ?;

-- name: CreateScriptPublication :execrows
INSERT INTO script_publications (
    id,
    script_id,
    script_version_id,
    channel,
    status,
    active_publication_key,
    submitted_by,
    published_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    'active',
    ?,
    ?,
    ?
);

-- name: GetScriptPublicationForUpdate :one
SELECT *
FROM script_publications
WHERE id = ?
FOR UPDATE;

-- name: GetScriptPublicationByChannel :one
SELECT *
FROM script_publications
WHERE script_id = ?
  AND script_version_id = ?
  AND channel = ?;

-- name: SupersedeActiveScriptPublications :execrows
UPDATE script_publications
SET status = 'unpublished',
    active_publication_key = NULL,
    unpublished_at = ?,
    row_version = row_version + 1
WHERE script_id = ?
  AND channel = ?
  AND status = 'active';

-- name: ReactivateScriptPublication :execrows
UPDATE script_publications
SET status = 'active',
    active_publication_key = ?,
    published_at = ?,
    unpublished_at = NULL,
    reason_code = NULL,
    reason_text = NULL,
    row_version = row_version + 1
WHERE id = ?;

-- name: UnpublishScriptPublication :execrows
UPDATE script_publications
SET status = 'unpublished',
    active_publication_key = NULL,
    unpublished_at = ?,
    reason_code = ?,
    reason_text = ?,
    row_version = row_version + 1
WHERE id = ?
  AND status = 'active';

-- name: ListScriptChapters :many
SELECT *
FROM script_chapters
WHERE script_version_id = ?
ORDER BY sequence_no, id;

-- name: GetScriptChapter :one
SELECT *
FROM script_chapters
WHERE id = ?
  AND script_version_id = ?;

-- name: CreateScriptChapter :execrows
INSERT INTO script_chapters (
    id,
    script_version_id,
    chapter_key,
    title,
    summary,
    sequence_no,
    source_refs_json
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
);

-- name: UpdateScriptChapter :execrows
UPDATE script_chapters
SET chapter_key = ?,
    title = ?,
    summary = ?,
    sequence_no = ?,
    source_refs_json = ?,
    row_version = row_version + 1
WHERE id = ?
  AND script_version_id = ?;

-- name: DeleteScriptChapter :execrows
DELETE FROM script_chapters
WHERE id = ?
  AND script_version_id = ?;

-- name: ListScriptStages :many
SELECT *
FROM script_phases
WHERE script_version_id = ?
ORDER BY sequence_no, id;

-- name: GetScriptStage :one
SELECT *
FROM script_phases
WHERE id = ?
  AND script_version_id = ?;

-- name: CreateScriptStage :execrows
INSERT INTO script_phases (
    id,
    script_version_id,
    chapter_id,
    phase_key,
    phase_type,
    sequence_no,
    title,
    entry_condition_json,
    exit_condition_json,
    timeout_seconds,
    fallback_action,
    allowed_actions_json,
    visibility_scope,
    is_terminal
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
    ?
);

-- name: UpdateScriptStage :execrows
UPDATE script_phases
SET chapter_id = ?,
    phase_key = ?,
    phase_type = ?,
    sequence_no = ?,
    title = ?,
    entry_condition_json = ?,
    exit_condition_json = ?,
    timeout_seconds = ?,
    fallback_action = ?,
    allowed_actions_json = ?,
    visibility_scope = ?,
    is_terminal = ?,
    row_version = row_version + 1
WHERE id = ?
  AND script_version_id = ?;

-- name: DeleteScriptStage :execrows
DELETE FROM script_phases
WHERE id = ?
  AND script_version_id = ?;

-- name: ListScriptCharacters :many
SELECT *
FROM script_characters
WHERE script_version_id = ?
ORDER BY sort_order, id;

-- name: GetScriptCharacter :one
SELECT *
FROM script_characters
WHERE id = ?
  AND script_version_id = ?;

-- name: CreateScriptCharacter :execrows
INSERT INTO script_characters (
    id,
    script_version_id,
    character_key,
    display_name,
    role_class,
    public_profile,
    public_goals_json,
    attributes_json,
    private_payload_id,
    visibility_scope,
    sensitivity,
    source_refs_json,
    generated_origin,
    locked_fields_json,
    sort_order
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
    ?
);

-- name: UpdateScriptCharacter :execrows
UPDATE script_characters
SET character_key = ?,
    display_name = ?,
    role_class = ?,
    public_profile = ?,
    public_goals_json = ?,
    attributes_json = ?,
    private_payload_id = ?,
    visibility_scope = ?,
    sensitivity = ?,
    source_refs_json = ?,
    generated_origin = ?,
    locked_fields_json = ?,
    sort_order = ?,
    row_version = row_version + 1
WHERE id = ?
  AND script_version_id = ?;

-- name: DeleteScriptCharacter :execrows
DELETE FROM script_characters
WHERE id = ?
  AND script_version_id = ?;

-- name: ListScriptClues :many
SELECT *
FROM script_clues
WHERE script_version_id = ?
ORDER BY sort_order, id;

-- name: GetScriptClue :one
SELECT *
FROM script_clues
WHERE id = ?
  AND script_version_id = ?;

-- name: CreateScriptClue :execrows
INSERT INTO script_clues (
    id,
    script_version_id,
    clue_key,
    title,
    public_text,
    clue_type,
    is_repeatable,
    max_discoveries,
    visibility_scope,
    sensitivity,
    reveal_gate_id,
    private_payload_id,
    related_fact_ids_json,
    source_refs_json,
    sort_order
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
    ?
);

-- name: UpdateScriptClue :execrows
UPDATE script_clues
SET clue_key = ?,
    title = ?,
    public_text = ?,
    clue_type = ?,
    is_repeatable = ?,
    max_discoveries = ?,
    visibility_scope = ?,
    sensitivity = ?,
    reveal_gate_id = ?,
    private_payload_id = ?,
    related_fact_ids_json = ?,
    source_refs_json = ?,
    sort_order = ?,
    row_version = row_version + 1
WHERE id = ?
  AND script_version_id = ?;

-- name: DeleteScriptClue :execrows
DELETE FROM script_clues
WHERE id = ?
  AND script_version_id = ?;

-- name: ListScriptEndings :many
SELECT *
FROM script_endings
WHERE script_version_id = ?
ORDER BY sort_order, id;

-- name: GetScriptEnding :one
SELECT *
FROM script_endings
WHERE id = ?
  AND script_version_id = ?;

-- name: CreateScriptEnding :execrows
INSERT INTO script_endings (
    id,
    script_version_id,
    ending_key,
    title,
    description,
    conditions_json,
    is_hidden,
    visibility_scope,
    sort_order
) VALUES (
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

-- name: UpdateScriptEnding :execrows
UPDATE script_endings
SET ending_key = ?,
    title = ?,
    description = ?,
    conditions_json = ?,
    is_hidden = ?,
    visibility_scope = ?,
    sort_order = ?,
    row_version = row_version + 1
WHERE id = ?
  AND script_version_id = ?;

-- name: DeleteScriptEnding :execrows
DELETE FROM script_endings
WHERE id = ?
  AND script_version_id = ?;

-- name: ListScriptAssets :many
SELECT *
FROM script_assets
WHERE script_version_id = ?
ORDER BY sort_order, id;

-- name: GetScriptAsset :one
SELECT *
FROM script_assets
WHERE id = ?
  AND script_version_id = ?;

-- name: CreateScriptAsset :execrows
INSERT INTO script_assets (
    id,
    script_version_id,
    asset_key,
    asset_type,
    uri,
    mime_type,
    sha256,
    byte_size,
    visibility_scope,
    sensitivity,
    metadata_json,
    sort_order
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
    ?
);

-- name: UpdateScriptAsset :execrows
UPDATE script_assets
SET asset_key = ?,
    asset_type = ?,
    uri = ?,
    mime_type = ?,
    sha256 = ?,
    byte_size = ?,
    visibility_scope = ?,
    sensitivity = ?,
    metadata_json = ?,
    sort_order = ?,
    row_version = row_version + 1
WHERE id = ?
  AND script_version_id = ?;

-- name: DeleteScriptAsset :execrows
DELETE FROM script_assets
WHERE id = ?
  AND script_version_id = ?;

-- name: ListScriptSearchPoints :many
SELECT *
FROM script_search_points
WHERE script_version_id = ?
ORDER BY id;

-- name: GetScriptSearchPoint :one
SELECT *
FROM script_search_points
WHERE id = ?
  AND script_version_id = ?;

-- name: CreateScriptSearchPoint :execrows
INSERT INTO script_search_points (
    id,
    script_version_id,
    point_key,
    phase_id,
    scene_id,
    title,
    public_description,
    max_searches,
    conflict_policy,
    visibility_scope,
    reveal_gate_id,
    metadata_json
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
    ?
);

-- name: UpdateScriptSearchPoint :execrows
UPDATE script_search_points
SET point_key = ?,
    phase_id = ?,
    scene_id = ?,
    title = ?,
    public_description = ?,
    max_searches = ?,
    conflict_policy = ?,
    visibility_scope = ?,
    reveal_gate_id = ?,
    metadata_json = ?,
    row_version = row_version + 1
WHERE id = ?
  AND script_version_id = ?;

-- name: DeleteScriptSearchPoint :execrows
DELETE FROM script_search_points
WHERE id = ?
  AND script_version_id = ?;

-- name: ListScriptSearchPointClues :many
SELECT *
FROM script_search_point_clues
WHERE script_version_id = ?
ORDER BY search_point_id, clue_id;

-- name: UpsertScriptSearchPointClue :exec
INSERT INTO script_search_point_clues (
    script_version_id,
    search_point_id,
    clue_id,
    weight,
    is_default,
    metadata_json
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
ON DUPLICATE KEY UPDATE
    weight = VALUES(weight),
    is_default = VALUES(is_default),
    metadata_json = VALUES(metadata_json);

-- name: DeleteScriptSearchPointClue :execrows
DELETE FROM script_search_point_clues
WHERE script_version_id = ?
  AND search_point_id = ?
  AND clue_id = ?;
