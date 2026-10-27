-- +goose Up
CREATE INDEX version_created_at_idx ON version (created_at);
CREATE INDEX deletion_outbox_created_at_idx ON deletion_outbox (created_at);

-- +goose Down
DROP INDEX deletion_outbox_created_at_idx;
DROP INDEX version_created_at_idx;
