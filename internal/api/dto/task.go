package dto

import (
	"time"

	"github.com/google/uuid"
)

type TaskResponse struct {
	ID          uuid.UUID  `json:"id"`
	Key         string     `json:"key"`
	ProjectID   uuid.UUID  `json:"project_id"`
	OwnerID     uuid.UUID  `json:"owner_id"`
	ExecutorID  *uuid.UUID `json:"executor_id,omitempty"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	ColumnID    uuid.UUID  `json:"column_id"`
	SwimlaneID  *uuid.UUID `json:"swimlane_id,omitempty"`
	Progress    *int       `json:"progress,omitempty"`
}

type CreateTaskRequest struct {
	ProjectID        uuid.UUID  `json:"project_id" binding:"required"`
	OwnerMemberID    uuid.UUID  `json:"owner_member_id" binding:"required"`
	ExecutorMemberID *uuid.UUID `json:"executor_member_id,omitempty"`
	Name             string     `json:"name" binding:"required"`
	Description      string     `json:"description"`
	Deadline         *time.Time `json:"deadline,omitempty"`
	ColumnID         uuid.UUID  `json:"column_id" binding:"required"`
	SwimlaneID       *uuid.UUID `json:"swimlane_id,omitempty"`
}

type UpdateTaskRequest struct {
	Name             *string                  `json:"name,omitempty"`
	Description      NullableField[string]    `json:"description"`
	Deadline         NullableField[time.Time] `json:"deadline"`
	ExecutorMemberID NullableField[uuid.UUID] `json:"executor_member_id"`
	ColumnID         *uuid.UUID               `json:"column_id,omitempty"`
	SwimlaneID       NullableField[uuid.UUID] `json:"swimlane_id"`
}

type DeleteTaskRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type SearchTasksRequest struct {
	ProjectID  *uuid.UUID `form:"project_id"`
	OwnerID    *uuid.UUID `form:"owner_id"`
	ExecutorID *uuid.UUID `form:"executor_id"`
	ColumnID   *uuid.UUID `form:"column_id"`
}

type TaskWatcherResponse struct {
	ID              uuid.UUID `json:"id"`
	TaskID          uuid.UUID `json:"task_id"`
	ProjectMemberID uuid.UUID `json:"project_member_id"`
}

type AddWatcherRequest struct {
	ProjectMemberID uuid.UUID `json:"project_member_id" binding:"required"`
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
