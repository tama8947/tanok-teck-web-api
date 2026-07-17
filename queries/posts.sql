-- name: ListPosts :many
SELECT * FROM posts
WHERE locale = $1 AND status = 'PUBLISHED'
ORDER BY published_at DESC, id DESC
LIMIT $2 OFFSET $3;

-- name: CountPosts :one
SELECT COUNT(*) FROM posts WHERE locale = $1 AND status = 'PUBLISHED';

-- name: GetPostBySlug :one
SELECT * FROM posts WHERE slug = $1 AND locale = $2 AND status = 'PUBLISHED';

-- name: GetSiblingSlug :one
SELECT slug, locale FROM posts WHERE translation_group_id = $1 AND locale != $2 AND status = 'PUBLISHED' LIMIT 1;

-- name: GetPublishedPostIDs :many
SELECT id FROM posts WHERE status = 'PUBLISHED' AND index_now_submitted_at IS NULL;

-- name: UpdateIndexNowTimestamp :exec
UPDATE posts SET index_now_submitted_at = NOW() WHERE id = ANY($1::text[]);
