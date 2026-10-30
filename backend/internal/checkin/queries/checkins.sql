-- name: CreateCheckin :one
INSERT INTO checkins (id, user_id, cycle_date, checked_in_at, idempotency_key)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetCheckinByDate :one
SELECT * FROM checkins
WHERE user_id = $1 AND cycle_date = $2;

-- name: GetCheckinByIdempotencyKey :one
SELECT * FROM checkins
WHERE user_id = $1 AND idempotency_key = $2;
