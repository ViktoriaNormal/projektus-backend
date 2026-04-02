-- Migration 000020: Update dependency types
-- Rename: related → relates_to, child → subtask
-- Add: is_blocked_by

UPDATE task_dependencies SET dependency_type = 'relates_to' WHERE dependency_type = 'related';
UPDATE task_dependencies SET dependency_type = 'subtask' WHERE dependency_type = 'child';

ALTER TABLE task_dependencies DROP CONSTRAINT IF EXISTS task_dependencies_type_check;
ALTER TABLE task_dependencies ADD CONSTRAINT task_dependencies_type_check
    CHECK (dependency_type IN ('blocks', 'is_blocked_by', 'relates_to', 'parent', 'subtask'));
