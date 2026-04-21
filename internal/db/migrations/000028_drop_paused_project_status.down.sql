-- Откат миграции 000028: возвращаем статус `paused` в CHECK-ограничение
-- на `projects.status`. Данные не восстанавливаются — на момент отката
-- информация об исходных `paused`-проектах уже утеряна.

ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_status_check;
ALTER TABLE projects ADD CONSTRAINT projects_status_check
    CHECK (status IN ('active', 'archived', 'paused'));
