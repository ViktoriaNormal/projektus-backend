package domain

import (
	"time"

	"github.com/google/uuid"
)

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

type ProjectOwner struct {
	ID        string
	FullName  string
	AvatarURL *string
	Email     string
}

type Project struct {
	ID                    uuid.UUID     `json:"id"`
	Key                   string        `json:"key"`
	Name                  string        `json:"name"`
	Description           *string       `json:"description,omitempty"`
	Type                  ProjectType   `json:"project_type"`
	OwnerID               uuid.UUID     `json:"owner_id"`
	Status                ProjectStatus `json:"status"`
	SprintDurationWeeks   *int          `json:"sprint_duration_weeks,omitempty"`
	IncompleteTasksAction string        `json:"incomplete_tasks_action"`
	CreatedAt             time.Time     `json:"-"`
	Owner                 *ProjectOwner `json:"-"`
}
