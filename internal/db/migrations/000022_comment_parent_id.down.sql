DROP INDEX IF EXISTS idx_comments_parent;
ALTER TABLE comments DROP COLUMN IF EXISTS parent_comment_id;
