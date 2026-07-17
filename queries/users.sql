-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetFirstAdmin :one
SELECT * FROM users WHERE permissions LIKE '%dashboard%' ORDER BY created_at ASC LIMIT 1;
