-- name: GetTagsByImage :many
WITH image_version AS (
    SELECT current_version_id AS version_id
    FROM image i
    WHERE i.id = $1
)
SELECT t.id, t.name FROM tag t
JOIN image_version iv ON EXISTS (
    SELECT 1 FROM version_tag vt
    WHERE vt.version_id = iv.version_id AND vt.tag_id = t.id
);

-- name: GetTagsByVersion :many
SELECT tag.name FROM tag
INNER JOIN version_tag ON version_tag.tag_id = tag.id
WHERE version_tag.version_id = $1;

-- name: CreateTags :exec
INSERT INTO tag ("name")
SELECT *
FROM UNNEST(@tag_names::TEXT[]) AS n("name")
ON CONFLICT ("name") DO NOTHING;

-- name: GetTagsWithPrefix :many
SELECT * FROM public.tag
WHERE name LIKE CONCAT(@prefix::TEXT, '%')
ORDER BY name
LIMIT $1;

-- name: CountTagsWithPrefix :one
SELECT COUNT(*) FROM public.tag
WHERE name LIKE CONCAT(@text, '%');
