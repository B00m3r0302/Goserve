-- name: DeleteChirp :execrows
DELETE FROM chirps
WHERE id = $1
AND user_id = $2;