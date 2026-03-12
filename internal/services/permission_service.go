package services

import (
	"context"

	"github.com/google/uuid"
)

// PermissionService currently wraps system-level permission checks.
// Project-level permissions will be added later.
type PermissionService struct {
	roleService *RoleService
}

func NewPermissionService(roleService *RoleService) *PermissionService {
	return &PermissionService{roleService: roleService}
}

func (s *PermissionService) HasPermission(ctx context.Context, userID uuid.UUID, permission string, projectID *uuid.UUID) (bool, error) {
	// For stage 1 we only support system-level permissions.
	return s.roleService.UserHasSystemPermission(ctx, userID, permission)
}
