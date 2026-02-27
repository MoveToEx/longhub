-- +goose Up
SELECT 'up SQL query';

CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE "rating" AS ENUM (
	'none',
	'moderate',
	'violent'
);

CREATE TYPE "upload_status" AS ENUM (
	'active',
	'expired',
	'completed'
);

CREATE TYPE "deletion_status" AS ENUM (
	'active',
	'approved',
	'rejected',
	'revoked'
);

CREATE TABLE IF NOT EXISTS "user" (
	"id" BIGINT GENERATED ALWAYS AS IDENTITY,
	"created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
	"email" VARCHAR(255) NOT NULL,
	"username" VARCHAR(255) UNIQUE NOT NULL,
	"permission" BIGINT NOT NULL DEFAULT 0,
	"password_hash" BYTEA,
	"handle" BYTEA NOT NULL DEFAULT GEN_RANDOM_BYTES(64) UNIQUE,
	"preference" JSONB NOT NULL DEFAULT '{}'::JSONB,
	PRIMARY KEY("id")
);

CREATE INDEX "user_handle_idx"
ON public.user(handle);

CREATE VIEW "user_identifier" AS 
SELECT "id", "created_at", "username" FROM public.user;

CREATE TABLE IF NOT EXISTS "image" (
	"id" BIGINT GENERATED ALWAYS AS IDENTITY,
	"created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
	"updated_at" TIMESTAMP NOT NULL DEFAULT NOW(),
	"deleted_at" TIMESTAMP,
	"user_id" BIGINT NOT NULL,
	"image_key" TEXT NOT NULL,
	"image_url" TEXT NOT NULL,
	"image_hash" VARCHAR(255),
	"active_deletion_id" BIGINT,
	"current_version_id" BIGINT,
	"embedding" VECTOR,
	PRIMARY KEY("id")
);


CREATE INDEX "image_index_0"
ON "image" ("deleted_at");

CREATE TABLE IF NOT EXISTS "version" (
	"id" BIGINT GENERATED ALWAYS AS IDENTITY,
	"created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
	"image_id" BIGINT NOT NULL,
	"version" INT NOT NULL DEFAULT 1,
	"text" TEXT NOT NULL,
	"rating" RATING NOT NULL,
	"user_id" BIGINT NOT NULL,
	PRIMARY KEY("id"),
	UNIQUE ("image_id", "version")
);

CREATE INDEX "version_index_0"
ON "version" ("image_id", "version" DESC);

CREATE INDEX version_text_trgm
ON "version" USING GIN ("text" gin_trgm_ops);

CREATE INDEX version_current_idx
ON "version" ("image_id", "version" DESC);


CREATE TABLE IF NOT EXISTS "version_tag" (
	"version_id" BIGINT NOT NULL,
	"tag_id" BIGINT NOT NULL,
	PRIMARY KEY("version_id", "tag_id")
);

CREATE INDEX version_tag_version_idx
ON version_tag (version_id);

CREATE TABLE IF NOT EXISTS "tag" (
	"id" BIGINT GENERATED ALWAYS AS IDENTITY,
	"name" VARCHAR(255) NOT NULL UNIQUE,
	PRIMARY KEY("id")
);


CREATE TABLE IF NOT EXISTS "user_favorite" (
	"user_id" BIGINT NOT NULL,
	"image_id" BIGINT NOT NULL,
	"shortcut" VARCHAR(255),
	"favorited_at" TIMESTAMP NOT NULL DEFAULT NOW(),
	PRIMARY KEY("user_id", "image_id")
);

CREATE TABLE IF NOT EXISTS "appkey" (
	"id" BIGINT GENERATED ALWAYS AS IDENTITY,
	"label" TEXT NOT NULL DEFAULT 'New key',
	"created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
	"user_id" BIGINT NOT NULL,
	"permission" BIGINT NOT NULL,
	"last_activated_at" TIMESTAMP,
	"key" VARCHAR(255) UNIQUE NOT NULL,
	PRIMARY KEY("id")
);


CREATE TABLE IF NOT EXISTS "webhook" (
	"id" BIGINT GENERATED ALWAYS AS IDENTITY,
	"created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
	"user_id" BIGINT NOT NULL,
	"label" TEXT NOT NULL,
	"endpoint" TEXT NOT NULL,
	"event" INTEGER NOT NULL,
	"secret" TEXT NOT NULL,
	"last_activated_at" TIMESTAMP,
	"last_response_status" INTEGER,
	PRIMARY KEY("id")
);



CREATE TABLE IF NOT EXISTS "webauthn_session" (
	"id" TEXT NOT NULL,
	"created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
	"data" JSONB NOT NULL,
	PRIMARY KEY("id")
);


CREATE TABLE IF NOT EXISTS "webauthn_passkey" (
	"id" BYTEA NOT NULL,
	"name" TEXT NOT NULL DEFAULT 'Unnamed',
	"user_id" BIGINT NOT NULL,
	"public_key" BYTEA NOT NULL,
	"sign_count" BIGINT NOT NULL,
	"transports" TEXT[],
	"flags" SMALLINT NOT NULL,
	"created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
	"aaguid" UUID NOT NULL,
	PRIMARY KEY("id")
);


