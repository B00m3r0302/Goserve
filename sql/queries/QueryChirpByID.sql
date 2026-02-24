-- name: QueryChirpById :one
SELECT * FROM chirps WHERE id = $1;