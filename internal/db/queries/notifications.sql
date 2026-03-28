-- Notification settings

-- name: GetNotificationSettingsByUser :many
SELECT id, user_id, event_type, in_system, in_email, reminder_offset_minutes
FROM notification_settings
WHERE user_id = $1
ORDER BY event_type;

-- name: UpsertNotificationSetting :exec
INSERT INTO notification_settings (user_id, event_type, in_system, in_email, reminder_offset_minutes)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, event_type) DO UPDATE
SET in_system = EXCLUDED.in_system,
    in_email = EXCLUDED.in_email,
    reminder_offset_minutes = EXCLUDED.reminder_offset_minutes;

-- name: GetNotificationSetting :one
SELECT id, user_id, event_type, in_system, in_email, reminder_offset_minutes
FROM notification_settings
WHERE user_id = $1 AND event_type = $2;

-- Notification feed

-- name: CreateNotification :one
INSERT INTO notifications (user_id, event_type, title, body, payload)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, event_type, title, body, payload, is_read, created_at;

-- name: GetUserNotifications :many
SELECT id, user_id, event_type, title, body, payload, is_read, created_at
FROM notifications
WHERE user_id = $1
  AND ($2::bool IS FALSE OR is_read = FALSE)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: MarkNotificationAsRead :exec
UPDATE notifications
SET is_read = TRUE
WHERE id = $1
  AND user_id = $2;

-- name: MarkAllNotificationsAsRead :exec
UPDATE notifications
SET is_read = TRUE
WHERE user_id = $1
  AND is_read = FALSE;

-- name: GetUnreadNotificationCount :one
SELECT COUNT(*)::INT
FROM notifications
WHERE user_id = $1
  AND is_read = FALSE;
