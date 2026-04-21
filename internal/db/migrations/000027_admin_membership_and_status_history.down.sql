-- Откат: удаляем то, что добавлено. Каскады почистят зависимости.
DELETE FROM task_status_history;

DELETE FROM tasks WHERE id::text LIKE 'e2000000-%';

DELETE FROM boards WHERE id::text LIKE 'd0000000-0000-0001-%';
