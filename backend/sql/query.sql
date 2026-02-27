-- name: GetImage :one
SELECT
    i.id, v.text, v.rating, image_url, image_key, i.created_at, 
    sqlc.embed(ui), i.current_version_id, v.version
FROM public.image i
JOIN public.user_identifier ui ON i.user_id = ui.id 
JOIN public.version v ON i.current_version_id = v.id
WHERE i.id = $1 AND i.deleted_at IS NULL LIMIT 1;

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

-- name: ListImages :many
SELECT i.id, image_url, image_key, i.user_id, current_version_id, v.text, v.rating
FROM public.image i
JOIN public.version v ON i.current_version_id = v.id
ORDER BY i.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountImages :one
SELECT COUNT(*) FROM public.image
WHERE deleted_at IS NULL;

-- name: SetImageEmbedding :exec
UPDATE image SET embedding = $2::vector WHERE id = $1;

-- name: SetImageHash :exec
UPDATE image SET "image_hash" = $2 WHERE "id" = $1;

-- name: SetImageVersion :exec
UPDATE image SET "current_version_id" = $2 WHERE "id" = $1;

-- name: SetVersionText :exec
UPDATE version SET "text" = $2 WHERE "id" = $1;

-- name: SetVersionRating :exec
UPDATE version SET "rating" = $2 WHERE "id" = $1;

-- name: CreateImage :one
INSERT INTO image(image_key, image_url, user_id)
VALUES ($1, $2, $3)
RETURNING id;

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
RETURNING image.id;

-- name: GetImagesByUser :many
SELECT
    image.id, text, rating, image_url, image_key, current_version_id FROM image
JOIN version ON image.current_version_id = version.id
WHERE image.user_id = $1 AND image.deleted_at IS NULL
LIMIT $2 OFFSET $3;

-- name: CountImagesByUser :one
SELECT COUNT(*) FROM image
WHERE "user_id" = $1 AND "deleted_at" IS NULL;

-- name: CountVersionsByUser :one
SELECT COUNT(*) FROM version
WHERE user_id = $1;

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
WHERE tag.id = $1;

-- name: CreateNewVersion :exec
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
)
INSERT INTO version_tag (version_id, tag_id)
SELECT nv.id, t.id
FROM new_version nv
JOIN tag t ON t.name = ANY(@tag_names::TEXT[]);

-- name: CreateTags :exec
INSERT INTO tag ("name")
SELECT *
FROM UNNEST(@tag_names::TEXT[]) AS n("name")
ON CONFLICT ("name") DO NOTHING;

-- name: SetUserPermission :exec
UPDATE public.user SET "permission" = $2 WHERE "id" = $1;

-- name: CreateUser :exec
INSERT INTO public.user (
    "email", "username", "permission", "password_hash"
) VALUES (
    $1, $2, $3, $4
);

-- name: GetUser :one
SELECT * FROM public.user
WHERE "id" = $1;

-- name: GetUserByName :one
SELECT * FROM public.user
WHERE "username" = $1 LIMIT 1;

-- name: GetOther :one
SELECT id, created_at, username FROM public.user
WHERE id = $1;

