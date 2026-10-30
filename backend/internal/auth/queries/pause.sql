-- name: SetUserPause :exec
-- $2 NULL = cancelar: limpia también pause_since (la ventana de la racha,
-- R-12); $2 con valor = activar desde ahora.
UPDATE users SET pause_until = $2,
                 pause_since = CASE WHEN $2::timestamptz IS NULL THEN NULL ELSE now() END,
                 updated_at = now()
WHERE id = $1;

-- name: GetPauseUntil :one
SELECT pause_until FROM users
WHERE id = $1;
