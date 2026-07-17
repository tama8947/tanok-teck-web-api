-- name: CreateEvent :exec
INSERT INTO events (id, type, path, visitor_id, session_id, country, city, region, element, metadata, user_agent, referrer, device, browser, os, x, y, scroll_depth, utm_source, utm_medium, utm_campaign, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, NOW());

-- name: GetSessionByID :one
SELECT * FROM sessions WHERE id = $1;

-- name: CreateSession :exec
INSERT INTO sessions (id, visitor_id, start_time, end_time, page_count, bounce)
VALUES ($1, $2, NOW(), NOW(), 1, false);

-- name: UpdateSessionPageCount :exec
UPDATE sessions SET page_count = page_count + 1, end_time = NOW(), bounce = $2 WHERE id = $1;

-- name: GetDailyStatsByDate :one
SELECT * FROM daily_stats WHERE date = $1 AND path = $2;

-- name: UpsertDailyStats :exec
INSERT INTO daily_stats (id, date, path, pageviews, unique_views, clicks, avg_duration, avg_scroll_depth, scroll_depth_breakdown, top_country)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (date, path) DO UPDATE SET pageviews = daily_stats.pageviews + $4, unique_views = $5, clicks = daily_stats.clicks + $6, scroll_depth_breakdown = $9;
