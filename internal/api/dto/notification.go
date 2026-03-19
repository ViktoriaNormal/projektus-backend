package dto

import "time"

type NotificationResponse struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	EventType string     `json:"eventType"`
	Channel   string     `json:"channel"`
	Title     string     `json:"title"`
	Body      *string    `json:"body,omitempty"`
	IsRead    bool       `json:"isRead"`
	CreatedAt time.Time  `json:"createdAt"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
}

type NotificationFeedResponse struct {
	Items       []NotificationResponse `json:"items"`
	UnreadCount int                    `json:"unreadCount"`
}

type NotificationSettingResponse struct {
	ID                    string `json:"id"`
	UserID                string `json:"userId"`
	EventType             string `json:"eventType"`
	InSystem              bool   `json:"inSystem"`
	InEmail               bool   `json:"inEmail"`
	ReminderOffsetMinutes *int   `json:"reminderOffsetMinutes,omitempty"`
}

type UpdateNotificationSettingItem struct {
	EventType             string `json:"eventType" binding:"required"`
	InSystem              *bool  `json:"inSystem"`
	InEmail               *bool  `json:"inEmail"`
	ReminderOffsetMinutes *int   `json:"reminderOffsetMinutes"`
}
