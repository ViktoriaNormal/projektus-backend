-- name: CreateChecklist :one
INSERT INTO checklists (task_id, name)
VALUES ($1, $2)
RETURNING id, task_id, name, created_at;

-- name: ListChecklistsByTask :many
SELECT id, task_id, name, created_at
FROM checklists
WHERE task_id = $1
ORDER BY created_at;

-- name: UpdateChecklistName :one
UPDATE checklists SET name = $2 WHERE id = $1
RETURNING id, task_id, name, created_at;

-- name: DeleteChecklist :exec
DELETE FROM checklists WHERE id = $1;

-- name: CreateChecklistItem :one
INSERT INTO checklist_items (checklist_id, content, is_checked, sort_order)
VALUES ($1, $2, $3, $4)
RETURNING id, checklist_id, content, is_checked, sort_order;

-- name: ListChecklistItems :many
SELECT id, checklist_id, content, is_checked, sort_order
FROM checklist_items
WHERE checklist_id = $1
ORDER BY sort_order;

-- name: UpdateChecklistItemStatus :one
UPDATE checklist_items
SET is_checked = $2
WHERE id = $1
RETURNING id, checklist_id, content, is_checked, sort_order;

-- name: UpdateChecklistItemContent :one
UPDATE checklist_items SET content = $2 WHERE id = $1
RETURNING id, checklist_id, content, is_checked, sort_order;

-- name: DeleteChecklistItem :exec
DELETE FROM checklist_items WHERE id = $1;
