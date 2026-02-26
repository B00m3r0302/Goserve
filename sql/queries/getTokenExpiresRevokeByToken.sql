-- name: GetTokenExpiresRevokeByToken :one
SELECT token, expires_at, revoked_at, user_id
FROM refresh_tokens
WHERE token = $1;