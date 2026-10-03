-- +goose Up
-- Legacy timestamps contain wall-clock times in the database session's timezone.
-- Preserve those instants when adding timezone information (Asia/Shanghai in production).
DROP VIEW user_identifier;

ALTER TABLE public.user
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE current_setting('TimeZone');

CREATE VIEW user_identifier AS
SELECT id, created_at, username FROM public.user;

ALTER TABLE image
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE current_setting('TimeZone'),
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE current_setting('TimeZone'),
    ALTER COLUMN deleted_at TYPE TIMESTAMPTZ USING deleted_at AT TIME ZONE current_setting('TimeZone');

ALTER TABLE version
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE current_setting('TimeZone');

ALTER TABLE user_favorite
    ALTER COLUMN favorited_at TYPE TIMESTAMPTZ USING favorited_at AT TIME ZONE current_setting('TimeZone');

ALTER TABLE appkey
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE current_setting('TimeZone'),
    ALTER COLUMN last_activated_at TYPE TIMESTAMPTZ USING last_activated_at AT TIME ZONE current_setting('TimeZone');

ALTER TABLE webhook
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE current_setting('TimeZone'),
    ALTER COLUMN last_activated_at TYPE TIMESTAMPTZ USING last_activated_at AT TIME ZONE current_setting('TimeZone');

ALTER TABLE webauthn_passkey
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE current_setting('TimeZone');

ALTER TABLE deletion
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE current_setting('TimeZone'),
    ALTER COLUMN processed_at TYPE TIMESTAMPTZ USING processed_at AT TIME ZONE current_setting('TimeZone');

ALTER TABLE upload_session
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE current_setting('TimeZone'),
    ALTER COLUMN completed_at TYPE TIMESTAMPTZ USING completed_at AT TIME ZONE current_setting('TimeZone');

-- Rebuild search documents whose createdAt Unix timestamps used the wrong timezone.
UPDATE image SET indexed_version = NULL;

-- +goose Down
DROP VIEW user_identifier;

ALTER TABLE public.user
    ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE current_setting('TimeZone');

CREATE VIEW user_identifier AS
SELECT id, created_at, username FROM public.user;

ALTER TABLE image
    ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE current_setting('TimeZone'),
    ALTER COLUMN updated_at TYPE TIMESTAMP USING updated_at AT TIME ZONE current_setting('TimeZone'),
    ALTER COLUMN deleted_at TYPE TIMESTAMP USING deleted_at AT TIME ZONE current_setting('TimeZone');

ALTER TABLE version
    ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE current_setting('TimeZone');

ALTER TABLE user_favorite
    ALTER COLUMN favorited_at TYPE TIMESTAMP USING favorited_at AT TIME ZONE current_setting('TimeZone');

ALTER TABLE appkey
    ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE current_setting('TimeZone'),
    ALTER COLUMN last_activated_at TYPE TIMESTAMP USING last_activated_at AT TIME ZONE current_setting('TimeZone');

ALTER TABLE webhook
    ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE current_setting('TimeZone'),
    ALTER COLUMN last_activated_at TYPE TIMESTAMP USING last_activated_at AT TIME ZONE current_setting('TimeZone');

ALTER TABLE webauthn_passkey
    ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE current_setting('TimeZone');

ALTER TABLE deletion
    ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE current_setting('TimeZone'),
    ALTER COLUMN processed_at TYPE TIMESTAMP USING processed_at AT TIME ZONE current_setting('TimeZone');

ALTER TABLE upload_session
    ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE current_setting('TimeZone'),
    ALTER COLUMN completed_at TYPE TIMESTAMP USING completed_at AT TIME ZONE current_setting('TimeZone');

UPDATE image SET indexed_version = NULL;
