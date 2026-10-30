-- name: CreateUser :one
INSERT INTO users (id, phone, display_name, timezone)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByPhone :one
SELECT * FROM users
WHERE phone = $1;
