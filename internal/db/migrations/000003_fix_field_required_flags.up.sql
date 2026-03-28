-- Fix is_required flags for system board fields in templates
-- Only truly required: Название (user fills), Автор (auto), Дата создания (auto)
-- All others are optional

-- Scrum board fields
UPDATE fields SET is_required = true  WHERE id = '20000000-0000-0000-0001-000000000001'; -- Название
UPDATE fields SET is_required = false WHERE id = '20000000-0000-0000-0001-000000000002'; -- Описание
UPDATE fields SET is_required = false WHERE id = '20000000-0000-0000-0001-000000000003'; -- Статус задачи
UPDATE fields SET is_required = true  WHERE id = '20000000-0000-0000-0001-000000000004'; -- Автор (auto)
UPDATE fields SET is_required = false WHERE id = '20000000-0000-0000-0001-000000000005'; -- Исполнитель
UPDATE fields SET is_required = false WHERE id = '20000000-0000-0000-0001-000000000006'; -- Наблюдатели
UPDATE fields SET is_required = false WHERE id = '20000000-0000-0000-0001-000000000007'; -- Дедлайн
UPDATE fields SET is_required = false WHERE id = '20000000-0000-0000-0001-000000000008'; -- Приоритизация
UPDATE fields SET is_required = false WHERE id = '20000000-0000-0000-0001-000000000009'; -- Оценка трудозатрат
UPDATE fields SET is_required = false WHERE id = '20000000-0000-0000-0001-000000000010'; -- Спринт
UPDATE fields SET is_required = true  WHERE id = '20000000-0000-0000-0001-000000000011'; -- Дата создания (auto)

-- Kanban board fields
UPDATE fields SET is_required = true  WHERE id = '20000000-0000-0000-0002-000000000001'; -- Название
UPDATE fields SET is_required = false WHERE id = '20000000-0000-0000-0002-000000000002'; -- Описание
UPDATE fields SET is_required = false WHERE id = '20000000-0000-0000-0002-000000000003'; -- Статус задачи
UPDATE fields SET is_required = true  WHERE id = '20000000-0000-0000-0002-000000000004'; -- Автор (auto)
UPDATE fields SET is_required = false WHERE id = '20000000-0000-0000-0002-000000000005'; -- Исполнитель
UPDATE fields SET is_required = false WHERE id = '20000000-0000-0000-0002-000000000006'; -- Наблюдатели
UPDATE fields SET is_required = false WHERE id = '20000000-0000-0000-0002-000000000007'; -- Дедлайн
UPDATE fields SET is_required = false WHERE id = '20000000-0000-0000-0002-000000000008'; -- Приоритизация
UPDATE fields SET is_required = false WHERE id = '20000000-0000-0000-0002-000000000009'; -- Оценка трудозатрат
UPDATE fields SET is_required = true  WHERE id = '20000000-0000-0000-0002-000000000010'; -- Дата создания (auto)
