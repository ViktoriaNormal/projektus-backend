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

