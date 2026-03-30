-- =============================================================================
-- 000008: Использовать специальные field_type для системных полей
-- =============================================================================

-- Приоритизация: select → priority
UPDATE fields
SET field_type = 'priority'
WHERE is_system = true AND name = 'Приоритизация' AND field_type = 'select';

-- Оценка трудозатрат: number → estimation
UPDATE fields
SET field_type = 'estimation'
WHERE is_system = true AND name = 'Оценка трудозатрат' AND field_type = 'number';
