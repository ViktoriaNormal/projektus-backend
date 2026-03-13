package dto

import "github.com/google/uuid"

type BoardResponse struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   *uuid.UUID `json:"projectId,omitempty"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Order       int16     `json:"order"`
}

type CreateBoardRequest struct {
	ProjectID   uuid.UUID `json:"projectId" binding:"required"`
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description"`
	Order       int16     `json:"order"`
}

type UpdateBoardRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Order       *int16  `json:"order,omitempty"`
}

type ColumnResponse struct {
	ID         uuid.UUID  `json:"id"`
	BoardID    uuid.UUID  `json:"boardId"`
	Name       string     `json:"name"`
	SystemType *string    `json:"systemType,omitempty"`
	WipLimit   *int16     `json:"wipLimit,omitempty"`
	Order      int16      `json:"order"`
}

type CreateColumnRequest struct {
	Name       string  `json:"name" binding:"required"`
	SystemType *string `json:"systemType,omitempty"`
	WipLimit   *int16  `json:"wipLimit,omitempty"`
	Order      int16   `json:"order"`
}

type UpdateColumnRequest struct {
	Name       *string `json:"name,omitempty"`
	SystemType *string `json:"systemType,omitempty"`
	WipLimit   *int16  `json:"wipLimit,omitempty"`
	Order      *int16  `json:"order,omitempty"`
}

type SwimlaneResponse struct {
	ID       uuid.UUID `json:"id"`
	BoardID  uuid.UUID `json:"boardId"`
	Name     string    `json:"name"`
	WipLimit *int16    `json:"wipLimit,omitempty"`
	Order    int16     `json:"order"`
}

type CreateSwimlaneRequest struct {
	Name     string `json:"name" binding:"required"`
	WipLimit *int16 `json:"wipLimit,omitempty"`
	Order    int16  `json:"order"`
}

type UpdateSwimlaneRequest struct {
	Name     *string `json:"name,omitempty"`
	WipLimit *int16  `json:"wipLimit,omitempty"`
	Order    *int16  `json:"order,omitempty"`
}

type NoteResponse struct {
	ID         uuid.UUID  `json:"id"`
	ColumnID   *uuid.UUID `json:"columnId,omitempty"`
	SwimlaneID *uuid.UUID `json:"swimlaneId,omitempty"`
	Content    string     `json:"content"`
}

type CreateNoteRequest struct {
	Content string `json:"content" binding:"required"`
}

type UpdateNoteRequest struct {
	Content *string `json:"content,omitempty"`
}

