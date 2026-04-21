package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type RoleRepository interface {
	ListSystemRoles(ctx context.Context) ([]domain.Role, error)
	GetRoleByID(ctx context.Context, id uuid.UUID) (*domain.Role, error)
	CreateSystemRole(ctx context.Context, name, description string) (*domain.Role, error)
	CreateAdminSystemRole(ctx context.Context, name, description string) (*domain.Role, error)
	GetSystemAdminRole(ctx context.Context) (*domain.Role, error)
	CountActiveSystemAdmins(ctx context.Context) (int32, error)
	UpdateSystemRole(ctx context.Context, id uuid.UUID, name, description string) (*domain.Role, error)
	DeleteRole(ctx context.Context, id uuid.UUID) error

	ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error)
	AddPermissionToRole(ctx context.Context, roleID uuid.UUID, permissionCode, access string) error
	RemovePermissionFromRole(ctx context.Context, roleID uuid.UUID, permissionCode string) error
	RemoveAllPermissionsFromRole(ctx context.Context, roleID uuid.UUID) error

	ListUserSystemRoles(ctx context.Context, userID uuid.UUID) ([]domain.Role, error)
	AssignRoleToUser(ctx context.Context, roleID, userID uuid.UUID) error
	RemoveRoleFromUser(ctx context.Context, roleID, userID uuid.UUID) error
	DeleteUserRoles(ctx context.Context, userID uuid.UUID) error
	DeleteUserSystemRoles(ctx context.Context, userID uuid.UUID) error

	CountUsersWithRole(ctx context.Context, roleID uuid.UUID) (int32, error)
	UserHasSystemPermission(ctx context.Context, userID uuid.UUID, code string) (bool, error)

	ListProjectRoles(ctx context.Context, projectID uuid.UUID) ([]domain.Role, error)
	CreateProjectRole(ctx context.Context, projectID uuid.UUID, name, description string) (*domain.Role, error)
}

type roleRepository struct {
	q *db.Queries
}

func NewRoleRepository(q *db.Queries) RoleRepository {
	return &roleRepository{q: q}
}

func (r *roleRepository) ListSystemRoles(ctx context.Context) ([]domain.Role, error) {
	rows, err := r.q.ListSystemRoles(ctx)
	if err != nil {
		return nil, errctx.Wrap(err, "ListSystemRoles")
	}

	roles := make([]domain.Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, domain.Role{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description,
			Scope:       domain.RoleScope(row.Scope),
			IsAdmin:     row.IsAdmin,
		})
	}
	return roles, nil
}

func (r *roleRepository) GetRoleByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	row, err := r.q.GetRoleByID(ctx, id)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetRoleByID", "id", id)
	}

	return &domain.Role{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Scope:       domain.RoleScope(row.Scope),
		IsAdmin:     row.IsAdmin,
	}, nil
}

func (r *roleRepository) CreateSystemRole(ctx context.Context, name, description string) (*domain.Role, error) {
	row, err := r.q.CreateSystemRole(ctx, db.CreateSystemRoleParams{
		Name:        name,
		Description: description,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateSystemRole", "name", name)
	}

	return &domain.Role{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Scope:       domain.RoleScope(row.Scope),
		IsAdmin:     row.IsAdmin,
	}, nil
}

func (r *roleRepository) CreateAdminSystemRole(ctx context.Context, name, description string) (*domain.Role, error) {
	row, err := r.q.CreateAdminSystemRole(ctx, db.CreateAdminSystemRoleParams{
		Name:        name,
		Description: description,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateAdminSystemRole", "name", name)
	}

	return &domain.Role{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Scope:       domain.RoleScope(row.Scope),
		IsAdmin:     row.IsAdmin,
	}, nil
}

func (r *roleRepository) GetSystemAdminRole(ctx context.Context) (*domain.Role, error) {
	row, err := r.q.GetSystemAdminRole(ctx)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetSystemAdminRole")
	}

	return &domain.Role{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Scope:       domain.RoleScope(row.Scope),
		IsAdmin:     row.IsAdmin,
	}, nil
}

func (r *roleRepository) CountActiveSystemAdmins(ctx context.Context) (int32, error) {
	return r.q.CountActiveSystemAdmins(ctx)
}

func (r *roleRepository) UpdateSystemRole(ctx context.Context, id uuid.UUID, name, description string) (*domain.Role, error) {
	row, err := r.q.UpdateSystemRole(ctx, db.UpdateSystemRoleParams{
		ID:          id,
		Name:        name,
		Description: description,
	})
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "UpdateSystemRole", "id", id)
	}

	return &domain.Role{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Scope:       domain.RoleScope(row.Scope),
		IsAdmin:     row.IsAdmin,
	}, nil
}

