-- Откат миграции 000033: удаляем добавленные уведомления и возвращаем
-- настройки уведомлений к исходному состоянию «все включено в системе,
-- e-mail выключен» (поведение 000026).

DELETE FROM notifications
WHERE is_read = false
  AND created_at >= TIMESTAMP '2026-04-09 00:00:00';  -- диапазон, в который мы добавляли

UPDATE notification_settings
SET in_system = true,
    in_email = false;
