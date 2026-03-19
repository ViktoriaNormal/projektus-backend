package services

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

const (
	SystemPermissionManageRoles          = "system.roles.manage"
	SystemPermissionManageUsers          = "system.users.manage"
	SystemPermissionManagePasswordPolicy = "system.password_policy.manage"
	SystemPermissionManageTemplates      = "system.project_templates.manage"
	SystemPermissionViewAllProjects      = "system.projects.view_all"
	SystemPermissionCreateProjects       = "system.projects.create"
	SystemPermissionDeleteProjects       = "system.projects.delete"
	SystemPermissionArchiveProjects      = "system.projects.archive"
	SystemPermissionEditAllProjects      = "system.projects.edit_all"
)

type RoleService struct {
	repo repositories.RoleRepository
}

func NewRoleService(repo repositories.RoleRepository) *RoleService {
	return &RoleService{repo: repo}
}

func (s *RoleService) ListSystemRoles(ctx context.Context) ([]domain.Role, error) {
	return s.repo.ListSystemRoles(ctx)
}

func (s *RoleService) GetSystemRole(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	return s.repo.GetRoleByID(ctx, id)
}

func (s *RoleService) CreateSystemRole(ctx context.Context, name, description string, permissions []string) (*domain.Role, error) {
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	role, err := s.repo.CreateSystemRole(ctx, name, description)
	if err != nil {
		return nil, err
	}
	if err := s.setRolePermissions(ctx, role.ID, permissions); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *RoleService) UpdateSystemRole(ctx context.Context, id uuid.UUID, name, description string, permissions []string) (*domain.Role, error) {
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	role, err := s.repo.UpdateSystemRole(ctx, id, name, description)
	if err != nil {
		return nil, err
	}
	if permissions != nil {
		if err := s.setRolePermissions(ctx, role.ID, permissions); err != nil {
			return nil, err
		}
	}
	return role, nil
}

func (s *RoleService) setRolePermissions(ctx context.Context, roleID uuid.UUID, permissions []string) error {
	if err := s.repo.RemoveAllPermissionsFromRole(ctx, roleID); err != nil {
		return err
	}
	for _, code := range permissions {
		perm, err := s.repo.GetPermissionByCode(ctx, code)
		if err != nil {
			return err
		}
		if err := s.repo.AddPermissionToRole(ctx, roleID, perm.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *RoleService) DeleteSystemRole(ctx context.Context, id uuid.UUID) error {
	role, err := s.repo.GetRoleByID(ctx, id)
	if err != nil {
		return err
	}
	if role.Name == "Админ" {
		return domain.ErrForbidden
	}
	return s.repo.DeleteRole(ctx, id)
}

func (s *RoleService) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	return s.repo.ListPermissions(ctx)
}

func (s *RoleService) AssignSystemRolesToUser(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	// Replace only system roles, leave project roles intact
	if err := s.repo.DeleteUserSystemRoles(ctx, userID); err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		if err := s.repo.AssignRoleToUser(ctx, roleID, userID); err != nil {
			return err
		}
	}
	return nil
}

func (s *RoleService) GetUserSystemRoles(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	return s.repo.ListUserSystemRoles(ctx, userID)
}

func (s *RoleService) UserHasSystemPermission(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	return s.repo.UserHasSystemPermission(ctx, userID, code)
}

func (s *RoleService) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	perms, err := s.repo.ListRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}
	codes := make([]string, len(perms))
	for i, p := range perms {
		codes[i] = p.Code
	}
	return codes, nil
}

