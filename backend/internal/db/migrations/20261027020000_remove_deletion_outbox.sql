-- +goose Up
DROP TABLE deletion_outbox;

-- +goose Down
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
CREATE INDEX deletion_outbox_created_at_idx ON deletion_outbox(created_at);
