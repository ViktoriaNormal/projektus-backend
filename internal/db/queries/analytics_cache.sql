-- Analytics cache storage

-- name: SaveAnalyticsCache :one
INSERT INTO analytics_cache (project_id, report_type, parameters, result_data, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetAnalyticsCache :one
SELECT *
FROM analytics_cache
WHERE project_id = $1
  AND report_type = $2
  AND parameters = $3
  AND expires_at > NOW()
ORDER BY generated_at DESC
LIMIT 1;

-- name: CleanExpiredAnalyticsCache :exec
DELETE FROM analytics_cache
WHERE expires_at <= NOW();

