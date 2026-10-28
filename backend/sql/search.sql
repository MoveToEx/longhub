-- name: GetUnindexedVersion :one

SELECT
    v.*,
    sqlc.embed(im),
    ui.username AS uploader,
    ARRAY(
      SELECT name
      FROM tag t
      WHERE EXISTS (
        SELECT 1 FROM version_tag vt
        WHERE vt.version_id = v.id AND vt.tag_id = t.id
      )
    )::TEXT[] AS tags
FROM image im
JOIN version v ON v.id = im.current_version_id
JOIN user_identifier ui ON ui.id = im.user_id
WHERE im.id = @image_id
  AND v.id = @version_id
  AND (im.indexed_version IS NULL OR im.indexed_version != v.id)
FOR UPDATE OF im;

-- name: CommitUnindexedVersion :exec
UPDATE image
SET indexed_version = @version_id
WHERE id = @image_id
  AND current_version_id = @version_id;

-- name: GetUnindexedImages :many
SELECT im.id AS image_id,
       CASE WHEN im.deleted_at IS NOT NULL THEN 0 ELSE v.id END::BIGINT AS version_id
FROM image im
LEFT JOIN version v ON v.id = im.current_version_id
WHERE (im.deleted_at IS NOT NULL AND im.indexed_version IS DISTINCT FROM 0)
   OR (im.deleted_at IS NULL AND v.id IS NOT NULL AND im.indexed_version IS DISTINCT FROM v.id)
ORDER BY im.id
LIMIT 100;

-- name: GetImagesByIds :many
SELECT v.text, v.rating, im.id, im.image_url, ARRAY(
  SELECT name
  FROM tag t
  WHERE EXISTS (
    SELECT 1 FROM version_tag vt
    WHERE vt.version_id = v.id AND vt.tag_id = t.id
  )
)::TEXT[] AS tags FROM image im
INNER JOIN version v ON im.current_version_id = v.id
WHERE im.id = ANY(@ids::BIGINT[])
  AND im.deleted_at IS NULL
ORDER BY ARRAY_POSITION(@ids::BIGINT[], im.id);
