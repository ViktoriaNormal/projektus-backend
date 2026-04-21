package dto

import "github.com/google/uuid"

type ProjectRoleDefinitionResponse struct {
	ID          uuid.UUID                          `json:"id"`
	Name        string                             `json:"name"`
	Description string                             `json:"description"`
	IsAdmin     bool                               `json:"is_admin"`
	// Order — стабильная позиция роли в списке (ASC). Админские роли
	// изначально сверху (sort_order=1), остальные идут следом.
	Order       int32                              `json:"order"`
	Permissions []ProjectRoleDefPermissionResponse `json:"permissions"`
}

type ProjectRoleDefPermissionResponse struct {
	Area   string `json:"area"`
	Access string `json:"access"`
}

type CreateProjectRoleRequest struct {
	Name        string                `json:"name" binding:"required"`
	Description string                `json:"description"`
	Permissions []RolePermissionInput `json:"permissions" binding:"required"`
}

type UpdateProjectRoleRequest struct {
	Name        *string               `json:"name,omitempty"`
	Description *string               `json:"description,omitempty"`
	Permissions []RolePermissionInput  `json:"permissions,omitempty"`
}
