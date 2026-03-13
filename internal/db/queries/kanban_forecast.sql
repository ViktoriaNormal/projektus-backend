-- Kanban Monte Carlo forecast cache

-- name: SaveForecastCache :one
INSERT INTO kanban_forecast_cache (project_id, work_item_count, forecast_data, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetForecastCache :one
SELECT *
FROM kanban_forecast_cache
WHERE project_id = $1
  AND work_item_count = $2
  AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1;

-- name: CleanExpiredForecastCache :exec
DELETE FROM kanban_forecast_cache
WHERE expires_at <= NOW();

