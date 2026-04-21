-- Откат миграции 000029: удаляем индекс и столбец soft-delete.
-- Проекты, помеченные как удалённые, после отката снова станут видны —
-- перед откатом их следует реально удалить или перенести.

DROP INDEX IF EXISTS idx_projects_deleted;
ALTER TABLE projects DROP COLUMN IF EXISTS deleted_at;
