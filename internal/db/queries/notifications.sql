-- Notification settings

-- name: GetNotificationSettingsByUser :many
SELECT *
FROM notification_settings
WHERE user_id = $1
ORDER BY event_type;

-- name: UpsertNotificationSetting :exec
INSERT INTO notification_settings (user_id, event_type, in_system, in_email, reminder_offset_minutes)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, event_type) DO UPDATE
SET in_system = EXCLUDED.in_system,
    in_email = EXCLUDED.in_email,
    reminder_offset_minutes = EXCLUDED.reminder_offset_minutes,
    updated_at = NOW();

-- name: GetNotificationSetting :one
SELECT *
FROM notification_settings
WHERE user_id = $1 AND event_type = $2;

-- Notification feed

-- name: CreateNotification :one
INSERT INTO notifications (user_id, event_type, channel, title, body, payload, email_status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetUserNotifications :many
SELECT *
FROM notifications
WHERE user_id = $1
  AND ($2::bool IS FALSE OR is_read = FALSE)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: MarkNotificationAsRead :exec
UPDATE notifications
SET is_read = TRUE,
    read_at = NOW()
WHERE id = $1
  AND user_id = $2;

-- name: MarkAllNotificationsAsRead :exec
UPDATE notifications
SET is_read = TRUE,
    read_at = NOW()
WHERE user_id = $1
  AND is_read = FALSE;

-- name: GetUnreadNotificationCount :one
SELECT COUNT(*)::INT
FROM notifications
WHERE user_id = $1
  AND is_read = FALSE;

-- name: GetNotificationsForEmail :many
SELECT *
FROM notifications
WHERE channel = 'email'
  AND email_status = 'pending';

