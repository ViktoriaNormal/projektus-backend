package domain

import (
	"time"

	"github.com/google/uuid"
)

type ProductBacklogItem struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	TaskID    uuid.UUID `json:"task_id"`
	Order     int       `json:"order"`
	AddedAt   time.Time `json:"added_at"`
}
