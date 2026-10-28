-- name: GetImageVersionForWebhook :one
SELECT
    i.id AS image_id,
    i.image_url,
    i.created_at,
    v.text,
    v.rating,
    v.version,
    uploader.id AS uploader_id,
    uploader.username AS uploader_username,
    initiator.id AS initiator_id,
    initiator.username AS initiator_username,
    ARRAY(
      SELECT name
      FROM tag t
      WHERE EXISTS (
        SELECT 1 FROM version_tag vt
        WHERE vt.version_id = v.id AND vt.tag_id = t.id
      )
    )::TEXT[] AS tags
FROM public.image i
JOIN public.version v ON v.image_id = i.id
JOIN public.user_identifier uploader ON uploader.id = i.user_id
JOIN public.user_identifier initiator ON initiator.id = v.user_id
WHERE i.id = @image_id
  AND v.id = @version_id
  AND i.deleted_at IS NULL
LIMIT 1;

-- name: GetPreviousImageVersionForWebhook :one
SELECT v.text, v.rating, ARRAY(
    SELECT t.name FROM tag t
    JOIN version_tag vt ON vt.tag_id = t.id
    WHERE vt.version_id = v.id
    ORDER BY t.name
)::TEXT[] AS tags
FROM version v
WHERE v.image_id = @image_id AND v.version < @version::INT
ORDER BY v.version DESC
LIMIT 1;

-- name: GetWebhook :one


SELECT * FROM webhook
WHERE id = $1;

-- name: GetWebhooksByUser :many
SELECT * FROM webhook
WHERE user_id = $1;

-- name: RecordWebhookFailure :exec
UPDATE webhook
SET failure_count = failure_count + 1,
    active = CASE WHEN failure_count + 1 >= $3 THEN FALSE ELSE active END,
    last_response_status = $2,
    last_activated_at = NOW()
WHERE id = $1;

-- name: RecordWebhookSuccess :exec
UPDATE webhook
SET failure_count = CASE WHEN active THEN 0 ELSE failure_count END,
    last_response_status = $2,
    last_activated_at = NOW()
WHERE id = $1;

-- name: CountWebhooksByUser :one
SELECT COUNT(*) FROM webhook
WHERE user_id = $1;

-- name: NewWebhook :one
INSERT INTO webhook(user_id, label, endpoint, event_types, secret, active, body_template, header_templates)
VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE(sqlc.narg(header_templates)::jsonb, '{}'::jsonb))
RETURNING *;

-- name: DeleteWebhook :exec
DELETE FROM webhook
WHERE id = $1;

-- name: UpdateWebhook :exec
UPDATE webhook
SET label = $1,
    endpoint = $2,
    event_types = $3,
    secret = $4,
    failure_count = CASE WHEN active = FALSE AND $5 = TRUE THEN 0 ELSE failure_count END,
    active = $5,
    body_template = $7,
    header_templates = COALESCE(sqlc.narg(header_templates)::jsonb, '{}'::jsonb)
WHERE id = $6;

-- name: GetWebhooks :many
SELECT * FROM webhook
WHERE active = TRUE AND failure_count < $3
ORDER BY id ASC
LIMIT $1 OFFSET $2;

-- name: GetWebhooksByEvent :many
SELECT * FROM webhook
WHERE id > @after_id
  AND event_types & @event_type::BIGINT != 0
  AND active = TRUE
  AND failure_count < @failure_count
ORDER BY id ASC
LIMIT @page_limit;
