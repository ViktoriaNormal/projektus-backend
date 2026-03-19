package domain

import (
	"time"

	"github.com/google/uuid"
)

type SprintTask struct {
	ID       uuid.UUID `json:"id"`
	SprintID uuid.UUID `json:"sprint_id"`
	TaskID   uuid.UUID `json:"task_id"`
	Order    int       `json:"order"`
	AddedAt  time.Time `json:"added_at"`
}
