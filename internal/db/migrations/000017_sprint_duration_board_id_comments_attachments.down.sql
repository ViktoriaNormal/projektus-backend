DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS comments;
ALTER TABLE tasks DROP COLUMN IF EXISTS board_id;
ALTER TABLE projects DROP COLUMN IF EXISTS sprint_duration_weeks;
