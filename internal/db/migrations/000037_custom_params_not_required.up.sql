-- Migration 000037: Приводит существующие данные в соответствие с новой
-- политикой «кастомные параметры задач и проектов не могут быть
-- обязательными» (обязательными разрешено быть только системным параметрам).
--
-- Таблицы board_fields и project_params хранят ТОЛЬКО кастомные записи —
-- системные поля/параметры генерируются в коде (см. domain/system_fields.go)
-- и в БД не лежат. Поэтому фильтра по is_system здесь нет: он был бы
-- тавтологичным. Обе таблицы покрывают и записи реальных проектов, и
-- записи шаблонов — различаются через project_id / template_id.
--
-- До применения имеет смысл снять счётчики, сколько записей затронет:
--   SELECT COUNT(*) FROM board_fields   WHERE is_required = true;
--   SELECT COUNT(*) FROM project_params WHERE is_required = true;

UPDATE board_fields   SET is_required = false WHERE is_required = true;
UPDATE project_params SET is_required = false WHERE is_required = true;
