package domain

import (
	"time"

	"github.com/google/uuid"
)

type ProductBacklogItem struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	TaskID    uuid.UUID
	Order     int
	AddedAt   time.Time
}

