-- Project templates

-- name: ListProjectTemplates :many
SELECT id, name, description, project_type, created_at
FROM project_templates
ORDER BY created_at DESC;

-- name: CreateProjectTemplate :one
INSERT INTO project_templates (name, description, project_type)
VALUES ($1, $2, $3)
RETURNING id, name, description, project_type, created_at;

