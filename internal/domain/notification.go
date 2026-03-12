package domain

import "time"

type EventType string

const (
	// Task-related events
	EventTaskAssigned                EventType = "task_assigned"
	EventTaskMentionedInComment      EventType = "task_mentioned_in_comment"
	EventTaskStatusChangedAuthor     EventType = "task_status_changed_author"
	EventTaskStatusChangedAssignee   EventType = "task_status_changed_assignee"
	EventTaskStatusChangedWatcher    EventType = "task_status_changed_watcher"
	EventTaskDeadlineApproaching     EventType = "task_deadline_approaching"
	EventTaskDeadlineReached         EventType = "task_deadline_reached"
	// Meeting-related events
	EventMeetingInvitationReceived   EventType = "meeting_invitation_received"
	EventMeetingUpdated              EventType = "meeting_updated"
	EventMeetingCancelled            EventType = "meeting_cancelled"
	EventMeetingReminder             EventType = "meeting_reminder"
)

type ChannelType string

const (
	ChannelSystem ChannelType = "system"
	ChannelEmail  ChannelType = "email"
)

type NotificationSetting struct {
	ID                    string
	UserID                string
	EventType             EventType
	InSystem              bool
	InEmail               bool
	ReminderOffsetMinutes *int // nil, если не применяется
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type Notification struct {
	ID          string
	UserID      string
	EventType   EventType
	Channel     ChannelType
	Title       string
	Body        *string
	PayloadJSON []byte
	IsRead      bool
	CreatedAt   time.Time
	ReadAt      *time.Time
	EmailStatus *string
	EmailSentAt *time.Time
}

