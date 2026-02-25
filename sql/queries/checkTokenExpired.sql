-- name: CheckTokenExpired :exec
SELECT expires_at
FROM refresh_tokens
WHERE token = $1
AND expires < NOW();