func (r *roleRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	err := r.q.DeleteRole(ctx, id)
	return errctx.Wrap(err, "DeleteRole", "id", id)
}

func (r *roleRepository) ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error) {
	rows, err := r.q.ListRolePermissions(ctx, roleID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListRolePermissions", "roleID", roleID)
	}

	perms := make([]domain.Permission, 0, len(rows))
	for _, row := range rows {
		access := ""
		if row.Access.Valid {
			access = row.Access.String
		}
		perms = append(perms, domain.Permission{
			Code:   row.PermissionCode,
			Access: access,
		})
	}
	return perms, nil
}

func (r *roleRepository) AddPermissionToRole(ctx context.Context, roleID uuid.UUID, permissionCode, access string) error {
	return r.q.AddPermissionToRole(ctx, db.AddPermissionToRoleParams{
		RoleID:         roleID,
		PermissionCode: permissionCode,
		Access:         sql.NullString{String: access, Valid: access != ""},
	})
}

func (r *roleRepository) RemovePermissionFromRole(ctx context.Context, roleID uuid.UUID, permissionCode string) error {
	return r.q.RemovePermissionFromRole(ctx, db.RemovePermissionFromRoleParams{
		RoleID:         roleID,
		PermissionCode: permissionCode,
	})
}

func (r *roleRepository) RemoveAllPermissionsFromRole(ctx context.Context, roleID uuid.UUID) error {
	return r.q.RemoveAllPermissionsFromRole(ctx, roleID)
}

func (r *roleRepository) ListUserSystemRoles(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	rows, err := r.q.ListUserSystemRoles(ctx, userID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListUserSystemRoles", "userID", userID)
	}

	roles := make([]domain.Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, domain.Role{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description,
			Scope:       domain.RoleScope(row.Scope),
			IsAdmin:     row.IsAdmin,
		})
	}
	return roles, nil
}

func (r *roleRepository) AssignRoleToUser(ctx context.Context, roleID, userID uuid.UUID) error {
	return r.q.AssignRoleToUser(ctx, db.AssignRoleToUserParams{
		RoleID: roleID,
		UserID: userID,
	})
}

func (r *roleRepository) RemoveRoleFromUser(ctx context.Context, roleID, userID uuid.UUID) error {
	return r.q.RemoveRoleFromUser(ctx, db.RemoveRoleFromUserParams{
		RoleID: roleID,
		UserID: userID,
	})
}

func (r *roleRepository) DeleteUserRoles(ctx context.Context, userID uuid.UUID) error {
	return r.q.DeleteUserRoles(ctx, userID)
}

func (r *roleRepository) DeleteUserSystemRoles(ctx context.Context, userID uuid.UUID) error {
	return r.q.DeleteUserSystemRoles(ctx, userID)
}

func (r *roleRepository) CountUsersWithRole(ctx context.Context, roleID uuid.UUID) (int32, error) {
	return r.q.CountUsersWithRole(ctx, roleID)
}

func (r *roleRepository) UserHasSystemPermission(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	return r.q.UserHasSystemPermission(ctx, db.UserHasSystemPermissionParams{
		UserID:         userID,
		PermissionCode: code,
	})
}

func (r *roleRepository) ListProjectRoles(ctx context.Context, projectID uuid.UUID) ([]domain.Role, error) {
	pid := uuid.NullUUID{UUID: projectID, Valid: true}
	rows, err := r.q.ListProjectRoles(ctx, pid)
	if err != nil {
		return nil, errctx.Wrap(err, "ListProjectRoles", "projectID", projectID)
	}

	roles := make([]domain.Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, domain.Role{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description,
			Scope:       domain.RoleScope(row.Scope),
			ProjectID:   nullUUIDToPtr(row.ProjectID),
		})
	}
	return roles, nil
}

func (r *roleRepository) CreateProjectRole(ctx context.Context, projectID uuid.UUID, name, description string) (*domain.Role, error) {
	row, err := r.q.CreateProjectRole(ctx, db.CreateProjectRoleParams{
		Name:        name,
		Description: description,
		ProjectID:   uuid.NullUUID{UUID: projectID, Valid: true},
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateProjectRole", "projectID", projectID, "name", name)
	}
	return &domain.Role{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Scope:       domain.RoleScope(row.Scope),
		ProjectID:   nullUUIDToPtr(row.ProjectID),
	}, nil
}

