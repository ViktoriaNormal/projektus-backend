package domain

import (
	"time"

	"github.com/google/uuid"
)

type ProjectTemplate struct {
	ID          uuid.UUID
	Name        string
	Description *string
	Type        ProjectType
	CreatedAt   time.Time
}

