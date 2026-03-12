package services

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

const SystemPermissionManageRoles = "system.roles.manage"

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

func (s *RoleService) CreateSystemRole(ctx context.Context, name, description string) (*domain.Role, error) {
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.CreateSystemRole(ctx, name, description)
}

func (s *RoleService) UpdateSystemRole(ctx context.Context, id uuid.UUID, name, description string) (*domain.Role, error) {
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.UpdateSystemRole(ctx, id, name, description)
}

func (s *RoleService) DeleteSystemRole(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteRole(ctx, id)
}

func (s *RoleService) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	return s.repo.ListPermissions(ctx)
}

func (s *RoleService) AssignSystemRolesToUser(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	// Replace existing roles with provided set
	if err := s.repo.DeleteUserRoles(ctx, userID); err != nil {
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

