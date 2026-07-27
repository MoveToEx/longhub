-- +goose Up
SELECT 'up SQL query';

ALTER TABLE webhook
ADD COLUMN failure_count INTEGER NOT NULL DEFAULT 0;

-- +goose Down
SELECT 'down SQL query';

ALTER TABLE webhook
DROP COLUMN failure_count;