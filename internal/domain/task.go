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
	ID               string      `json:"id"`
	Key              string      `json:"key"`
	ProjectID        string      `json:"project_id"`
	BoardID          string      `json:"board_id"`
	OwnerID          string      `json:"owner_id"`
	ExecutorID       *string     `json:"executor_id,omitempty"`
	OwnerUserID      *string     `json:"owner_user_id,omitempty"`
	ExecutorUserID   *string     `json:"executor_user_id,omitempty"`
	Name             string      `json:"name"`
	Description      *string     `json:"description,omitempty"`
	Deadline         *time.Time  `json:"deadline,omitempty"`
	ColumnID         *string     `json:"column_id,omitempty"`
	SwimlaneID       *string     `json:"swimlane_id,omitempty"`
	DeletedAt        *time.Time  `json:"-"`
	CreatedAt        time.Time   `json:"created_at"`
	Priority         *string     `json:"priority,omitempty"`
	Estimation       *string     `json:"estimation,omitempty"`
	Checklists       []Checklist `json:"checklists,omitempty"`
	StoryPoints      *int        `json:"story_points,omitempty"`
	ColumnName       *string     `json:"column_name,omitempty"`
	ColumnSystemType *string     `json:"column_system_type,omitempty"`
	Tags             []Tag       `json:"tags,omitempty"`
}

type BacklogType string

const (
	BacklogTypeProduct BacklogType = "product"
	BacklogTypeSprint  BacklogType = "sprint"
)

type TaskDependencyType string

const (
	TaskDependencyBlocks      TaskDependencyType = "blocks"
	TaskDependencyIsBlockedBy TaskDependencyType = "is_blocked_by"
	TaskDependencyRelatesTo   TaskDependencyType = "relates_to"
	TaskDependencyParent      TaskDependencyType = "parent"
	TaskDependencySubtask     TaskDependencyType = "subtask"
)

type TaskWatcher struct {
	TaskID   string `json:"task_id"`
	MemberID string `json:"member_id"`
}

type TaskFieldValue struct {
	TaskID        string     `json:"task_id"`
	FieldID       string     `json:"field_id"`
	ValueText     *string    `json:"value_text,omitempty"`
	ValueNumber   *string    `json:"value_number,omitempty"`
	ValueDatetime *time.Time `json:"value_datetime,omitempty"`
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

type SprintTaskWithoutColumn struct {
	TaskID  string
	BoardID string
}
