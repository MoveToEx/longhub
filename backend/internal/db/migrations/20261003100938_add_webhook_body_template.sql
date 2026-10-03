-- +goose Up
ALTER TABLE webhook ADD COLUMN body_template TEXT;

-- +goose Down
ALTER TABLE webhook DROP COLUMN body_template;
