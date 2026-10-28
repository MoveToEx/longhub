-- name: GetSyncVersion :one
-- Version creation records uploads and metadata edits; image tombstones record deletions.
SELECT GREATEST(
    (SELECT MAX(created_at) FROM version),
    (SELECT MAX(deleted_at) FROM image)
)::TIMESTAMPTZ AS timestamp;

-- name: ListSyncImages :many
SELECT i.id, i.image_url, i.image_key, i.created_at,
       v.id AS current_version_id, v.version, v.text, v.rating,
       sqlc.embed(ui),
       ARRAY(SELECT t.id FROM tag t JOIN version_tag vt ON vt.tag_id = t.id
             WHERE vt.version_id = v.id ORDER BY t.id)::BIGINT[] AS tag_ids,
       ARRAY(SELECT t.name FROM tag t JOIN version_tag vt ON vt.tag_id = t.id
             WHERE vt.version_id = v.id ORDER BY t.id)::TEXT[] AS tag_names
FROM image i
JOIN version v ON v.id = i.current_version_id
JOIN user_identifier ui ON ui.id = i.user_id
WHERE i.deleted_at IS NULL AND i.id > @after_id::BIGINT
ORDER BY i.id
LIMIT @page_limit::INT;
