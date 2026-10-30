-- name: SetUserPause :exec
UPDATE users SET pause_until = $2, updated_at = now()
WHERE id = $1;

-- name: GetPauseUntil :one
SELECT pause_until FROM users
WHERE id = $1;
