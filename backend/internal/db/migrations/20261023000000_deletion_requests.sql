-- +goose Up
ALTER TABLE deletion ADD COLUMN processed_by BIGINT REFERENCES public.user(id) ON DELETE SET NULL;
CREATE UNIQUE INDEX deletion_one_active_per_user ON deletion(image_id, user_id) WHERE status = 'active';
CREATE UNIQUE INDEX deletion_one_approved_per_image ON deletion(image_id) WHERE status = 'approved';
CREATE INDEX deletion_image_created_idx ON deletion(image_id, created_at DESC, id DESC);
CREATE INDEX deletion_status_created_idx ON deletion(status, created_at DESC, id DESC);

CREATE VIEW deletion_request_detail AS
SELECT d.*, u.username AS requester_username, m.username AS moderator_username,
       i.deleted_at AS image_deleted_at, i.active_deletion_id
FROM deletion d
JOIN public.user u ON u.id = d.user_id
JOIN image i ON i.id = d.image_id
LEFT JOIN public.user m ON m.id = d.processed_by;

CREATE TABLE deletion_outbox (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  image_id BIGINT NOT NULL REFERENCES image(id),
  kind TEXT NOT NULL CHECK (kind IN ('search', 'delete')),
  payload JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  dispatched_at TIMESTAMPTZ,
  after_webhook_id BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX deletion_outbox_pending_idx ON deletion_outbox(id) WHERE dispatched_at IS NULL;
INSERT INTO deletion_outbox(image_id, kind) SELECT id, 'search' FROM image WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP TABLE deletion_outbox;
DROP VIEW deletion_request_detail;
DROP INDEX deletion_one_active_per_user;
DROP INDEX deletion_one_approved_per_image;
DROP INDEX deletion_image_created_idx;
DROP INDEX deletion_status_created_idx;
ALTER TABLE deletion DROP COLUMN processed_by;
