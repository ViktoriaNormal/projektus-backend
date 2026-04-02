-- Down: restore is_system column and priority_options.
-- System field records cannot be restored — run seed migration if needed.
ALTER TABLE fields ADD COLUMN IF NOT EXISTS is_system BOOLEAN DEFAULT false NOT NULL;
ALTER TABLE boards DROP COLUMN IF EXISTS priority_options;
