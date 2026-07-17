-- name: GetCronRunBySlot :one
SELECT * FROM cron_runs WHERE slot = $1;

-- name: CreateCronRun :exec
INSERT INTO cron_runs (id, slot, status, created_at) VALUES ($1, $2, 'running', NOW());

-- name: UpdateCronRunSuccess :exec
UPDATE cron_runs SET status = 'SUCCESS', post_id = $2 WHERE id = $1;

-- name: UpdateCronRunFailed :exec
UPDATE cron_runs SET status = 'FAILED', error = $2 WHERE id = $1;

-- name: PickNextTopic :one
SELECT * FROM cron_topics WHERE is_active = true AND used_at IS NULL ORDER BY use_count ASC, created_at ASC LIMIT 1;

-- name: MarkTopicUsed :exec
UPDATE cron_topics SET used_at = NOW(), use_count = use_count + 1 WHERE id = $1;
