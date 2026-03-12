package domain

import "github.com/google/uuid"

type RoleScope string

const (
	RoleScopeSystem  RoleScope = "system"
	RoleScopeProject RoleScope = "project"
)

type Permission struct {
	ID          uuid.UUID
	Code        string
	Description string
}

type Role struct {
	ID          uuid.UUID
	Name        string
	Description string
	Scope       RoleScope
	ProjectID   *uuid.UUID
	Permissions []Permission
}

