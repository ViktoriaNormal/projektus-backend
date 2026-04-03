package dto

type NotificationResponse struct {
	ID                string  `json:"id"`
	Type              string  `json:"type"`
	Message           string  `json:"message"`
	Read              bool    `json:"read"`
	CreatedAt         string  `json:"created_at"`
	TaskID            *string `json:"task_id"`
	TaskKey           *string `json:"task_key"`
	MeetingID         *string `json:"meeting_id"`
	MeetingName       *string `json:"meeting_name"`
	MeetingStartTime  *string `json:"meeting_start_time,omitempty"`
	ParticipantStatus *string `json:"participant_status,omitempty"`
}

type NotificationFeedResponse struct {
	Items       []NotificationResponse `json:"items"`
	UnreadCount int                    `json:"unread_count"`
}

type NotificationSettingResponse struct {
	EventType string `json:"event_type"`
	Enabled   bool   `json:"enabled"`
}

type UpdateNotificationSettingItem struct {
	EventType string `json:"event_type" binding:"required"`
	Enabled   *bool  `json:"enabled"`
}
