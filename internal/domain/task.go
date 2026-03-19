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
	ID          string      `json:"id"`
	Key         string      `json:"key"`
	ProjectID   string      `json:"project_id"`
	OwnerID     string      `json:"owner_id"`
	ExecutorID  *string     `json:"executor_id,omitempty"`
	Name        string      `json:"name"`
	Description *string     `json:"description,omitempty"`
	Deadline    *time.Time  `json:"deadline,omitempty"`
	ColumnID    string      `json:"column_id"`
	SwimlaneID  *string     `json:"swimlane_id,omitempty"`
	DeletedAt   *time.Time  `json:"-"`
	Checklists  []Checklist `json:"checklists,omitempty"`
	StoryPoints *int        `json:"story_points,omitempty"`
}

type BacklogType string

const (
	BacklogTypeProduct BacklogType = "product"
	BacklogTypeSprint  BacklogType = "sprint"
)

type TaskDependencyType string

const (
	TaskDependencyBlocks  TaskDependencyType = "blocks"
	TaskDependencyRelated TaskDependencyType = "related"
	TaskDependencyParent  TaskDependencyType = "parent"
	TaskDependencyChild   TaskDependencyType = "child"
)

type TaskWatcher struct {
	ID              string    `json:"id"`
	TaskID          string    `json:"task_id"`
	ProjectMemberID string    `json:"project_member_id"`
	CreatedAt       time.Time `json:"created_at"`
}

type TaskDependency struct {
	ID              string             `json:"id"`
	TaskID          string             `json:"task_id"`
	DependsOnTaskID string             `json:"depends_on_task_id"`
	Type            TaskDependencyType `json:"type"`
	CreatedAt       time.Time          `json:"created_at"`
}

type Checklist struct {
	ID     string          `json:"id"`
	TaskID string          `json:"task_id"`
	Name   string          `json:"name"`
	Items  []ChecklistItem `json:"items,omitempty"`
}

type ChecklistItem struct {
	ID          string `json:"id"`
	ChecklistID string `json:"checklist_id"`
	Content     string `json:"content"`
	IsChecked   bool   `json:"is_checked"`
	Order       int16  `json:"order"`
}
