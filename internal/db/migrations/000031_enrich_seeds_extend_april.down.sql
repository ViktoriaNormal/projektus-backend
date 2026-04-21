-- Откат миграции 000031: снимаем CHECK на estimation и удаляем добавленные
-- встречи/чек-листы/зависимости. Нормализованные значения estimation и дедлайны
-- не восстанавливаются — исходные «мусорные» строки типа «4ч» к тому моменту
-- уже потеряны (и это правильно, потому что CHECK их всё равно не пустит).

ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_estimation_numeric;

-- Удаляем встречи и их участников, добавленные миграцией 000031
-- (отличаются UUID-префиксами a5001000, a5301000, и фикс. ретро спринта 6).
DELETE FROM meeting_participants
WHERE meeting_id IN (
    SELECT id FROM meetings
    WHERE id::text LIKE 'a5001000-%'
       OR id::text LIKE 'a5301000-%'
       OR id IN (
            'a5200000-0000-0000-0001-000000000006'::uuid,
            'a5200000-0000-0000-0002-000000000006'::uuid
       )
);

DELETE FROM meetings
WHERE id::text LIKE 'a5001000-%'
   OR id::text LIKE 'a5301000-%'
   OR id IN (
        'a5200000-0000-0000-0001-000000000006'::uuid,
        'a5200000-0000-0000-0002-000000000006'::uuid
   );

-- Чек-листы и пункты — удаляем все записи, созданные через эту миграцию.
-- Поскольку в сидах 000026/000027 их ещё не было, смело сносим всё содержимое
-- этих таблиц; при необходимости bootstrap/тесты создадут свои.
TRUNCATE checklist_items, checklists CASCADE;

-- Task dependencies — удаляем все записи (в сидах 000026/000027 их не было).
TRUNCATE task_dependencies CASCADE;

-- Watchers — удаляем все записи (в сидах 000026/000027 их не было).
TRUNCATE task_watchers CASCADE;
