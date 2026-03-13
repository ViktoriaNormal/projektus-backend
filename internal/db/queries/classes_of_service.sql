-- Classes of service helpers

-- name: UpdateTaskClassOfService :exec
UPDATE tasks
SET class_of_service = $2
WHERE id = $1;

-- name: GetTasksByClassOfService :many
SELECT *
FROM tasks
WHERE project_id = $1
  AND class_of_service = $2
  AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetDefaultClassesOfService :many
SELECT * FROM (
    VALUES
        ('expedite'::text),
        ('fixed_date'::text),
        ('standard'::text),
        ('intangible'::text)
) AS t(class_of_service);

