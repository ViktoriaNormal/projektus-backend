package services

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type PermissionService struct {
	roleService *RoleService
	repo        repositories.PermissionRepository
}

func NewPermissionService(roleService *RoleService, repo repositories.PermissionRepository) *PermissionService {
	return &PermissionService{roleService: roleService, repo: repo}
}

func (s *PermissionService) HasPermission(ctx context.Context, userID uuid.UUID, permission string, projectID *uuid.UUID) (bool, error) {
	return s.roleService.UserHasSystemPermission(ctx, userID, permission)
}

// GetMyPermissions returns effective project permissions for the user.
// Logic: system.projects.manage = full → all areas full; view → all view; none → project role.
func (s *PermissionService) GetMyPermissions(ctx context.Context, userID, projectID uuid.UUID) ([]domain.ProjectRolePermission, error) {
	sysAccess, err := s.repo.GetSystemProjectManageAccess(ctx, userID)
	if err != nil {
		return nil, err
	}

	// system.projects.manage = full → полный доступ ко всем areas.
	if sysAccess == "full" {
		areas := repositories.ProjectPermissionAreas
		result := make([]domain.ProjectRolePermission, len(areas))
		for i, a := range areas {
			result[i] = domain.ProjectRolePermission{Area: a.Area, Access: "full"}
		}
		return result, nil
	}

	// system.projects.manage = view → минимум "view" для всех areas,
	// но проектная роль может повысить до "full".
	sysMinAccess := ""
	if sysAccess == "view" {
		sysMinAccess = "view"
	}

	rolePerms, err := s.repo.ListMemberProjectPermissions(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}

	permMap := make(map[string]string)
	for _, p := range rolePerms {
		existing, ok := permMap[p.Area]
		if !ok || accessRank(p.Access) > accessRank(existing) {
			permMap[p.Area] = p.Access
		}
	}

	areas := repositories.ProjectPermissionAreas
	result := make([]domain.ProjectRolePermission, len(areas))
	for i, a := range areas {
		access := "none"
		if sysMinAccess != "" && accessRank(sysMinAccess) > accessRank(access) {
			access = sysMinAccess
		}
		if v, ok := permMap[a.Area]; ok && accessRank(v) > accessRank(access) {
			access = v
		}
		result[i] = domain.ProjectRolePermission{Area: a.Area, Access: access}
	}
	return result, nil
}

// GetEffectiveAreaAccess returns the effective access level for a single project permission area.
func (s *PermissionService) GetEffectiveAreaAccess(ctx context.Context, userID, projectID uuid.UUID, area string) (string, error) {
	sysAccess, err := s.repo.GetSystemProjectManageAccess(ctx, userID)
	if err != nil {
		return "", err
	}
	if sysAccess == "full" {
		return "full", nil
	}

	roleAccess, err := s.repo.GetMemberAreaMaxAccess(ctx, userID, projectID, area)
	if err != nil {
		return "", err
	}

	access := "none"
	if sysAccess == "view" && accessRank("view") > accessRank(access) {
		access = "view"
	}
	if accessRank(roleAccess) > accessRank(access) {
		access = roleAccess
	}
	return access, nil
}

// HasProjectAreaAccess reports whether the user's effective access to area is at least minAccess.
func (s *PermissionService) HasProjectAreaAccess(ctx context.Context, userID, projectID uuid.UUID, area, minAccess string) (bool, error) {
	effective, err := s.GetEffectiveAreaAccess(ctx, userID, projectID, area)
	if err != nil {
		return false, err
	}
	return accessRank(effective) >= accessRank(minAccess), nil
}

// GetProjectManageAccess returns the user's system-level access for system.projects.manage.
// Returns "full", "view", or "none".
func (s *PermissionService) GetProjectManageAccess(ctx context.Context, userID uuid.UUID) string {
	access, err := s.repo.GetSystemProjectManageAccess(ctx, userID)
	if err != nil {
		return "none"
	}
	return access
}

func accessRank(access string) int {
	switch access {
	case "full":
		return 2
	case "view":
		return 1
	default:
		return 0
	}
}
