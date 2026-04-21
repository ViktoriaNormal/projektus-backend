-- Откат миграции 000034: удаляем корпоративные еженедельные встречи
-- и связанные приглашения (префикс UUID a7000000-...).

DELETE FROM meeting_participants mp
USING meetings m
WHERE mp.meeting_id = m.id
  AND m.id::text LIKE 'a7000000-%';

DELETE FROM meetings WHERE id::text LIKE 'a7000000-%';