-- name: CreateUploadSession :one
INSERT INTO public.upload_session (
    "user_id", "key"
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetUploadSession :one
SELECT * FROM public.upload_session
WHERE "id" = $1;

-- name: CompleteUploadSession :one
UPDATE public.upload_session
SET "completed_at" = NOW(), "status" = 'completed'
WHERE "id" = $1
    AND "user_id" = $2
    AND "status" = 'active'
RETURNING *;

-- name: MarkUploadSessionAsExpired :exec
UPDATE public.upload_session
SET "status" = 'expired'
WHERE "id" = $1;

-- name: SetUserPreference :exec
UPDATE public.user
SET "preference" = $2
WHERE "id" = $1;

-- name: FilterImages :many
WITH current_version AS (
  SELECT DISTINCT ON (image_id)
  id AS version_id, image_id, text, rating
  FROM public.version
  ORDER BY image_id, version_id DESC
),
include_tag_ids AS (
  SELECT id
  FROM tag
  WHERE tag.name = ANY (@include_tags::TEXT[])
),
exclude_tag_ids AS (
  SELECT id
  FROM tag
  WHERE tag.name = ANY (@exclude_tags::TEXT[])
)
SELECT i.id, i.image_url, i.image_key, i.created_at, cv.text, cv.rating, i.user_id, i.deleted_at
FROM image i
JOIN current_version cv ON cv.image_id = i.id

-- INCLUDE: must contain ALL include tags
JOIN version_tag vt_in
  ON vt_in.version_id = cv.version_id
JOIN include_tag_ids it
  ON it.id = vt_in.tag_id
  
-- EXCLUDE: must contain NONE of exclude tags
LEFT JOIN version_tag vt_ex
  ON vt_ex.version_id = cv.version_id
 AND vt_ex.tag_id IN (SELECT id FROM exclude_tag_ids)
WHERE cv.text LIKE CONCAT('%', @text, '%')
GROUP BY i.id, cv.version_id
HAVING
  COUNT(DISTINCT it.id) = (SELECT COUNT(*) FROM include_tag_ids)
  AND COUNT(vt_ex.tag_id) = 0
LIMIT $1
OFFSET $2;

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
WHERE t.name = $1
ORDER BY RANDOM()
LIMIT $2;

-- name: GetTagsWithPrefix :many
SELECT * FROM public.tag
WHERE name LIKE CONCAT(@prefix::TEXT, '%')
ORDER BY name
LIMIT $1;

-- name: CountTagsWithPrefix :one
SELECT COUNT(*) FROM public.tag
WHERE name LIKE CONCAT(@text, '%');

-- name: QuickSearchFavorites :many
WITH current_version AS (
  SELECT DISTINCT ON (image_id)
         id AS version_id,
         image_id,
         text, rating
  FROM version
  ORDER BY image_id, version DESC
)
SELECT
  i.id, i.image_url, cv.text, cv.rating, 
  MAX(LENGTH(q.term))::FLOAT AS relevance
FROM image i
JOIN current_version cv ON cv.image_id = i.id
JOIN user_favorite fav ON fav.image_id = i.id
JOIN UNNEST(@keywords::TEXT[]) AS q(term) ON fav.shortcut ILIKE q.term || '%'
WHERE fav.user_id = @user_id
GROUP BY i.id, i.image_url, cv.text, cv.rating
ORDER BY relevance DESC, i.id
LIMIT 12;

-- name: QuickSearchImage :many
WITH current_version AS (
  SELECT DISTINCT ON (image_id)
         id AS version_id,
         image_id,
         text, rating
  FROM version
  ORDER BY image_id, version DESC
),

text_hits AS (
  SELECT
    cv.image_id,
    SUM(similarity(cv.text, q.term)) AS text_score
  FROM current_version cv,
       unnest(@keywords::TEXT[]) AS q(term)
  WHERE cv.text LIKE '%' || q.term || '%'
  GROUP BY cv.image_id
),

tag_hits AS (
  SELECT
    cv.image_id, COUNT(*) * 0.5 AS tag_score
  FROM current_version cv
  JOIN version_tag vt
    ON vt.version_id = cv.version_id
  JOIN tag t
    ON t.id = vt.tag_id
  WHERE t.name = ANY (@keywords::TEXT[])
  GROUP BY cv.image_id
)

SELECT
  i.id, i.image_url, cv.text, cv.rating, 
  (COALESCE(th.text_score, 0.0) + COALESCE(tg.tag_score, 0.0))::FLOAT AS relevance
FROM image i
JOIN current_version cv ON cv.image_id = i.id
LEFT JOIN text_hits th ON th.image_id = i.id
LEFT JOIN tag_hits tg ON tg.image_id = i.id
WHERE
  th.image_id IS NOT NULL
  OR tg.image_id IS NOT NULL
ORDER BY relevance DESC, i.id
LIMIT 12;

-- name: GetUserContribution :many
WITH image_daily AS (
  SELECT
    created_at::date AS day,
    COUNT(*) AS image_count
  FROM image
  WHERE image.user_id = $1
    AND created_at >= (CURRENT_DATE - INTERVAL '365 days')
  GROUP BY created_at::date
),

version_daily AS (
  SELECT
    created_at::date AS day,
    COUNT(*) AS version_count
  FROM version
  WHERE version.user_id = $1
    AND created_at >= (CURRENT_DATE - INTERVAL '365 days')
  GROUP BY created_at::date
)

SELECT
  COALESCE(i.day, v.day) AS day,
  COALESCE(i.image_count, 0)   AS image_count,
  COALESCE(v.version_count, 0) AS version_count
FROM image_daily i
FULL OUTER JOIN version_daily v
  ON v.day = i.day
ORDER BY i.day;

-- ###################
-- # Favorite

-- name: GetFavoriteState :one
SELECT * FROM user_favorite
WHERE user_id = $1 AND image_id = $2;

-- name: AddFavorite :exec
INSERT INTO user_favorite(user_id, image_id, shortcut)
VALUES ($1, $2, $3);

-- name: SetFavoriteShortcut :exec
UPDATE user_favorite
SET shortcut = $3
WHERE user_id = $1 AND image_id = $2;

-- name: DeleteFavorite :exec
DELETE FROM user_favorite
WHERE user_id = $1 AND image_id = $2;

-- name: CountFavoriteImages :one
SELECT COUNT(*)
FROM user_favorite
WHERE user_id = $1;

-- name: GetFavoriteImages :many
WITH current_version AS (
  SELECT DISTINCT ON (image_id)
         id AS version_id,
         image_id,
         text, rating, version
  FROM version
  ORDER BY image_id, version DESC
)
SELECT fav.shortcut, fav.favorited_at, fav.user_id, i.id, i.image_key, i.image_url, cv.text, cv.rating, cv.version FROM user_favorite fav
JOIN image i ON fav.image_id = i.id
JOIN current_version cv ON i.current_version_id = cv.version_id
WHERE fav.user_id = $3
ORDER BY fav.favorited_at DESC, i.id
OFFSET $1
LIMIT $2;

-- name: GetFavoriteByShortcut :many
SELECT * FROM user_favorite
WHERE shortcut LIKE @keyword || '%'
  AND user_id = $1;

-- name: GetFavoriteTags :many
WITH current_version AS (
  SELECT DISTINCT ON (v.image_id)
         v.id AS version_id,
         v.image_id
  FROM version v
  ORDER BY v.image_id, v.version DESC
),
favorite_versions AS (
  SELECT cv.version_id
  FROM user_favorite f
  JOIN current_version cv
    ON cv.image_id = f.image_id
  WHERE f.user_id = $1
)
SELECT DISTINCT t.id, t.name, COUNT(*) AS count
FROM favorite_versions fv
JOIN version_tag vt ON vt.version_id = fv.version_id
JOIN tag t ON t.id = vt.tag_id
GROUP BY t.id
ORDER BY count DESC
LIMIT $2;

-- ###################
-- # WebAuthn

-- name: GetWebAuthnSession :one
SELECT * FROM webauthn_session
WHERE id = $1;

-- name: CreateWebAuthnSession :exec
INSERT INTO webauthn_session(id, data)
VALUES ($1, $2);

-- name: GetCredentials :many
SELECT * FROM webauthn_passkey
WHERE user_id = $1;

-- name: CreateCredential :exec
INSERT INTO webauthn_passkey(id, user_id, public_key, sign_count, transports, aaguid, flags)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetPasskey :many
SELECT id, name, user_id, aaguid, created_at FROM webauthn_passkey
WHERE user_id = $1;

-- name: GetUserByHandle :one
SELECT * FROM public.user
WHERE handle = $1;

-- name: DeletePasskey :exec
DELETE FROM webauthn_passkey
WHERE id = $1 AND user_id = $2;

-- name: SetPasskeyName :exec
UPDATE webauthn_passkey
SET name = $1
WHERE id = $2 AND user_id = $3;

-- ###################
-- # Integration

-- name: GetUserByAppKey :one
SELECT u.*, k.id AS key_id, k.permission AS key_permission FROM public.user u
INNER JOIN appkey k ON u.id = k.user_id
WHERE k.key = $1;

-- name: GetAppKeysByUser :many
SELECT id, created_at, label, user_id, permission, last_activated_at FROM appkey
WHERE user_id = $1;

-- name: GetAppKeyById :one
SELECT * FROM appkey
WHERE id = $1;

-- name: GetAppKey :one
SELECT * FROM appkey
WHERE key = $1;

-- name: UpdateAppKey :exec
UPDATE appkey
SET permission = $1, label = $2
WHERE id = $3;

-- name: DeleteAppKey :exec
DELETE FROM appkey
WHERE id = $1;

-- name: CreateAppKey :one
INSERT INTO appkey(user_id, permission, label, key)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: UpdateAppKeyUsedDate :exec
UPDATE appkey
SET last_activated_at = NOW()
WHERE id = $1;

