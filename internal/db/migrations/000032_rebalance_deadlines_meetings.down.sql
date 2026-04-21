-- Откат миграции 000032: убираем пользовательские встречи и их участников,
-- возвращаем «всех членов проекта» в проектные встречи.
-- Дедлайны задач не восстанавливаем — оригинальные значения к этому моменту
-- уже перезаписаны миграцией 000031 и 000032.

-- 1. Удаляем пользовательские встречи (префикс UUID a6000000-...).
DELETE FROM meeting_participants mp
USING meetings m
WHERE mp.meeting_id = m.id
  AND m.id::text LIKE 'a6000000-%';

DELETE FROM meetings WHERE id::text LIKE 'a6000000-%';

-- 2. Перезаполняем участников проектных встреч «всеми членами проекта»
--    (поведение сидов до миграции 000032).
DELETE FROM meeting_participants mp
USING meetings m
WHERE mp.meeting_id = m.id
  AND m.project_id IS NOT NULL;

INSERT INTO meeting_participants (meeting_id, user_id, status)
SELECT m.id, mm.user_id, 'accepted'
FROM meetings m
JOIN members mm ON mm.project_id = m.project_id
WHERE m.project_id IS NOT NULL
ON CONFLICT (meeting_id, user_id) DO NOTHING;
