package services

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type PermissionService struct {
	roleService *RoleService
	queries     *db.Queries
}

func NewPermissionService(roleService *RoleService, queries *db.Queries) *PermissionService {
	return &PermissionService{roleService: roleService, queries: queries}
}

func (s *PermissionService) HasPermission(ctx context.Context, userID uuid.UUID, permission string, projectID *uuid.UUID) (bool, error) {
	return s.roleService.UserHasSystemPermission(ctx, userID, permission)
}

// GetMyPermissions returns effective project permissions for the user.
// Logic: system.projects.manage = full → all areas full; view → all view; none → project role.
func (s *PermissionService) GetMyPermissions(ctx context.Context, userID, projectID uuid.UUID) ([]domain.ProjectRolePermission, error) {
	// Check system-level access for projects.
	sysAccess, err := s.queries.GetSystemPermissionAccess(ctx, db.GetSystemPermissionAccessParams{
		UserID:         userID,
		PermissionCode: repositories.SystemPermissionManageProjects,
	})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// system.projects.manage = full → полный доступ ко всем areas.
	if err == nil && sysAccess.String == "full" {
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
	if err == nil && sysAccess.String == "view" {
		sysMinAccess = "view"
	}

	// Get permissions from the user's project role.
	rows, err := s.queries.GetMemberProjectPermissions(ctx, db.GetMemberProjectPermissionsParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err != nil {
		return nil, err
	}

	// Build result: merge permissions (highest access wins if multiple roles).
	permMap := make(map[string]string)
	for _, r := range rows {
		existing, ok := permMap[r.PermissionCode]
		if !ok || accessRank(r.Access.String) > accessRank(existing) {
			permMap[r.PermissionCode] = r.Access.String
		}
	}

	// Return all areas. Apply system minimum if set, then project role override.
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

// GetProjectManageAccess returns the user's system-level access for system.projects.manage.
// Returns "full", "view", or "none".
func (s *PermissionService) GetProjectManageAccess(ctx context.Context, userID uuid.UUID) string {
	access, err := s.queries.GetSystemPermissionAccess(ctx, db.GetSystemPermissionAccessParams{
		UserID:         userID,
		PermissionCode: repositories.SystemPermissionManageProjects,
	})
	if err != nil || !access.Valid {
		return "none"
	}
	return access.String
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
