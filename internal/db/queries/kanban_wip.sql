-- Kanban WIP limits and counts

-- name: GetWipLimits :many
SELECT b.id AS board_id,
       c.id AS column_id,
       NULL::uuid AS swimlane_id,
       COALESCE(c.wip_limit, 0)::int AS limit
FROM columns c
JOIN boards b ON c.board_id = b.id
WHERE b.project_id = $1
UNION ALL
SELECT b2.id AS board_id,
       NULL::uuid AS column_id,
       s.id AS swimlane_id,
       COALESCE(s.wip_limit, 0)::int AS limit
FROM swimlanes s
JOIN boards b2 ON s.board_id = b2.id
WHERE b2.project_id = $1
ORDER BY board_id, column_id, swimlane_id;

-- name: UpdateColumnWipLimit :exec
UPDATE columns
SET wip_limit  = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateSwimlaneWipLimit :exec
UPDATE swimlanes
SET wip_limit  = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: GetCurrentWipCounts :many
SELECT c.board_id,
       t.column_id,
       t.swimlane_id,
       COUNT(*)::int AS count
FROM tasks t
JOIN columns c ON t.column_id = c.id
WHERE c.board_id = $1
  AND t.deleted_at IS NULL
GROUP BY c.board_id, t.column_id, t.swimlane_id;

