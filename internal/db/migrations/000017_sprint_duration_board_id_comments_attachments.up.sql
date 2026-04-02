-- Migration 000017: sprint_duration_weeks, board_id on tasks, comments, attachments

-- ============================================================
-- 1. Add sprint_duration_weeks to projects (default 2 weeks for Scrum)
-- ============================================================
ALTER TABLE projects ADD COLUMN IF NOT EXISTS sprint_duration_weeks INT DEFAULT 2;

-- ============================================================
-- 2. Add board_id to tasks (direct reference, avoids join through columns)
-- ============================================================
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS board_id UUID REFERENCES boards(id) ON DELETE CASCADE;

-- Populate board_id from column_id → columns.board_id for existing tasks.
UPDATE tasks t
SET board_id = c.board_id
FROM columns c
WHERE t.column_id = c.id AND t.board_id IS NULL;

-- Now make it NOT NULL.
ALTER TABLE tasks ALTER COLUMN board_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_tasks_board ON tasks(board_id);

-- ============================================================
-- 3. Create comments table
-- ============================================================
CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_comments_task ON comments(task_id);

-- ============================================================
-- 4. Create attachments table
-- ============================================================
CREATE TABLE IF NOT EXISTS attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    file_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    content_type TEXT NOT NULL DEFAULT '',
    uploaded_by UUID NOT NULL REFERENCES users(id),
    uploaded_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CHECK (task_id IS NOT NULL OR comment_id IS NOT NULL)
);
CREATE INDEX IF NOT EXISTS idx_attachments_task ON attachments(task_id) WHERE task_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_attachments_comment ON attachments(comment_id) WHERE comment_id IS NOT NULL;
