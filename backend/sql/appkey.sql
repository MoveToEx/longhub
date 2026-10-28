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
