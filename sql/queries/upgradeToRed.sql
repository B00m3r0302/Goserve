-- name: UpgradeToRed :exec
UPDATE users
SET is_chirpy_rec = true
WHERE id = $1;