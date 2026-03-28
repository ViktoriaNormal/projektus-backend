-- Откат 000005: восстановить options и описания

-- Статус задачи — восстановить options
UPDATE fields
SET options = '["Начальный", "В работе", "Завершено", "Отменено"]'
WHERE is_system = true AND name = 'Статус задачи' AND kind = 'board_field';

-- Приоритизация Scrum — восстановить options
UPDATE fields
SET options = '["Низкий", "Средний", "Высокий", "Критичный"]',
    description = 'Приоритет задачи'
WHERE id = '20000000-0000-0000-0001-000000000008';

-- Приоритизация Kanban — восстановить options
UPDATE fields
SET options = '["Ускоренный", "С фиксированной датой", "Стандартный", "Нематериальный"]',
    description = 'Класс обслуживания задачи'
WHERE id = '20000000-0000-0000-0002-000000000008';

-- Оценка трудозатрат Scrum
UPDATE fields
SET description = 'Оценка объёма работы в Story Points'
WHERE id = '20000000-0000-0000-0001-000000000009';

-- Оценка трудозатрат Kanban
UPDATE fields
SET description = 'Оценка объёма работы в формате времени'
WHERE id = '20000000-0000-0000-0002-000000000009';

-- Статус проекта — восстановить options
UPDATE fields
SET options = '["Активный", "Архивирован", "Приостановлен"]'
WHERE is_system = true AND name = 'Статус проекта' AND kind = 'project_param';
