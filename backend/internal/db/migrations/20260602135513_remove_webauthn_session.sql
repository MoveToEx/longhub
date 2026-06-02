-- +goose Up
SELECT 'up SQL query';

DROP TABLE IF EXISTS "webauthn_session";

-- +goose Down
SELECT 'down SQL query';

CREATE TABLE IF NOT EXISTS "webauthn_session" (
	"id" TEXT NOT NULL,
	"created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
	"data" JSONB NOT NULL,
	PRIMARY KEY("id")
);
