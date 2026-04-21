-- Migration 000029: soft-delete для проектов.
-- Добавляем столбец `deleted_at` и индекс на «живые» проекты. Жёсткое удаление
-- остаётся доступным только на уровне SQL (прямые миграции/техобслуживание),
-- прикладной код переходит на UPDATE ... SET deleted_at = NOW().

ALTER TABLE projects ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Частичный индекс ускоряет «горячие» выборки активных проектов
-- (ListUserProjects, ListAllProjects, аналитика).
CREATE INDEX IF NOT EXISTS idx_projects_deleted ON projects (deleted_at) WHERE deleted_at IS NULL;
