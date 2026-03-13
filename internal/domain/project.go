package domain

import "github.com/google/uuid"

type ProjectType string

const (
	ProjectTypeScrum  ProjectType = "scrum"
	ProjectTypeKanban ProjectType = "kanban"
)

type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusArchived ProjectStatus = "archived"
	ProjectStatusPaused   ProjectStatus = "paused"
)

type Project struct {
	ID          uuid.UUID
	Key         string
	Name        string
	Description *string
	Type        ProjectType
	OwnerID     uuid.UUID
	Status      ProjectStatus
}

