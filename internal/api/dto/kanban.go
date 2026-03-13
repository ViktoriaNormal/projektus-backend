package dto

import "github.com/google/uuid"

type WipLimitDTO struct {
	BoardID    uuid.UUID  `json:"boardId"`
	ColumnID   *uuid.UUID `json:"columnId,omitempty"`
	SwimlaneID *uuid.UUID `json:"swimlaneId,omitempty"`
	Limit      *int       `json:"limit,omitempty"`
}

type WipCountDTO struct {
	ColumnID   *uuid.UUID `json:"columnId,omitempty"`
	SwimlaneID *uuid.UUID `json:"swimlaneId,omitempty"`
	Count      int        `json:"count"`
	Limit      *int       `json:"limit,omitempty"`
	Exceeded   bool       `json:"exceeded"`
}

