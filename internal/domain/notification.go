package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	// Task-related events
	EventTaskAssigned            EventType = "task_assigned"
	EventCommentMention          EventType = "comment_mention"
	EventTaskStatusChangeAuthor  EventType = "task_status_change_author"
	EventTaskStatusChangeAssignee EventType = "task_status_change_assignee"
	EventTaskStatusChangeWatcher EventType = "task_status_change_watcher"
	// Meeting-related events
	EventMeetingInvite EventType = "meeting_invite"
	EventMeetingChange EventType = "meeting_change"
	EventMeetingCancel EventType = "meeting_cancel"
)

type ChannelType string

const (
	ChannelSystem ChannelType = "system"
	ChannelEmail  ChannelType = "email"
)

type NotificationSetting struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	EventType EventType `json:"event_type"`
	InSystem  bool      `json:"in_system"`
	InEmail   bool      `json:"in_email"`
}

type Notification struct {
	ID          uuid.UUID   `json:"id"`
	UserID      uuid.UUID   `json:"user_id"`
	EventType   EventType   `json:"event_type"`
	Channel     ChannelType `json:"channel"`
	Title       string      `json:"title"`
	Body        *string     `json:"body,omitempty"`
	PayloadJSON []byte      `json:"-"`
	IsRead      bool        `json:"is_read"`
	CreatedAt   time.Time   `json:"created_at"`
}

// NotificationPayload is the structured data stored in the notifications.payload JSONB column.
type NotificationPayload struct {
	TaskID           *string `json:"task_id,omitempty"`
	TaskKey          *string `json:"task_key,omitempty"`
	MeetingID        *string `json:"meeting_id,omitempty"`
	MeetingName      *string `json:"meeting_name,omitempty"`
	MeetingStartTime *string `json:"meeting_start_time,omitempty"`
}
