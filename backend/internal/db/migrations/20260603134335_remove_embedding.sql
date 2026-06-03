-- +goose Up
SELECT 'up SQL query';

ALTER TABLE image DROP COLUMN embedding;

-- +goose Down
SELECT 'down SQL query';

ALTER TABLE image ADD COLUMN embedding VECTOR;