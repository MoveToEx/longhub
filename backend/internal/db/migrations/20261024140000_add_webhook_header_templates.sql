-- +goose Up
ALTER TABLE webhook ADD COLUMN header_templates JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE webhook DROP COLUMN header_templates;
