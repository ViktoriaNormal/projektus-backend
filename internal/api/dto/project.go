package dto

import "github.com/google/uuid"

type CreateProjectRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	ProjectType string  `json:"project_type" binding:"required"` // scrum | kanban
	OwnerID     *string `json:"owner_id"`                        // uuid, опционально — по умолчанию текущий пользователь
}

type UpdateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

type ProjectOwnerResponse struct {
	ID       string  `json:"id"`
	FullName string  `json:"full_name"`
	AvatarURL *string `json:"avatar_url"`
	Email    string  `json:"email"`
}

type ProjectResponse struct {
	ID          uuid.UUID             `json:"id"`
	Key         string                `json:"key"`
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	ProjectType string                `json:"project_type"`
	OwnerID     uuid.UUID             `json:"owner_id"`
	Status      string                `json:"status"`
	CreatedAt   string                `json:"created_at"`
	Owner       *ProjectOwnerResponse `json:"owner,omitempty"`
}

