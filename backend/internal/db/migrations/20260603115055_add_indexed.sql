-- +goose Up
SELECT 'up SQL query';

ALTER TABLE image
ADD COLUMN indexed_version BIGINT;

CREATE INDEX "image_index_1"
ON "image" ("indexed_version");

-- +goose Down
SELECT 'down SQL query';

DROP INDEX "image_index_1";

ALTER TABLE image
DROP COLUMN indexed_version;