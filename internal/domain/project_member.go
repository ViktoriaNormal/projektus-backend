package domain

import "github.com/google/uuid"

type ProjectMember struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	UserID    uuid.UUID `json:"user_id"`
	Roles     []string  `json:"roles,omitempty"`
}
