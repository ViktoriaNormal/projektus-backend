package dto

import "github.com/google/uuid"

type ProjectMemberResponse struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	UserID    uuid.UUID `json:"user_id"`
	Roles     []string  `json:"roles,omitempty"`
}

type AddMemberRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
	Roles  []string  `json:"roles"`
}

type UpdateMemberRolesRequest struct {
	Roles []string `json:"roles" binding:"required"`
}
