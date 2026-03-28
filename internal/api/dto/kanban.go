package dto

import "github.com/google/uuid"

type WipLimitDTO struct {
	BoardID    uuid.UUID  `json:"board_id"`
	ColumnID   *uuid.UUID `json:"column_id,omitempty"`
	SwimlaneID *uuid.UUID `json:"swimlane_id,omitempty"`
	Limit      *int       `json:"limit,omitempty"`
}

type WipCountDTO struct {
	ColumnID   *uuid.UUID `json:"column_id,omitempty"`
	SwimlaneID *uuid.UUID `json:"swimlane_id,omitempty"`
	Count      int        `json:"count"`
	Limit      *int       `json:"limit,omitempty"`
	Exceeded   bool       `json:"exceeded"`
}
