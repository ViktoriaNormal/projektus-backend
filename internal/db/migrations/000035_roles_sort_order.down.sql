-- Откат миграции 000035. Снова удаляем sort_order.

DROP INDEX IF EXISTS idx_roles_template_sort;
DROP INDEX IF EXISTS idx_roles_project_sort;
ALTER TABLE roles DROP COLUMN IF EXISTS sort_order;
