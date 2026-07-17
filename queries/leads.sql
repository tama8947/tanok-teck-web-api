-- name: GetLeadByEmail :one
SELECT * FROM leads WHERE email = $1;

-- name: CreateLead :one
INSERT INTO leads (id, name, email, company, referrer, lead_magnet_id)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: CreateLeadEvent :exec
INSERT INTO lead_events (id, lead_id, type, referrer, post_id, created_at)
VALUES ($1, $2, $3, $4, $5, NOW());

-- name: GetLeadMagnetByID :one
SELECT * FROM lead_magnets WHERE id = $1;
