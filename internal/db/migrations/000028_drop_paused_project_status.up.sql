-- Migration 000028: удаление статуса `paused` из проектов.
-- Статус `paused` больше не используется продуктом — существующие проекты
-- со статусом `paused` переводятся в `active`, после чего CHECK-ограничение
-- на `projects.status` сужается до набора ('active','archived').

UPDATE projects SET status = 'active' WHERE status = 'paused';

ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_status_check;
ALTER TABLE projects ADD CONSTRAINT projects_status_check
    CHECK (status IN ('active', 'archived'));
