package dto

import "github.com/google/uuid"

type RolePermissionResponse struct {
	Code   string `json:"code"`
	Access string `json:"access"`
}

type RoleResponse struct {
	ID          uuid.UUID                `json:"id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	IsAdmin     bool                     `json:"is_admin"`
	Permissions []RolePermissionResponse  `json:"permissions"`
}

type RolePermissionRequest struct {
	Code   string `json:"code" binding:"required"`
	Access string `json:"access" binding:"required,oneof=full view none"`
}

type CreateRoleRequest struct {
	Name        string                  `json:"name" binding:"required"`
	Description string                  `json:"description"`
	Permissions []RolePermissionRequest  `json:"permissions"`
}

type UpdateRoleRequest struct {
	Name        string                  `json:"name" binding:"required"`
	Description string                  `json:"description"`
	Permissions []RolePermissionRequest  `json:"permissions"`
}

type PermissionResponse struct {
	Code        string `json:"code"`
	Scope       string `json:"scope"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ColumnSystemTypeResponse struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type FieldTypeResponse struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	AvailableFor []string `json:"available_for"`
}

type ReferenceDataResponse struct {
	Permissions       []PermissionResponse       `json:"permissions"`
	ColumnSystemTypes []ColumnSystemTypeResponse  `json:"column_system_types"`
	FieldTypes        []FieldTypeResponse         `json:"field_types"`
}

type AssignRolesRequest struct {
	RoleIDs []uuid.UUID `json:"role_ids" binding:"required"`
}

type ProjectRoleResponse struct {
	ProjectID   uuid.UUID `json:"project_id"`
	ProjectName string    `json:"project_name"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
}
