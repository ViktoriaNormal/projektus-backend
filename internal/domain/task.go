package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
)

type TaskStatus string

type Task struct {
	ID               uuid.UUID   `json:"id"`
	Key              string      `json:"key"`
	ProjectID        uuid.UUID   `json:"project_id"`
	BoardID          uuid.UUID   `json:"board_id"`
	OwnerID          uuid.UUID   `json:"owner_id"`
	ExecutorID       *uuid.UUID  `json:"executor_id,omitempty"`
	OwnerUserID      *uuid.UUID  `json:"owner_user_id,omitempty"`
	ExecutorUserID   *uuid.UUID  `json:"executor_user_id,omitempty"`
	Name             string      `json:"name"`
	Description      *string     `json:"description,omitempty"`
	Deadline         *time.Time  `json:"deadline,omitempty"`
	ColumnID         *uuid.UUID  `json:"column_id,omitempty"`
	SwimlaneID       *uuid.UUID  `json:"swimlane_id,omitempty"`
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
	TaskID   uuid.UUID `json:"task_id"`
	MemberID uuid.UUID `json:"member_id"`
}

type TaskFieldValue struct {
	TaskID        uuid.UUID  `json:"task_id"`
	FieldID       uuid.UUID  `json:"field_id"`
	ValueText     *string    `json:"value_text,omitempty"`
	ValueNumber   *string    `json:"value_number,omitempty"`
	ValueDatetime *time.Time `json:"value_datetime,omitempty"`
}

type TaskDependency struct {
	ID              uuid.UUID          `json:"id"`
	TaskID          uuid.UUID          `json:"task_id"`
	DependsOnTaskID uuid.UUID          `json:"depends_on_task_id"`
	Type            TaskDependencyType `json:"type"`
	CreatedAt       time.Time          `json:"created_at"`
}

type Checklist struct {
	ID     uuid.UUID       `json:"id"`
	TaskID uuid.UUID       `json:"task_id"`
	Name   string          `json:"name"`
	Items  []ChecklistItem `json:"items,omitempty"`
}

type ChecklistItem struct {
	ID          uuid.UUID `json:"id"`
	ChecklistID uuid.UUID `json:"checklist_id"`
	Content     string    `json:"content"`
	IsChecked   bool      `json:"is_checked"`
	Order       int16     `json:"order"`
}

type SprintTaskWithoutColumn struct {
	TaskID  uuid.UUID
	BoardID uuid.UUID
}
