-- name: GetCredentials :many


SELECT * FROM webauthn_passkey
WHERE user_id = $1;

-- name: CreateCredential :exec
INSERT INTO webauthn_passkey(id, user_id, public_key, sign_count, transports, aaguid, flags)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetPasskey :many
SELECT id, name, user_id, aaguid, created_at FROM webauthn_passkey
WHERE user_id = $1;

-- name: GetUserByHandle :one
SELECT * FROM public.user
WHERE handle = $1;

-- name: DeletePasskey :exec
DELETE FROM webauthn_passkey
WHERE id = $1 AND user_id = $2;

-- name: SetPasskeyName :exec
UPDATE webauthn_passkey
SET name = $1
WHERE id = $2 AND user_id = $3;
