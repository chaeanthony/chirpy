-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens(token, created_at, updated_at, user_id, expires_at)
VALUES ($1, NOW(), NOW(), $2, $3)
RETURNING *;

-- name: GetToken :one
SELECT * FROM refresh_tokens WHERE token = $1;

-- name: UpdateToken :one
UPDATE refresh_tokens 
SET revoked_at = $1, updated_at = $2
WHERE token = $3 
RETURNING *;