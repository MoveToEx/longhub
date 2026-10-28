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
