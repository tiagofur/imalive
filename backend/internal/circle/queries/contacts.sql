-- name: CreateContact :one
INSERT INTO contacts (id, user_id, display_name, phone, position)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetContact :one
SELECT * FROM contacts
WHERE id = $1 AND user_id = $2;

-- name: ListContactsByUser :many
SELECT * FROM contacts
WHERE user_id = $1
ORDER BY position ASC;

-- name: DeleteContact :execrows
DELETE FROM contacts
WHERE id = $1 AND user_id = $2;

-- name: CountContactsByUser :one
SELECT count(*) FROM contacts
WHERE user_id = $1;

-- name: NextContactPosition :one
SELECT COALESCE(max(position), 0) + 1 AS next FROM contacts
WHERE user_id = $1;

-- name: GetUserName :one
SELECT display_name FROM users
WHERE id = $1;
