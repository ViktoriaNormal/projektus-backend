package dto

import "github.com/google/uuid"

type ProjectTemplateResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	ProjectType string    `json:"project_type"`
}

type CreateTemplateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ProjectType string `json:"project_type" binding:"required"`
}

