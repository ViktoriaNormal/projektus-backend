package domain

import "time"

type EventType string

const (
	// Task-related events
	EventTaskAssigned              EventType = "task_assigned"
	EventTaskMentionedInComment    EventType = "task_mentioned_in_comment"
	EventTaskStatusChangedAuthor   EventType = "task_status_changed_author"
	EventTaskStatusChangedAssignee EventType = "task_status_changed_assignee"
	EventTaskStatusChangedWatcher  EventType = "task_status_changed_watcher"
	EventTaskDeadlineApproaching   EventType = "task_deadline_approaching"
	EventTaskDeadlineReached       EventType = "task_deadline_reached"
	// Meeting-related events
	EventMeetingInvitationReceived EventType = "meeting_invitation_received"
	EventMeetingUpdated            EventType = "meeting_updated"
	EventMeetingCancelled          EventType = "meeting_cancelled"
	EventMeetingReminder           EventType = "meeting_reminder"
)

type ChannelType string

const (
	ChannelSystem ChannelType = "system"
	ChannelEmail  ChannelType = "email"
)

type NotificationSetting struct {
	ID                    string    `json:"id"`
	UserID                string    `json:"user_id"`
	EventType             EventType `json:"event_type"`
	InSystem              bool      `json:"in_system"`
	InEmail               bool      `json:"in_email"`
	ReminderOffsetMinutes *int      `json:"reminder_offset_minutes,omitempty"`
}

type Notification struct {
	ID          string      `json:"id"`
	UserID      string      `json:"user_id"`
	EventType   EventType   `json:"event_type"`
	Channel     ChannelType `json:"channel"`
	Title       string      `json:"title"`
	Body        *string     `json:"body,omitempty"`
	PayloadJSON []byte      `json:"-"`
	IsRead      bool        `json:"is_read"`
	CreatedAt   time.Time   `json:"created_at"`
}
