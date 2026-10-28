-- +goose Up
DROP VIEW deletion_request_detail;
DROP INDEX deletion_one_active_per_user;
DROP INDEX deletion_one_approved_per_image;

-- Preserve previously restored images and retain their requests as rejected history.
CREATE TYPE deletion_status_permanent AS ENUM ('active', 'approved', 'rejected');
ALTER TABLE deletion ALTER COLUMN status TYPE deletion_status_permanent
  USING (CASE WHEN status::TEXT = 'revoked' THEN 'rejected' ELSE status::TEXT END)::deletion_status_permanent;
DROP TYPE deletion_status;
ALTER TYPE deletion_status_permanent RENAME TO deletion_status;

CREATE UNIQUE INDEX deletion_one_active_per_user ON deletion(image_id, user_id) WHERE status = 'active';
CREATE UNIQUE INDEX deletion_one_approved_per_image ON deletion(image_id) WHERE status = 'approved';
CREATE VIEW deletion_request_detail AS
SELECT d.*, u.username AS requester_username, m.username AS moderator_username,
       i.deleted_at AS image_deleted_at, i.active_deletion_id
FROM deletion d
JOIN public.user u ON u.id = d.user_id
JOIN image i ON i.id = d.image_id
LEFT JOIN public.user m ON m.id = d.processed_by;

-- Keep outbox history for catalog version timestamps, but stop pending restorations.
UPDATE deletion_outbox SET dispatched_at = NOW()
WHERE dispatched_at IS NULL AND kind = 'delete' AND payload->>'event' = 'restore';

-- +goose StatementBegin
CREATE FUNCTION enforce_permanent_image_deletion() RETURNS TRIGGER AS $$
BEGIN
  RAISE EXCEPTION 'Image deletions are permanent' USING ERRCODE = '23514';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER permanent_image_deletion
BEFORE UPDATE OF deleted_at ON image
FOR EACH ROW WHEN (OLD.deleted_at IS NOT NULL AND NEW.deleted_at IS NULL)
EXECUTE FUNCTION enforce_permanent_image_deletion();

-- +goose Down
DROP TRIGGER permanent_image_deletion ON image;
DROP FUNCTION enforce_permanent_image_deletion();
ALTER TYPE deletion_status ADD VALUE 'revoked';
