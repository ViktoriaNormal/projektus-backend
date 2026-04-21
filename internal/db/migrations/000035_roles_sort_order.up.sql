-- Migration 000035: возврат колонки sort_order для стабильного порядка ролей.
--
-- Была удалена в 000004 как «неиспользуемая», но сейчас нужна для консистентного
-- отображения ролей шаблона в админке. При PATCH'е Postgres кладёт обновлённую
-- строку в конец физического хранилища, и без ORDER BY роли видимо «всплывают».

ALTER TABLE roles ADD COLUMN IF NOT EXISTS sort_order INT NOT NULL DEFAULT 0;

-- Фиксируем стартовый порядок ролей в рамках каждого контекста
-- (template_id / project_id / system): админ-роли (is_admin=true) идут сверху
-- — они важнее функционально, в UI логично видеть их первыми. Внутри группы
-- сортируем по имени.
WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY COALESCE(template_id::text, project_id::text, scope)
               ORDER BY is_admin DESC, name ASC, id ASC
           ) AS rn
    FROM roles
)
UPDATE roles r
SET sort_order = ranked.rn
FROM ranked
WHERE r.id = ranked.id;

-- Индексы ускоряют горячие выборки «роли шаблона/проекта с сортировкой».
CREATE INDEX IF NOT EXISTS idx_roles_template_sort ON roles (template_id, sort_order) WHERE template_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_roles_project_sort  ON roles (project_id, sort_order)  WHERE project_id IS NOT NULL;
