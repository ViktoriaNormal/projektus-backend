package domain

import "time"

type SystemStatusType string

const (
	StatusInitial    SystemStatusType = "initial"
	StatusInProgress SystemStatusType = "in_progress"
	StatusPaused     SystemStatusType = "paused"
	StatusCompleted  SystemStatusType = "completed"
	StatusCancelled  SystemStatusType = "cancelled"
)

type Board struct {
	ID          string
	ProjectID   *string
	TemplateID  *string
	Name        string
	Description *string
	Order       int16
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Column struct {
	ID         string
	BoardID    string
	Name       string
	SystemType *SystemStatusType
	WipLimit   *int16
	Order      int16
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Swimlane struct {
	ID        string
	BoardID   string
	Name      string
	WipLimit  *int16
	Order     int16
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Note struct {
	ID         string
	ColumnID   *string
	SwimlaneID *string
	Content    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

