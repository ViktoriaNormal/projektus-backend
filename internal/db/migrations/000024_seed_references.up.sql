-- Справочные таблицы для шаблонов проектов
-- Encoding: UTF-8

SET client_encoding = 'UTF8';

-- 2.1 Системные типы колонок
CREATE TABLE IF NOT EXISTS ref_column_system_types (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0
);

INSERT INTO ref_column_system_types (key, name, description, sort_order) VALUES
    ('initial', 'Начальный', 'Задача создана, но не взята в работу', 1),
    ('in_progress', 'В работе', 'Задача в процессе выполнения', 2),
    ('completed', 'Выполнено', 'Задача выполнена и закрыта', 3)
;

-- 2.2 Все типы статусов задач
CREATE TABLE IF NOT EXISTS ref_task_status_types (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_column_type BOOLEAN NOT NULL DEFAULT false
);

INSERT INTO ref_task_status_types (key, name, description, is_column_type) VALUES
    ('initial', 'Начальный', 'Задача создана, но не взята в работу', true),
    ('in_progress', 'В работе', 'Задача в процессе выполнения', true),
    ('completed', 'Выполнено', 'Задача выполнена и закрыта', true),
    ('cancelled', 'Отменено', 'Задача не выполнена и закрыта (назначается задаче, не колонке)', false)
;

-- 2.3 Типы кастомных полей задач
CREATE TABLE IF NOT EXISTS ref_field_types (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

INSERT INTO ref_field_types (key, name) VALUES
    ('text', 'Текст'),
    ('number', 'Число'),
    ('datetime', 'Дата и время'),
    ('select', 'Выпадающий список'),
    ('multiselect', 'Множественный выбор'),
    ('checkbox', 'Флажок'),
    ('user', 'Пользователь')
;

-- 2.4 Единицы измерения оценки трудозатрат
CREATE TABLE IF NOT EXISTS ref_estimation_units (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    available_for TEXT[] NOT NULL DEFAULT '{}'
);

INSERT INTO ref_estimation_units (key, name, available_for) VALUES
    ('story_points', 'Story Points', ARRAY['scrum']),
    ('time', 'Время (дни/часы/минуты)', ARRAY['scrum', 'kanban'])
;

-- 2.5 Варианты группировки задач в дорожки
CREATE TABLE IF NOT EXISTS ref_swimlane_group_options (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    available_for TEXT[] NOT NULL DEFAULT '{}'
);

INSERT INTO ref_swimlane_group_options (key, name, available_for) VALUES
    ('priority', 'по приоритету', ARRAY['scrum', 'kanban']),
    ('service_class', 'по классу обслуживания', ARRAY['kanban']),
    ('assignee', 'по исполнителю', ARRAY['scrum', 'kanban']),
    ('type', 'по типу задачи', ARRAY['scrum', 'kanban']),
    ('tags', 'по меткам', ARRAY['scrum', 'kanban'])
;

-- 2.6 Типы приоритизации задач
CREATE TABLE IF NOT EXISTS ref_priority_types (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    available_for TEXT[] NOT NULL DEFAULT '{}',
    default_values TEXT[] NOT NULL DEFAULT '{}'
);

INSERT INTO ref_priority_types (key, name, available_for, default_values) VALUES
    ('priority', 'Приоритет', ARRAY['scrum', 'kanban'], ARRAY['Низкий', 'Средний', 'Высокий', 'Критичный']),
    ('service_class', 'Класс обслуживания', ARRAY['kanban'], ARRAY['Ускоренный', 'С фиксированной датой', 'Стандартный', 'Нематериальный'])
;

-- 2.7 Системные (обязательные) параметры задач
CREATE TABLE IF NOT EXISTS ref_system_task_fields (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    field_type TEXT NOT NULL,
    available_for TEXT[] NOT NULL DEFAULT '{}',
    description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0
);

INSERT INTO ref_system_task_fields (key, name, field_type, available_for, description, sort_order) VALUES
    ('name', 'Название', 'text', ARRAY['scrum', 'kanban'], '', 1),
    ('description', 'Описание', 'text', ARRAY['scrum', 'kanban'], '', 2),
    ('status', 'Статус', 'column', ARRAY['scrum', 'kanban'], 'Определяется колонкой доски', 3),
    ('owner', 'Автор', 'user', ARRAY['scrum', 'kanban'], '', 4),
    ('executor', 'Исполнитель', 'user', ARRAY['scrum', 'kanban'], '', 5),
    ('watchers', 'Наблюдатели', 'user_list', ARRAY['scrum', 'kanban'], '', 6),
    ('deadline', 'Крайний срок выполнения', 'datetime', ARRAY['scrum', 'kanban'], '', 7),
    ('priority', 'Приоритет', 'priority', ARRAY['scrum', 'kanban'], 'Для Kanban может быть заменён на «Класс обслуживания»', 8),
    ('estimation', 'Оценка трудозатрат', 'estimation', ARRAY['scrum', 'kanban'], 'Scrum: Story Points или время; Kanban: только время', 9),
    ('sprint', 'Спринт', 'sprint', ARRAY['scrum'], '', 10)
;
