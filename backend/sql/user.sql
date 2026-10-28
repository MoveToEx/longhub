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
