package services

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

const (
	ScrumRoleProductOwnerName = "Product Owner"
	ScrumRoleDevTeamName      = "Development Team Member"
	ScrumRoleScrumMasterName  = "Scrum Master"
)

type ScrumRoleService struct {
	roleRepo repositories.RoleRepository
}

func NewScrumRoleService(roleRepo repositories.RoleRepository) *ScrumRoleService {
	return &ScrumRoleService{roleRepo: roleRepo}
}

// InitializeScrumRoles создаёт стандартные Scrum-роли и привязывает к ним базовый набор прав.
func (s *ScrumRoleService) InitializeScrumRoles(ctx context.Context, projectID uuid.UUID) error {
	roles, err := s.roleRepo.ListProjectRoles(ctx, projectID)
	if err != nil {
		return err
	}

	exists := func(name string) bool {
		for _, r := range roles {
			if r.Name == name {
				return true
			}
		}
		return false
	}

	// Наборы прав по умолчанию (коды можно донастроить через админку).
	productOwnerPerms := []string{
		"projects.manage",
		"sprints.manage",
		"boards.manage",
		"tasks.manage",
		"analytics.view",
	}
	scrumMasterPerms := []string{
		"sprints.manage",
		"boards.manage",
		"analytics.view",
		"backlog.view",
		"tasks.view",
		"project.view",
	}
	devTeamPerms := []string{
		"tasks.manage",
		"backlog.view",
		"tasks.view",
		"project.view",
	}

	if !exists(ScrumRoleProductOwnerName) {
		if err := s.createProjectRoleWithPermissions(ctx, projectID, ScrumRoleProductOwnerName, "Scrum Product Owner", productOwnerPerms); err != nil {
			return err
		}
	}
	if !exists(ScrumRoleScrumMasterName) {
		if err := s.createProjectRoleWithPermissions(ctx, projectID, ScrumRoleScrumMasterName, "Scrum Master", scrumMasterPerms); err != nil {
			return err
		}
	}
	if !exists(ScrumRoleDevTeamName) {
		if err := s.createProjectRoleWithPermissions(ctx, projectID, ScrumRoleDevTeamName, "Scrum Development Team Member", devTeamPerms); err != nil {
			return err
		}
	}

	return nil
}

func (s *ScrumRoleService) createProjectRoleWithPermissions(ctx context.Context, projectID uuid.UUID, name, description string, permCodes []string) error {
	// создаём проектную роль через общий репозиторий ролей
	role, err := s.roleRepo.CreateProjectRole(ctx, projectID, name, description)
	if err != nil {
		return err
	}

	for _, code := range permCodes {
		perm, err := s.ensurePermission(ctx, code)
		if err != nil {
			return err
		}
		if err := s.roleRepo.AddPermissionToRole(ctx, role.ID, perm.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *ScrumRoleService) ensurePermission(ctx context.Context, code string) (*domain.Permission, error) {
	p, err := s.roleRepo.GetPermissionByCode(ctx, code)
	if err == nil {
		return p, nil
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	return s.roleRepo.CreatePermission(ctx, code, "")
}
