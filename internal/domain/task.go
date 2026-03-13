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
	Checklists []Checklist
}

type TaskDependencyType string

const (
	TaskDependencyBlocks TaskDependencyType = "blocks"
	TaskDependencyRelated TaskDependencyType = "related"
	TaskDependencyParent  TaskDependencyType = "parent"
	TaskDependencyChild   TaskDependencyType = "child"
)

type TaskWatcher struct {
	ID              string
	TaskID          string
	ProjectMemberID string
	CreatedAt       time.Time
}

type TaskDependency struct {
	ID              string
	TaskID          string
	DependsOnTaskID string
	Type            TaskDependencyType
	CreatedAt       time.Time
}

type Checklist struct {
	ID      string
	TaskID  string
	Name    string
	Items   []ChecklistItem
}

type ChecklistItem struct {
	ID         string
	ChecklistID string
	Content    string
	IsChecked  bool
	Order      int16
}

