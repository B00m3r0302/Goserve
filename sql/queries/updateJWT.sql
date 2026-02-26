-- name: UpdateJWT :exec
UPDATE refresh_tokens
SET
    token = $2,
    updated_at = NOW(),
    expires_at = NOW() + INTERVAL '1 hour',
    revoked_at = NULL
WHERE user_id = $1;