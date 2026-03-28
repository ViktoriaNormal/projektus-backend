package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type ProjectRoleRepository interface {
	List(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectRole, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectRole, error)
	Create(ctx context.Context, params db.CreateProjRoleDefinitionParams) (*domain.ProjectRole, error)
	Update(ctx context.Context, params db.UpdateProjRoleDefinitionParams) (*domain.ProjectRole, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CountMembers(ctx context.Context, roleID uuid.UUID) (int32, error)
	GetProjectAdminRoleID(ctx context.Context, projectID uuid.UUID) (uuid.UUID, error)
	CountMembersWithRole(ctx context.Context, projectID, roleID uuid.UUID) (int32, error)

	ListPermissions(ctx context.Context, roleID uuid.UUID) ([]db.RolePermission, error)
	UpsertPermission(ctx context.Context, roleID uuid.UUID, area, access string) error
	DeletePermissions(ctx context.Context, roleID uuid.UUID) error
}

type projectRoleRepository struct {
	q *db.Queries
}

func NewProjectRoleRepository(q *db.Queries) ProjectRoleRepository {
	return &projectRoleRepository{q: q}
}

func nullUUIDToString(n uuid.NullUUID) string {
	if n.Valid {
		return n.UUID.String()
	}
	return ""
}

func (r *projectRoleRepository) List(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectRole, error) {
	rows, err := r.q.ListProjRoleDefinitions(ctx, uuid.NullUUID{UUID: projectID, Valid: true})
	if err != nil {
		return nil, err
	}
	result := make([]domain.ProjectRole, len(rows))
	for i, row := range rows {
		result[i] = domain.ProjectRole{
			ID: row.ID.String(), ProjectID: nullUUIDToString(row.ProjectID),
			Name: row.Name, Description: row.Description,
			IsAdmin: row.IsAdmin,
		}
	}
	return result, nil
}

func (r *projectRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectRole, error) {
	row, err := r.q.GetProjRoleDefinitionByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	role := domain.ProjectRole{
		ID: row.ID.String(), ProjectID: nullUUIDToString(row.ProjectID),
		Name: row.Name, Description: row.Description,
		IsAdmin: row.IsAdmin,
	}
	return &role, nil
}

func (r *projectRoleRepository) Create(ctx context.Context, params db.CreateProjRoleDefinitionParams) (*domain.ProjectRole, error) {
	row, err := r.q.CreateProjRoleDefinition(ctx, params)
	if err != nil {
		return nil, err
	}
	role := domain.ProjectRole{
		ID: row.ID.String(), ProjectID: nullUUIDToString(row.ProjectID),
		Name: row.Name, Description: row.Description,
		IsAdmin: row.IsAdmin,
	}
	return &role, nil
}

func (r *projectRoleRepository) Update(ctx context.Context, params db.UpdateProjRoleDefinitionParams) (*domain.ProjectRole, error) {
	row, err := r.q.UpdateProjRoleDefinition(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	role := domain.ProjectRole{
		ID: row.ID.String(), ProjectID: nullUUIDToString(row.ProjectID),
		Name: row.Name, Description: row.Description,
		IsAdmin: row.IsAdmin,
	}
	return &role, nil
}

func (r *projectRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_ = r.q.DeleteProjRoleDefPermissionsByRoleID(ctx, id)
	return r.q.DeleteProjRoleDefinitionByID(ctx, id)
}

func (r *projectRoleRepository) CountMembers(ctx context.Context, roleID uuid.UUID) (int32, error) {
	return r.q.CountProjRoleDefinitionMembers(ctx, roleID)
}

func (r *projectRoleRepository) GetProjectAdminRoleID(ctx context.Context, projectID uuid.UUID) (uuid.UUID, error) {
	return r.q.GetProjectAdminRoleID(ctx, uuid.NullUUID{UUID: projectID, Valid: true})
}

func (r *projectRoleRepository) CountMembersWithRole(ctx context.Context, projectID, roleID uuid.UUID) (int32, error) {
	return r.q.CountMembersWithRole(ctx, db.CountMembersWithRoleParams{ProjectID: projectID, RoleID: roleID})
}

func (r *projectRoleRepository) ListPermissions(ctx context.Context, roleID uuid.UUID) ([]db.RolePermission, error) {
	return r.q.ListProjRoleDefPermissions(ctx, roleID)
}

func (r *projectRoleRepository) UpsertPermission(ctx context.Context, roleID uuid.UUID, area, access string) error {
	return r.q.UpsertProjRoleDefPermission(ctx, db.UpsertProjRoleDefPermissionParams{RoleID: roleID, PermissionCode: area, Access: sql.NullString{String: access, Valid: access != ""}})
}

func (r *projectRoleRepository) DeletePermissions(ctx context.Context, roleID uuid.UUID) error {
	return r.q.DeleteProjRoleDefPermissionsByRoleID(ctx, roleID)
}
