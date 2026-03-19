package domain

import (
	"time"

	"github.com/google/uuid"
)

type ProjectTemplate struct {
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`
	Description *string     `json:"description,omitempty"`
	Type        ProjectType `json:"project_type"`
	CreatedAt   time.Time   `json:"-"`
}
