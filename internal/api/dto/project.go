package dto

import "github.com/google/uuid"

type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ProjectType string `json:"project_type" binding:"required"` // scrum | kanban
}

type UpdateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

type ProjectResponse struct {
	ID          uuid.UUID `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	ProjectType string    `json:"project_type"`
	OwnerID     uuid.UUID `json:"owner_id"`
	Status      string    `json:"status"`
}

