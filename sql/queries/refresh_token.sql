-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens(token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
	$1,
	NOW(),
	NOW(),
	$2,
	NOW() + INTERVAL '60days',
	NULL
)
RETURNING *;

-- name: GetRefreshTokenByToken :one
SELECT * FROM refresh_tokens WHERE token = $1;

-- name: GetRefreshTokenByUserId :one
SELECT * FROM refresh_tokens WHERE user_id = $1;

-- name: RefreshTokenByToken :exec
UPDATE refresh_tokens
SET expires_at = NOW() + INTERVAL '60 days', updated_at = NOW()
WHERE token = $1;

-- name: RevokeRefreshTokenByToken :exec
UPDATE refresh_tokens
SET revoked_at = NOW(), updated_at = NOW()
WHERE token = $1;

