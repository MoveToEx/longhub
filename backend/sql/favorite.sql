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
WHERE fav.user_id = @user_id AND i.deleted_at IS NULL
GROUP BY i.id, i.image_url, cv.text, cv.rating
ORDER BY relevance DESC, i.id
LIMIT 12;

-- name: GetFavoriteState :one

SELECT * FROM user_favorite
WHERE user_id = $1 AND image_id = $2;

-- name: AddFavorite :execrows
WITH active_image AS (
  SELECT id FROM image WHERE id = $2 AND deleted_at IS NULL FOR SHARE
)
INSERT INTO user_favorite(user_id, image_id, shortcut)
SELECT $1, id, $3 FROM active_image;

-- name: SetFavoriteShortcut :exec
UPDATE user_favorite
SET shortcut = $3
WHERE user_id = $1 AND image_id = $2;

-- name: DeleteFavorite :exec
DELETE FROM user_favorite
WHERE user_id = $1 AND image_id = $2;

-- name: CountFavoriteImages :one
SELECT COUNT(*)
FROM user_favorite f JOIN image i ON i.id = f.image_id
JOIN version v ON v.id = i.current_version_id
WHERE f.user_id = $1 AND i.deleted_at IS NULL;

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
WHERE fav.user_id = $3 AND i.deleted_at IS NULL
ORDER BY fav.favorited_at DESC, i.id
OFFSET $1
LIMIT $2;

-- name: GetFavoriteByShortcut :many
SELECT f.* FROM user_favorite f JOIN image i ON i.id = f.image_id
WHERE f.shortcut LIKE @keyword || '%'
  AND f.user_id = $1 AND i.deleted_at IS NULL;

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
  JOIN image i ON i.id = f.image_id AND i.deleted_at IS NULL
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
