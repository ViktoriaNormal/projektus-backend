package domain

import "time"

type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
)

type TaskStatus string

type Task struct {
	ID         string
	Key        string
	ProjectID  string
	OwnerID    string
	ExecutorID *string
	Name       string
	Description *string
	Deadline   *time.Time
	ColumnID   string
	SwimlaneID *string
	DeletedAt  *time.Time
}

