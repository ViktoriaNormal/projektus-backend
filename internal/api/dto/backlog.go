package dto

import "github.com/google/uuid"

type TaskOrder struct {
	TaskID uuid.UUID `json:"task_id" binding:"required"`
	Order  int       `json:"order" binding:"required"`
}
