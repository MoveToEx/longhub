-- name: GetImageVersions :many
SELECT v.*, sqlc.embed(ui), ARRAY(
  SELECT name
  FROM tag t
  WHERE EXISTS (
    SELECT 1 FROM version_tag vt
    WHERE vt.version_id = v.id AND vt.tag_id = t.id
  )
)::TEXT[] AS tags
FROM public.version v
JOIN public.user_identifier ui ON v.user_id = ui.id
WHERE v.image_id = $1
ORDER BY v.created_at DESC;

-- name: SetVersionText :exec
UPDATE version SET "text" = $2 WHERE "id" = $1;

-- name: SetVersionRating :exec
UPDATE version SET "rating" = $2 WHERE "id" = $1;

-- name: InitializeVersion :one
WITH new_version AS (
    INSERT INTO version(text, rating, image_id, user_id)
    VALUES ($1, $2, $3, $4)
    RETURNING id, image_id
), version_tags AS (
    INSERT INTO version_tag ("version_id", "tag_id")
    SELECT nv.id, t.id
    FROM new_version nv
    JOIN tag t ON t.name = ANY(@tag_names::TEXT[])
)
UPDATE image
SET current_version_id = new_version.id
FROM new_version
WHERE image.id = new_version.image_id
RETURNING new_version.id;

-- name: CountVersionsByUser :one
SELECT COUNT(*) FROM version
WHERE user_id = $1;

-- name: CreateNewVersion :one
WITH new_version AS (
    INSERT INTO version("text", "rating", "user_id", "image_id", "version")
    SELECT $1, $2, $3, image.id, version.version + 1
    FROM image
    JOIN version ON image.current_version_id = version.id
    WHERE image.id = @image_id
    RETURNING id
),
new_image AS (
    UPDATE image
    SET current_version_id = nv.id
    FROM new_version nv
    WHERE image.id = @image_id
),
new_version_tags AS (
    INSERT INTO version_tag (version_id, tag_id)
    SELECT nv.id, t.id
    FROM new_version nv
    JOIN tag t ON t.name = ANY(@tag_names::TEXT[])
)
SELECT id FROM new_version;
