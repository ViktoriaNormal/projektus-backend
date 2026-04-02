-- name: ListTaskWatchers :many
SELECT task_id, member_id
FROM task_watchers
WHERE task_id = $1;

-- name: AddTaskWatcher :exec
INSERT INTO task_watchers (task_id, member_id)
VALUES ($1, $2)
ON CONFLICT (task_id, member_id) DO NOTHING;

-- name: RemoveTaskWatcher :exec
DELETE FROM task_watchers
WHERE task_id = $1 AND member_id = $2;
