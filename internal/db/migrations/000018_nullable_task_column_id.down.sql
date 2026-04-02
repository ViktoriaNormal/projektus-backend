-- Assign NULL column tasks to first initial column of their board before restoring NOT NULL.
UPDATE tasks t
SET column_id = (
    SELECT c.id FROM columns c
    WHERE c.board_id = t.board_id AND c.system_type = 'initial'
    ORDER BY c.sort_order ASC LIMIT 1
)
WHERE t.column_id IS NULL;

ALTER TABLE tasks ALTER COLUMN column_id SET NOT NULL;
