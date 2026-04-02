-- Migration 000018: Make tasks.column_id nullable for backlog tasks.
-- Tasks in product backlog and sprint backlog don't have a column until sprint starts.

ALTER TABLE tasks ALTER COLUMN column_id DROP NOT NULL;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_column_id_fkey;
ALTER TABLE tasks ADD CONSTRAINT tasks_column_id_fkey
    FOREIGN KEY (column_id) REFERENCES columns(id) ON DELETE SET NULL;
