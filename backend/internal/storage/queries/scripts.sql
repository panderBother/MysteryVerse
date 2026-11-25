-- name: CreateScript :execrows
INSERT INTO scripts (
    id,
    owner_user_id,
    public_slug,
    title,
    summary,
    game_type,
    source_kind,
    rights_type,
    status,
    cover_asset_uri,
    language,
    content_rating,
    min_players,
    max_players,
    duration_minutes,
    difficulty,
    ai_assisted_ratio_bp,
    active_draft_version_id,
    published_version_id
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
    ?
);

-- name: GetScript :one
SELECT *
FROM scripts
WHERE id = ? AND deleted_at IS NULL;
