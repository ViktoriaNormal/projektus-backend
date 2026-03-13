package dto

import (
	"time"

	"github.com/google/uuid"
)

type TaskResponse struct {
	ID         uuid.UUID  `json:"id"`
	Key        string     `json:"key"`
	ProjectID  uuid.UUID  `json:"projectId"`
	OwnerID    uuid.UUID  `json:"ownerId"`
	ExecutorID *uuid.UUID `json:"executorId,omitempty"`
	Name       string     `json:"name"`
	Description *string   `json:"description,omitempty"`
	Deadline   *time.Time `json:"deadline,omitempty"`
	ColumnID   uuid.UUID  `json:"columnId"`
	SwimlaneID *uuid.UUID `json:"swimlaneId,omitempty"`
	Progress   *int       `json:"progress,omitempty"`
}

type CreateTaskRequest struct {
	ProjectID      uuid.UUID  `json:"projectId" binding:"required"`
	OwnerMemberID  uuid.UUID  `json:"ownerMemberId" binding:"required"`
	ExecutorMemberID *uuid.UUID `json:"executorMemberId,omitempty"`
	Name           string     `json:"name" binding:"required"`
	Description    string     `json:"description"`
	Deadline       *time.Time `json:"deadline,omitempty"`
	ColumnID       uuid.UUID  `json:"columnId" binding:"required"`
	SwimlaneID     *uuid.UUID `json:"swimlaneId,omitempty"`
}

type UpdateTaskRequest struct {
	Name        *string     `json:"name,omitempty"`
	Description *string     `json:"description,omitempty"`
	Deadline    *time.Time  `json:"deadline,omitempty"`
	ExecutorMemberID *uuid.UUID `json:"executorMemberId,omitempty"`
	ColumnID    *uuid.UUID  `json:"columnId,omitempty"`
	SwimlaneID  *uuid.UUID  `json:"swimlaneId,omitempty"`
}

type DeleteTaskRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type SearchTasksRequest struct {
	ProjectID  *uuid.UUID `form:"projectId"`
	OwnerID    *uuid.UUID `form:"ownerId"`
	ExecutorID *uuid.UUID `form:"executorId"`
	ColumnID   *uuid.UUID `form:"columnId"`
}

type TaskWatcherResponse struct {
	ID              uuid.UUID `json:"id"`
	TaskID          uuid.UUID `json:"taskId"`
	ProjectMemberID uuid.UUID `json:"projectMemberId"`
}

type AddWatcherRequest struct {
	ProjectMemberID uuid.UUID `json:"projectMemberId" binding:"required"`
}

type TaskDependencyResponse struct {
	ID              uuid.UUID `json:"id"`
	TaskID          uuid.UUID `json:"taskId"`
	DependsOnTaskID uuid.UUID `json:"dependsOnTaskId"`
	Type            string    `json:"type"`
}

type AddDependencyRequest struct {
	DependsOnTaskID uuid.UUID `json:"dependsOnTaskId" binding:"required"`
	Type            string    `json:"type" binding:"required"`
}

type ChecklistResponse struct {
	ID     uuid.UUID            `json:"id"`
	TaskID uuid.UUID            `json:"taskId"`
	Name   string               `json:"name"`
	Items  []ChecklistItemResponse `json:"items"`
}

type ChecklistItemResponse struct {
	ID          uuid.UUID `json:"id"`
	ChecklistID uuid.UUID `json:"checklistId"`
	Content     string    `json:"content"`
	IsChecked   bool      `json:"isChecked"`
	Order       int16     `json:"order"`
}

type CreateChecklistRequest struct {
	Name string `json:"name" binding:"required"`
}

type CreateChecklistItemRequest struct {
	Content string `json:"content" binding:"required"`
	Order   int16  `json:"order"`
}

type SetChecklistItemStatusRequest struct {
	IsChecked bool `json:"isChecked" binding:"required"`
}

