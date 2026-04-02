-- Migration 000016:
-- 1. Add missing system field columns to tasks (priority, estimation, watchers)
-- 2. Remove description from project_params and board_fields

-- ============================================================
-- 1. Add missing columns to tasks
-- ============================================================

-- Priority value (e.g., "Высокий", "Ускоренный")
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS priority TEXT;

-- Estimation value (story points as number string, or time like "2ч 30м")
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS estimation TEXT;

-- ============================================================
-- 2. Create task_watchers junction table
-- ============================================================
CREATE TABLE IF NOT EXISTS task_watchers (
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    member_id UUID NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, member_id)
);
CREATE INDEX IF NOT EXISTS idx_task_watchers_member ON task_watchers(member_id);

-- ============================================================
-- 3. Drop description from project_params and board_fields
-- ============================================================
ALTER TABLE project_params DROP COLUMN IF EXISTS description;
ALTER TABLE board_fields DROP COLUMN IF EXISTS description;
