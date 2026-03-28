package dto

import "time"

type NotificationResponse struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	EventType string     `json:"event_type"`
	Channel   string     `json:"channel"`
	Title     string     `json:"title"`
	Body      *string    `json:"body,omitempty"`
	IsRead    bool       `json:"is_read"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
}

type NotificationFeedResponse struct {
	Items       []NotificationResponse `json:"items"`
	UnreadCount int                    `json:"unread_count"`
}

type NotificationSettingResponse struct {
	ID                    string `json:"id"`
	UserID                string `json:"user_id"`
	EventType             string `json:"event_type"`
	InSystem              bool   `json:"in_system"`
	InEmail               bool   `json:"in_email"`
	ReminderOffsetMinutes *int   `json:"reminder_offset_minutes,omitempty"`
}

type UpdateNotificationSettingItem struct {
	EventType             string `json:"event_type" binding:"required"`
	InSystem              *bool  `json:"in_system"`
	InEmail               *bool  `json:"in_email"`
	ReminderOffsetMinutes *int   `json:"reminder_offset_minutes"`
}
