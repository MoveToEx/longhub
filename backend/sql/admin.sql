-- name: AdminListImages :many

SELECT i.id, i.image_url, i.image_key, i.created_at, i.updated_at, i.deleted_at,
       i.user_id, u.username AS uploader_username,
       v.id AS version_id, v.version, v.text, v.rating,
       ARRAY(SELECT t.name FROM tag t
             JOIN version_tag vt ON vt.tag_id = t.id
             WHERE vt.version_id = v.id ORDER BY t.name)::TEXT[] AS tags
FROM image i
JOIN public.user u ON u.id = i.user_id
LEFT JOIN version v ON v.id = i.current_version_id
WHERE (@status::TEXT = 'ALL'
       OR (@status::TEXT = 'ACTIVE' AND i.deleted_at IS NULL)
       OR (@status::TEXT = 'DELETED' AND i.deleted_at IS NOT NULL))
  AND (@search::TEXT = '' OR i.id = @search_id::BIGINT
       OR v.text ILIKE @pattern::TEXT
       OR u.username ILIKE @pattern::TEXT
       OR EXISTS (SELECT 1 FROM version_tag vt JOIN tag t ON t.id = vt.tag_id
                  WHERE vt.version_id = v.id AND t.name ILIKE @pattern::TEXT))
ORDER BY
  CASE WHEN @sort_by::TEXT = 'ID' AND @direction::TEXT = 'ASC' THEN i.id END ASC,
  CASE WHEN @sort_by::TEXT = 'ID' AND @direction::TEXT = 'DESC' THEN i.id END DESC,
  CASE WHEN @sort_by::TEXT = 'CREATED_AT' AND @direction::TEXT = 'ASC' THEN i.created_at END ASC,
  CASE WHEN @sort_by::TEXT = 'CREATED_AT' AND @direction::TEXT = 'DESC' THEN i.created_at END DESC,
  CASE WHEN @sort_by::TEXT = 'UPDATED_AT' AND @direction::TEXT = 'ASC' THEN i.updated_at END ASC,
  CASE WHEN @sort_by::TEXT = 'UPDATED_AT' AND @direction::TEXT = 'DESC' THEN i.updated_at END DESC,
  i.id DESC
LIMIT @page_limit::INT OFFSET @page_offset::INT;

-- name: AdminCountImages :one
SELECT COUNT(*) FROM image i
JOIN public.user u ON u.id = i.user_id
LEFT JOIN version v ON v.id = i.current_version_id
WHERE (@status::TEXT = 'ALL'
       OR (@status::TEXT = 'ACTIVE' AND i.deleted_at IS NULL)
       OR (@status::TEXT = 'DELETED' AND i.deleted_at IS NOT NULL))
  AND (@search::TEXT = '' OR i.id = @search_id::BIGINT
       OR v.text ILIKE @pattern::TEXT
       OR u.username ILIKE @pattern::TEXT
       OR EXISTS (SELECT 1 FROM version_tag vt JOIN tag t ON t.id = vt.tag_id
                  WHERE vt.version_id = v.id AND t.name ILIKE @pattern::TEXT));

-- name: AdminListUsers :many
WITH image_counts AS (
  SELECT user_id, COUNT(*) AS count FROM image WHERE deleted_at IS NULL GROUP BY user_id
), version_counts AS (
  SELECT user_id, COUNT(*) AS count FROM version GROUP BY user_id
)
SELECT u.id, u.username, u.email, u.permission, u.created_at,
       COALESCE(ic.count, 0)::BIGINT AS active_image_count,
       COALESCE(vc.count, 0)::BIGINT AS version_count
FROM public.user u
LEFT JOIN image_counts ic ON ic.user_id = u.id
LEFT JOIN version_counts vc ON vc.user_id = u.id
WHERE @search::TEXT = '' OR u.id = @search_id::BIGINT
   OR u.username ILIKE @pattern::TEXT OR u.email ILIKE @pattern::TEXT
ORDER BY
  CASE WHEN @sort_by::TEXT = 'ID' AND @direction::TEXT = 'ASC' THEN u.id END ASC,
  CASE WHEN @sort_by::TEXT = 'ID' AND @direction::TEXT = 'DESC' THEN u.id END DESC,
  CASE WHEN @sort_by::TEXT = 'USERNAME' AND @direction::TEXT = 'ASC' THEN u.username END ASC,
  CASE WHEN @sort_by::TEXT = 'USERNAME' AND @direction::TEXT = 'DESC' THEN u.username END DESC,
  CASE WHEN @sort_by::TEXT = 'EMAIL' AND @direction::TEXT = 'ASC' THEN u.email END ASC,
  CASE WHEN @sort_by::TEXT = 'EMAIL' AND @direction::TEXT = 'DESC' THEN u.email END DESC,
  CASE WHEN @sort_by::TEXT = 'CREATED_AT' AND @direction::TEXT = 'ASC' THEN u.created_at END ASC,
  CASE WHEN @sort_by::TEXT = 'CREATED_AT' AND @direction::TEXT = 'DESC' THEN u.created_at END DESC,
  CASE WHEN @sort_by::TEXT = 'PERMISSION' AND @direction::TEXT = 'ASC' THEN u.permission END ASC,
  CASE WHEN @sort_by::TEXT = 'PERMISSION' AND @direction::TEXT = 'DESC' THEN u.permission END DESC,
  u.id DESC
LIMIT @page_limit::INT OFFSET @page_offset::INT;

-- name: AdminCountUsers :one
SELECT COUNT(*) FROM public.user u
WHERE @search::TEXT = '' OR u.id = @search_id::BIGINT
   OR u.username ILIKE @pattern::TEXT OR u.email ILIKE @pattern::TEXT;

-- name: AdminUpdateUser :one
WITH updated AS (
  UPDATE public.user
  SET email = @email::TEXT,
      password_hash = COALESCE(sqlc.narg(password_hash)::BYTEA, password_hash),
      permission = (permission & ~(@editable_permissions::BIGINT & 15::BIGINT))
                 | (@permission::BIGINT & @editable_permissions::BIGINT & 15::BIGINT)
  WHERE id = @id::BIGINT
    AND (permission & 16::BIGINT = 0 OR @editable_permissions::BIGINT & 8::BIGINT != 0)
  RETURNING id, username, email, permission, created_at
)
SELECT updated.*,
       (SELECT COUNT(*) FROM image WHERE user_id = updated.id AND deleted_at IS NULL) AS active_image_count,
       (SELECT COUNT(*) FROM version WHERE user_id = updated.id) AS version_count
FROM updated;