CREATE INDEX "appkey_index_0"
ON "appkey" ("key");

CREATE TABLE IF NOT EXISTS "deletion" (
	"id" BIGINT GENERATED ALWAYS AS IDENTITY,
	"created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
	"processed_at" TIMESTAMP,
	"image_id" BIGINT NOT NULL,
    "user_id" BIGINT NOT NULL,
	"reason" TEXT NOT NULL,
	"status" DELETION_STATUS NOT NULL,
	PRIMARY KEY("id")
);


CREATE TABLE IF NOT EXISTS "upload_session" (
	"id" BIGINT GENERATED ALWAYS AS IDENTITY,
	"created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
	"completed_at" TIMESTAMP,
	"user_id" BIGINT NOT NULL,
	"key" VARCHAR(255) NOT NULL,
	"status" UPLOAD_STATUS NOT NULL DEFAULT 'active',
	PRIMARY KEY("id")
);

CREATE TABLE IF NOT EXISTS "server_config" (
	"name" VARCHAR(255) NOT NULL,
	"value" JSONB NOT NULL DEFAULT '{}'::JSONB,
	PRIMARY KEY("name")
);

ALTER TABLE "webauthn_passkey"
ADD FOREIGN KEY("user_id") REFERENCES "user"("id")
ON UPDATE NO ACTION ON DELETE CASCADE;

ALTER TABLE "image"
ADD FOREIGN KEY("user_id") REFERENCES "user"("id")
ON UPDATE NO ACTION ON DELETE NO ACTION;
ALTER TABLE "image"
ADD FOREIGN KEY("current_version_id") REFERENCES "version"("id")
ON UPDATE NO ACTION ON DELETE NO ACTION;
ALTER TABLE "image"
ADD FOREIGN KEY("active_deletion_id") REFERENCES "deletion"("id")
ON UPDATE NO ACTION ON DELETE NO ACTION;

ALTER TABLE "version"
ADD FOREIGN KEY("image_id") REFERENCES "image"("id")
ON UPDATE NO ACTION ON DELETE NO ACTION;

ALTER TABLE "user_favorite"
ADD FOREIGN KEY("user_id") REFERENCES "user"("id")
ON UPDATE NO ACTION ON DELETE NO ACTION;
ALTER TABLE "user_favorite"
ADD FOREIGN KEY("image_id") REFERENCES "image"("id")
ON UPDATE NO ACTION ON DELETE NO ACTION;

ALTER TABLE "version_tag"
ADD FOREIGN KEY ("version_id") REFERENCES "version"("id")
ON UPDATE NO ACTION ON DELETE NO ACTION;
ALTER TABLE "version_tag"
ADD FOREIGN KEY ("tag_id") REFERENCES "tag"("id")
ON UPDATE NO ACTION ON DELETE NO ACTION;

ALTER TABLE "webhook"
ADD FOREIGN KEY("user_id") REFERENCES "user"("id")
ON UPDATE NO ACTION ON DELETE NO ACTION;

ALTER TABLE "appkey"
ADD FOREIGN KEY("user_id") REFERENCES "user"("id")
ON UPDATE NO ACTION ON DELETE NO ACTION;

ALTER TABLE "deletion"
ADD FOREIGN KEY("image_id") REFERENCES "image"("id")
ON UPDATE NO ACTION ON DELETE NO ACTION;
ALTER TABLE "deletion"
ADD FOREIGN KEY("user_id") REFERENCES "user"("id")
ON UPDATE NO ACTION ON DELETE NO ACTION;

ALTER TABLE "upload_session"
ADD FOREIGN KEY("user_id") REFERENCES "user"("id")
ON UPDATE NO ACTION ON DELETE NO ACTION;

-- +goose Down
SELECT 'down SQL query';

DROP TABLE IF EXISTS "server_config";
DROP TABLE IF EXISTS "upload_session" CASCADE;
DROP TABLE IF EXISTS "deletion" CASCADE;
DROP TABLE IF EXISTS "webauthn_passkey" CASCADE;
DROP TABLE IF EXISTS "webauthn_session" CASCADE;
DROP TABLE IF EXISTS "webhook" CASCADE;
DROP TABLE IF EXISTS "appkey" CASCADE;
DROP TABLE IF EXISTS "user_favorite" CASCADE;
DROP TABLE IF EXISTS "tag" CASCADE;
DROP TABLE IF EXISTS "version_tag" CASCADE;
DROP TABLE IF EXISTS "version" CASCADE;
DROP TABLE IF EXISTS "image" CASCADE;
DROP TABLE IF EXISTS "user" CASCADE;

DROP TYPE IF EXISTS rating;
DROP TYPE IF EXISTS upload_status;
DROP TYPE IF EXISTS deletion_status;

DROP EXTENSION IF EXISTS vector;
DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS pgcrypto;

