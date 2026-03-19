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
	ID          uuid.UUID     `json:"id"`
	Key         string        `json:"key"`
	Name        string        `json:"name"`
	Description *string       `json:"description,omitempty"`
	Type        ProjectType   `json:"project_type"`
	OwnerID     uuid.UUID     `json:"owner_id"`
	Status      ProjectStatus `json:"status"`
}
