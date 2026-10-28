-- name: GetImage :one
SELECT
    i.id, v.text, v.rating, image_url, image_key, i.created_at, 
    sqlc.embed(ui), i.current_version_id, v.version
FROM public.image i
JOIN public.user_identifier ui ON i.user_id = ui.id 
JOIN public.version v ON i.current_version_id = v.id
WHERE i.id = $1 AND i.deleted_at IS NULL LIMIT 1;

-- name: ListImages :many
SELECT i.id, image_url, image_key, i.user_id, current_version_id, v.text, v.rating
FROM public.image i
JOIN public.version v ON i.current_version_id = v.id
WHERE i.deleted_at ISNULL
ORDER BY i.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountImages :one
SELECT COUNT(*) FROM public.image i
JOIN version v ON v.id = i.current_version_id
WHERE i.deleted_at IS NULL;

-- name: SetImageHash :exec
UPDATE image SET "image_hash" = $2 WHERE "id" = $1;

-- name: SetImageVersion :exec
UPDATE image SET "current_version_id" = $2 WHERE "id" = $1;

-- name: CreateImage :one
INSERT INTO image(image_key, image_url, user_id)
VALUES ($1, $2, $3)
RETURNING id;

-- name: GetImagesByUser :many
SELECT
    im.id, text, rating, image_url, image_key, current_version_id FROM image im
JOIN version ON im.current_version_id = version.id
WHERE im.user_id = $1 AND im.deleted_at IS NULL
ORDER BY im.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountImagesByUser :one
SELECT COUNT(*) FROM image i
JOIN version v ON v.id = i.current_version_id
WHERE i.user_id = $1 AND i.deleted_at IS NULL;

-- name: GetRandomImage :many
WITH current_version AS (
  SELECT DISTINCT ON (image_id)
    id AS version_id, image_id, text, rating, version
  FROM version
  ORDER BY image_id, version DESC
)
SELECT i.id, i.image_url, i.image_key, cv.text, cv.rating
FROM current_version cv
JOIN image i ON cv.image_id = i.id
JOIN version_tag vt ON vt.version_id = cv.version_id
JOIN tag t ON t.id = vt.tag_id
WHERE t.name = $1 AND i.deleted_at IS NULL
ORDER BY RANDOM()
LIMIT $2;
