UPDATE fields SET field_type = 'select' WHERE is_system = true AND name = 'Приоритизация' AND field_type = 'priority';
UPDATE fields SET field_type = 'number' WHERE is_system = true AND name = 'Оценка трудозатрат' AND field_type = 'estimation';
