package domain

import "github.com/google/uuid"

type ProjectMember struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	UserID    uuid.UUID
	Roles     []string
}

