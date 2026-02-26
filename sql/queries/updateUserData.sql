-- name: UpdateUserData :exec
UPDATE users
SET
    email = $2,
    updated_at = NOW(),
    hashed_password = $3
WHERE id = $1;
