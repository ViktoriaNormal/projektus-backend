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
	ID          string    `json:"id"`
	ProjectID   *string   `json:"project_id,omitempty"`
	TemplateID  *string   `json:"template_id,omitempty"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Order       int16     `json:"order"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

type Column struct {
	ID         string            `json:"id"`
	BoardID    string            `json:"board_id"`
	Name       string            `json:"name"`
	SystemType *SystemStatusType `json:"system_type,omitempty"`
	WipLimit   *int16            `json:"wip_limit,omitempty"`
	Order      int16             `json:"order"`
	CreatedAt  time.Time         `json:"-"`
	UpdatedAt  time.Time         `json:"-"`
}

type Swimlane struct {
	ID        string    `json:"id"`
	BoardID   string    `json:"board_id"`
	Name      string    `json:"name"`
	WipLimit  *int16    `json:"wip_limit,omitempty"`
	Order     int16     `json:"order"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type Note struct {
	ID         string    `json:"id"`
	ColumnID   *string   `json:"column_id,omitempty"`
	SwimlaneID *string   `json:"swimlane_id,omitempty"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"-"`
	UpdatedAt  time.Time `json:"-"`
}
