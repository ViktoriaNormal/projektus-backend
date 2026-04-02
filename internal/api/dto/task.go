package dto

import (
	"time"

	"github.com/google/uuid"
)

type TaskResponse struct {
	ID               uuid.UUID     `json:"id"`
	Key              string        `json:"key"`
	ProjectID        uuid.UUID     `json:"project_id"`
	BoardID          uuid.UUID     `json:"board_id"`
	OwnerMemberID    uuid.UUID     `json:"owner_member_id"`
	ExecutorMemberID *uuid.UUID    `json:"executor_member_id,omitempty"`
	OwnerUserID      *uuid.UUID    `json:"owner_user_id,omitempty"`
	ExecutorUserID   *uuid.UUID    `json:"executor_user_id,omitempty"`
	Name             string        `json:"name"`
	Description      *string       `json:"description,omitempty"`
	Deadline         *time.Time    `json:"deadline,omitempty"`
	ColumnID         *uuid.UUID    `json:"column_id,omitempty"`
	SwimlaneID       *uuid.UUID    `json:"swimlane_id,omitempty"`
	Priority         *string       `json:"priority,omitempty"`
	Estimation       *string       `json:"estimation,omitempty"`
	Progress         *int          `json:"progress,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	ColumnName       *string       `json:"column_name,omitempty"`
	ColumnSystemType *string       `json:"column_system_type,omitempty"`
	Tags             []TagResponse `json:"tags"`
}

type CreateTaskRequest struct {
	ProjectID        uuid.UUID  `json:"project_id" binding:"required"`
	OwnerMemberID    uuid.UUID  `json:"owner_member_id" binding:"required"`
	ExecutorMemberID *uuid.UUID `json:"executor_member_id,omitempty"`
	Name             string     `json:"name" binding:"required"`
	Description      string     `json:"description"`
	Deadline         *time.Time `json:"deadline,omitempty"`
	ColumnID         uuid.UUID  `json:"column_id"`
	BoardID          *uuid.UUID `json:"board_id,omitempty"`
	SwimlaneID       *uuid.UUID `json:"swimlane_id,omitempty"`
	Priority         *string    `json:"priority,omitempty"`
	Estimation       *string    `json:"estimation,omitempty"`

	// Nested entities (optional, created atomically with the task)
	Checklists       []CreateTaskChecklist   `json:"checklists,omitempty"`
	Tags             []string                `json:"tags,omitempty"`
	WatcherMemberIDs []uuid.UUID             `json:"watcher_member_ids,omitempty"`
	FieldValues      []CreateTaskFieldValue  `json:"field_values,omitempty"`
	Dependencies     []CreateTaskDependency  `json:"dependencies,omitempty"`
	AddToBacklog     bool                    `json:"add_to_backlog,omitempty"`
}

type CreateTaskChecklist struct {
	Name  string                    `json:"name" binding:"required"`
	Items []CreateTaskChecklistItem `json:"items,omitempty"`
}

type CreateTaskChecklistItem struct {
	Content   string `json:"content" binding:"required"`
	IsChecked bool   `json:"is_checked"`
	Order     int32  `json:"order"`
}

type CreateTaskFieldValue struct {
	FieldID       uuid.UUID  `json:"field_id" binding:"required"`
	ValueText     *string    `json:"value_text,omitempty"`
	ValueNumber   *string    `json:"value_number,omitempty"`
	ValueDatetime *time.Time `json:"value_datetime,omitempty"`
}

type CreateTaskDependency struct {
	DependsOnTaskID uuid.UUID `json:"depends_on_task_id" binding:"required"`
	Type            string    `json:"type" binding:"required"`
}

type UpdateTaskRequest struct {
	Name             *string                  `json:"name,omitempty"`
	Description      NullableField[string]    `json:"description"`
	Deadline         NullableField[time.Time] `json:"deadline"`
	ExecutorMemberID NullableField[uuid.UUID] `json:"executor_member_id"`
	ColumnID         *uuid.UUID               `json:"column_id,omitempty"`
	SwimlaneID       NullableField[uuid.UUID] `json:"swimlane_id"`
	Priority         NullableField[string]    `json:"priority"`
	Estimation       NullableField[string]    `json:"estimation"`
}

type DeleteTaskRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type SearchTasksRequest struct {
	ProjectID *uuid.UUID `form:"project_id"`
	ColumnID  *uuid.UUID `form:"column_id"`
}

type TaskWatcherResponse struct {
	TaskID   uuid.UUID `json:"task_id"`
	MemberID uuid.UUID `json:"member_id"`
}

type AddWatcherRequest struct {
	MemberID uuid.UUID `json:"member_id" binding:"required"`
}

// Field values

type TaskFieldValueResponse struct {
	FieldID       uuid.UUID  `json:"field_id"`
	ValueText     *string    `json:"value_text,omitempty"`
	ValueNumber   *string    `json:"value_number,omitempty"`
	ValueDatetime *time.Time `json:"value_datetime,omitempty"`
}

type SetTaskFieldValueRequest struct {
	ValueText     *string    `json:"value_text,omitempty"`
	ValueNumber   *string    `json:"value_number,omitempty"`
	ValueDatetime *time.Time `json:"value_datetime,omitempty"`
}

type TaskDependencyResponse struct {
	ID              uuid.UUID `json:"id"`
	TaskID          uuid.UUID `json:"task_id"`
	DependsOnTaskID uuid.UUID `json:"depends_on_task_id"`
	Type            string    `json:"type"`
}

type AddDependencyRequest struct {
	DependsOnTaskID uuid.UUID `json:"depends_on_task_id" binding:"required"`
	Type            string    `json:"type" binding:"required"`
}

type ChecklistResponse struct {
	ID     uuid.UUID              `json:"id"`
	TaskID uuid.UUID              `json:"task_id"`
	Name   string                 `json:"name"`
	Items  []ChecklistItemResponse `json:"items"`
}

type ChecklistItemResponse struct {
	ID          uuid.UUID `json:"id"`
	ChecklistID uuid.UUID `json:"checklist_id"`
	Content     string    `json:"content"`
	IsChecked   bool      `json:"is_checked"`
	Order       int32     `json:"order"`
}

type CreateChecklistRequest struct {
	Name string `json:"name" binding:"required"`
}

type CreateChecklistItemRequest struct {
	Content string `json:"content" binding:"required"`
	Order   int32  `json:"order"`
}

type SetChecklistItemStatusRequest struct {
	IsChecked bool `json:"is_checked" binding:"required"`
}

type UpdateChecklistRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateChecklistItemRequest struct {
	Content string `json:"content" binding:"required"`
}
