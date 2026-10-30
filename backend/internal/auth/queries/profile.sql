-- name: GetUser :one
SELECT * FROM users
WHERE id = $1;

-- name: UpdateUserProfile :one
-- Partial profile update (R-02): every column keeps its value when the
-- nullable parameter is absent. updated_at is set explicitly (no triggers,
-- SQL auditable — docs/stack/backend-go.md).
UPDATE users SET
    display_name              = COALESCE(sqlc.narg('display_name'), display_name),
    checkin_time              = COALESCE(sqlc.narg('checkin_time'), checkin_time),
    escalation_window_minutes = COALESCE(sqlc.narg('escalation_window_minutes'), escalation_window_minutes),
    timezone                  = COALESCE(sqlc.narg('timezone'), timezone),
    updated_at                = now()
WHERE id = sqlc.arg('id')
RETURNING *;
