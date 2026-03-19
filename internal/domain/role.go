package domain

import "github.com/google/uuid"

type RoleScope string

const (
	RoleScopeSystem  RoleScope = "system"
	RoleScopeProject RoleScope = "project"
)

type Permission struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Description string    `json:"description,omitempty"`
}

type Role struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Scope       RoleScope    `json:"scope"`
	ProjectID   *uuid.UUID   `json:"project_id,omitempty"`
	Permissions []Permission `json:"permissions,omitempty"`
}
