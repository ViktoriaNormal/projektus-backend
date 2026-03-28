package domain

import "github.com/google/uuid"

type RoleScope string

const (
	RoleScopeSystem  RoleScope = "system"
	RoleScopeProject RoleScope = "project"
)

type Permission struct {
	Code   string `json:"code"`
	Access string `json:"access,omitempty"`
}

type PermissionDefinition struct {
	Code        string `json:"code"`
	Scope       string `json:"scope"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ColumnSystemTypeDefinition struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type FieldTypeDefinition struct {
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	AvailableFor  []string `json:"available_for"`
	AllowedScopes []string `json:"allowed_scopes"`
}

type Role struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Scope       RoleScope    `json:"scope"`
	IsAdmin     bool         `json:"is_admin"`
	ProjectID   *uuid.UUID   `json:"project_id,omitempty"`
	Permissions []Permission `json:"permissions,omitempty"`
}
