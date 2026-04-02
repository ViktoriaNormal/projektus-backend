-- Revert dependency type changes
UPDATE task_dependencies SET dependency_type = 'related' WHERE dependency_type = 'relates_to';
UPDATE task_dependencies SET dependency_type = 'child' WHERE dependency_type = 'subtask';
DELETE FROM task_dependencies WHERE dependency_type = 'is_blocked_by';

ALTER TABLE task_dependencies DROP CONSTRAINT IF EXISTS task_dependencies_type_check;
ALTER TABLE task_dependencies ADD CONSTRAINT task_dependencies_type_check
    CHECK (dependency_type IN ('blocks', 'blocked_by', 'related'));
