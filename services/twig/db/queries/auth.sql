-- name: CreateUser :one
INSERT INTO users (username, password_hash)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2 WHERE username = $1;

-- name: CreateSession :one
INSERT INTO sessions (id, user_id, created_at, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetSession :one
SELECT * FROM sessions WHERE id = $1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

-- name: CreateApiKey :one
INSERT INTO api_keys (user_id, key_hash, label, created_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetApiKeyByHash :one
SELECT * FROM api_keys WHERE key_hash = $1;

-- name: DeleteApiKeysByUser :exec
DELETE FROM api_keys WHERE user_id = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UpdateUserPasswordByID :exec
UPDATE users SET password_hash = $2 WHERE id = $1;

-- name: ListApiKeysByUser :many
SELECT id, label, created_at FROM api_keys WHERE user_id = $1 ORDER BY created_at, id;

-- name: DeleteApiKeyForUser :execrows
DELETE FROM api_keys WHERE id = $1 AND user_id = $2;
