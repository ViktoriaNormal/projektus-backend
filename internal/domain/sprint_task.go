package domain

import (
	"time"

	"github.com/google/uuid"
)

type SprintTask struct {
	ID       uuid.UUID
	SprintID uuid.UUID
	TaskID   uuid.UUID
	Order    int
	AddedAt  time.Time
}

