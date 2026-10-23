-- name: LockImageForDeletion :one
SELECT * FROM image WHERE id = $1 FOR UPDATE;

-- name: GetDeletionRequestRecord :one
SELECT * FROM deletion_request_detail WHERE id = $1;

-- name: GetDeletionRequestImage :one
SELECT image_id FROM deletion WHERE id = $1;

-- name: LockDeletionRequest :one
SELECT * FROM deletion WHERE id = $1 FOR UPDATE;

-- name: HasActiveDeletionRequest :one
SELECT EXISTS(SELECT 1 FROM deletion WHERE image_id = $1 AND user_id = $2 AND status = 'active');

-- name: CreateDeletionRequest :one
INSERT INTO deletion(image_id, user_id, reason, status) VALUES ($1, $2, $3, 'active') RETURNING id;

-- name: UpdateDeletionDecision :exec
UPDATE deletion SET status = @status,
  processed_at = CASE WHEN @status::deletion_status = 'active' THEN NULL ELSE NOW() END,
  processed_by = sqlc.narg(moderator_id)::BIGINT WHERE id = @id;

-- name: ApproveImageDeletion :exec
UPDATE image SET deleted_at = NOW(), active_deletion_id = @request_id::BIGINT,
  indexed_version = NULL WHERE id = @image_id;

-- name: RevokeImageDeletion :exec
UPDATE image SET deleted_at = NULL, active_deletion_id = NULL,
  indexed_version = NULL WHERE id = $1;

-- name: ListImageDeletionRequests :many
SELECT * FROM deletion_request_detail WHERE image_id = $1
ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3;

-- name: CountImageDeletionRequests :one
SELECT COUNT(*) FROM deletion WHERE image_id = $1;

-- name: AdminListDeletionRequests :many
SELECT * FROM deletion_request_detail
WHERE (@status::TEXT = 'ALL' OR UPPER(status::TEXT) = @status::TEXT)
  AND (@search::TEXT = '' OR id = @search_id::BIGINT OR image_id = @search_id::BIGINT
       OR reason ILIKE @pattern::TEXT OR requester_username ILIKE @pattern::TEXT)
ORDER BY
  CASE WHEN @sort_by::TEXT = 'ID' AND @direction::TEXT = 'ASC' THEN id END ASC,
  CASE WHEN @sort_by::TEXT = 'ID' AND @direction::TEXT = 'DESC' THEN id END DESC,
  CASE WHEN @sort_by::TEXT = 'CREATED_AT' AND @direction::TEXT = 'ASC' THEN created_at END ASC,
  CASE WHEN @sort_by::TEXT = 'CREATED_AT' AND @direction::TEXT = 'DESC' THEN created_at END DESC,
  CASE WHEN @sort_by::TEXT = 'PROCESSED_AT' AND @direction::TEXT = 'ASC' THEN processed_at END ASC NULLS LAST,
  CASE WHEN @sort_by::TEXT = 'PROCESSED_AT' AND @direction::TEXT = 'DESC' THEN processed_at END DESC NULLS LAST,
  id DESC LIMIT @page_limit::INT OFFSET @page_offset::INT;

-- name: AdminCountDeletionRequests :one
SELECT COUNT(*) FROM deletion_request_detail
WHERE (@status::TEXT = 'ALL' OR UPPER(status::TEXT) = @status::TEXT)
  AND (@search::TEXT = '' OR id = @search_id::BIGINT OR image_id = @search_id::BIGINT
       OR reason ILIKE @pattern::TEXT OR requester_username ILIKE @pattern::TEXT);

-- name: InsertDeletionOutbox :exec
INSERT INTO deletion_outbox(image_id, kind, payload) VALUES ($1, $2, $3);

-- name: LockPendingDeletionOutbox :many
SELECT * FROM deletion_outbox WHERE dispatched_at IS NULL ORDER BY id LIMIT 25 FOR UPDATE SKIP LOCKED;

-- name: AdvanceDeletionOutbox :exec
UPDATE deletion_outbox SET after_webhook_id = $2 WHERE id = $1;

-- name: CompleteDeletionOutbox :exec
UPDATE deletion_outbox SET dispatched_at = NOW() WHERE id = $1;

-- name: GetImageSnapshot :one
SELECT i.id, i.image_url, i.created_at, i.user_id, u.username,
       COALESCE(v.version, 0)::INT AS version,
       COALESCE(v.text, '')::TEXT AS text,
       COALESCE(v.rating::TEXT, 'none')::TEXT AS rating,
       ARRAY(SELECT t.name FROM tag t JOIN version_tag vt ON vt.tag_id = t.id
             WHERE vt.version_id = v.id ORDER BY t.name)::TEXT[] AS tags
FROM image i JOIN public.user u ON u.id = i.user_id
LEFT JOIN version v ON v.id = i.current_version_id WHERE i.id = $1;

-- name: CommitImageIndexRemoval :exec
-- Zero marks an acknowledged search tombstone; version IDs are always positive.
UPDATE image SET indexed_version = 0 WHERE id = $1
  AND (deleted_at IS NOT NULL OR current_version_id IS NULL);

-- name: GetImageDetails :one
SELECT i.id, COALESCE(v.text, '')::TEXT AS text, COALESCE(v.rating::TEXT, 'none')::TEXT AS rating,
       i.image_url, i.image_key, i.created_at, sqlc.embed(ui), i.current_version_id,
       COALESCE(v.version, 0)::INT AS version, i.deleted_at, i.active_deletion_id,
       COALESCE(d.reason, '')::TEXT AS deletion_reason
FROM image i JOIN user_identifier ui ON ui.id = i.user_id
LEFT JOIN version v ON v.id = i.current_version_id
LEFT JOIN deletion d ON d.id = i.active_deletion_id WHERE i.id = $1;
