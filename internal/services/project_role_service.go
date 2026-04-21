package services

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/catalog"
	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

// validateProjectRolePermissions — то же, что и для template-ролей: проверяет,
// что каждый Area — валидный project-scope код из catalog.AllPermissions.
// Защищает от опечаток типа «project.project.boards».
func validateProjectRolePermissions(permissions []domain.ProjectRolePermission) error {
	if len(permissions) == 0 {
		return nil
	}
	var bad []string
	for _, p := range permissions {
		if !catalog.IsValidProjectPermission(p.Area) {
			bad = append(bad, p.Area)
		}
	}
	if len(bad) > 0 {
		return &domain.InvalidPermissionCodeError{Codes: bad}
	}
	return nil
}

type ProjectRoleService struct {
	repo repositories.ProjectRoleRepository
}

func NewProjectRoleService(repo repositories.ProjectRoleRepository) *ProjectRoleService {
	return &ProjectRoleService{repo: repo}
}

func (s *ProjectRoleService) ListRoles(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectRole, error) {
	roles, err := s.repo.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for i := range roles {
		perms, err := s.repo.ListPermissions(ctx, roles[i].ID)
		if err != nil {
			return nil, err
		}
		roles[i].Permissions = make([]domain.ProjectRolePermission, len(perms))
		for j, p := range perms {
			roles[i].Permissions[j] = domain.ProjectRolePermission{Area: p.PermissionCode, Access: p.Access.String}
		}
	}
	return roles, nil
}

func (s *ProjectRoleService) CreateRole(ctx context.Context, projectID uuid.UUID, name, description string, permissions []domain.ProjectRolePermission) (*domain.ProjectRole, error) {
	if err := validateProjectRolePermissions(permissions); err != nil {
		return nil, err
	}
	role, err := s.repo.Create(ctx, db.CreateProjRoleDefinitionParams{
		ProjectID:   uuid.NullUUID{UUID: projectID, Valid: true},
		Name:        name,
		Description: description,
	})
	if err != nil {
		return nil, err
	}

	roleID := role.ID
	for _, p := range permissions {
		if err := s.repo.UpsertPermission(ctx, roleID, p.Area, p.Access); err != nil {
			return nil, err
		}
	}
	role.Permissions = permissions
	return role, nil
}

func (s *ProjectRoleService) UpdateRole(ctx context.Context, projectID uuid.UUID, roleID uuid.UUID, name, description *string, permissions []domain.ProjectRolePermission) (*domain.ProjectRole, error) {
	existing, err := s.repo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if existing.ProjectID != projectID {
		return nil, domain.ErrNotFound
	}
	// Администратор проекта: можно менять name и description, но нельзя менять permissions
	if existing.IsAdmin && permissions != nil {
		return nil, domain.ErrProjectAdminRole
	}
	if err := validateProjectRolePermissions(permissions); err != nil {
		return nil, err
	}

	finalName := existing.Name
	if name != nil {
		finalName = *name
	}
	finalDesc := existing.Description
	if description != nil {
		finalDesc = *description
	}

	role, err := s.repo.Update(ctx, db.UpdateProjRoleDefinitionParams{
		ID:          roleID,
		Name:        finalName,
		Description: finalDesc,
	})
	if err != nil {
		return nil, err
	}

	if permissions != nil && !existing.IsAdmin {
		for _, p := range permissions {
			if err := s.repo.UpsertPermission(ctx, roleID, p.Area, p.Access); err != nil {
				return nil, err
			}
		}
		role.Permissions = permissions
	} else {
		perms, _ := s.repo.ListPermissions(ctx, roleID)
		role.Permissions = make([]domain.ProjectRolePermission, len(perms))
		for i, p := range perms {
			role.Permissions[i] = domain.ProjectRolePermission{Area: p.PermissionCode, Access: p.Access.String}
		}
	}

	return role, nil
}

func (s *ProjectRoleService) DeleteRole(ctx context.Context, projectID uuid.UUID, roleID uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if existing.ProjectID != projectID {
		return domain.ErrNotFound
	}

	if existing.IsAdmin {
		return domain.ErrProjectAdminRole
	}

	memberCount, _ := s.repo.CountMembers(ctx, roleID)
	if memberCount > 0 {
		return domain.ErrRoleHasMembers
	}

	return s.repo.Delete(ctx, roleID)
}

