package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
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

func (r *projectRoleRepository) List(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectRole, error) {
	rows, err := r.q.ListProjRoleDefinitions(ctx, uuid.NullUUID{UUID: projectID, Valid: true})
	if err != nil {
		return nil, errctx.Wrap(err, "ListProjRoleDefinitions", "projectID", projectID)
	}
	result := make([]domain.ProjectRole, len(rows))
	for i, row := range rows {
		result[i] = domain.ProjectRole{
			ID: row.ID, ProjectID: nullUUIDOrNil(row.ProjectID),
			Name: row.Name, Description: row.Description,
			IsAdmin: row.IsAdmin, Order: row.SortOrder,
		}
	}
	return result, nil
}

func (r *projectRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectRole, error) {
	row, err := r.q.GetProjRoleDefinitionByID(ctx, id)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetProjRoleDefinitionByID", "id", id)
	}
	role := domain.ProjectRole{
		ID: row.ID, ProjectID: nullUUIDOrNil(row.ProjectID),
		Name: row.Name, Description: row.Description,
		IsAdmin: row.IsAdmin,
	}
	return &role, nil
}

func (r *projectRoleRepository) Create(ctx context.Context, params db.CreateProjRoleDefinitionParams) (*domain.ProjectRole, error) {
	row, err := r.q.CreateProjRoleDefinition(ctx, params)
	if err != nil {
		return nil, errctx.Wrap(err, "CreateProjRoleDefinition", "name", params.Name)
	}
	role := domain.ProjectRole{
		ID: row.ID, ProjectID: nullUUIDOrNil(row.ProjectID),
		Name: row.Name, Description: row.Description,
		IsAdmin: row.IsAdmin,
	}
	return &role, nil
}

func (r *projectRoleRepository) Update(ctx context.Context, params db.UpdateProjRoleDefinitionParams) (*domain.ProjectRole, error) {
	row, err := r.q.UpdateProjRoleDefinition(ctx, params)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "UpdateProjRoleDefinition", "id", params.ID)
	}
	role := domain.ProjectRole{
		ID: row.ID, ProjectID: nullUUIDOrNil(row.ProjectID),
		Name: row.Name, Description: row.Description,
		IsAdmin: row.IsAdmin,
	}
	return &role, nil
}

func (r *projectRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_ = r.q.DeleteProjRoleDefPermissionsByRoleID(ctx, id)
	return errctx.Wrap(r.q.DeleteProjRoleDefinitionByID(ctx, id), "DeleteProjRoleDefinitionByID", "id", id)
}

func (r *projectRoleRepository) CountMembers(ctx context.Context, roleID uuid.UUID) (int32, error) {
	n, err := r.q.CountProjRoleDefinitionMembers(ctx, roleID)
	if err != nil {
		return 0, errctx.Wrap(err, "CountProjRoleDefinitionMembers", "roleID", roleID)
	}
	return n, nil
}

func (r *projectRoleRepository) GetProjectAdminRoleID(ctx context.Context, projectID uuid.UUID) (uuid.UUID, error) {
	id, err := r.q.GetProjectAdminRoleID(ctx, uuid.NullUUID{UUID: projectID, Valid: true})
	if err != nil {
		return uuid.Nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetProjectAdminRoleID", "projectID", projectID)
	}
	return id, nil
}

func (r *projectRoleRepository) CountMembersWithRole(ctx context.Context, projectID, roleID uuid.UUID) (int32, error) {
	n, err := r.q.CountMembersWithRole(ctx, db.CountMembersWithRoleParams{ProjectID: projectID, RoleID: roleID})
	if err != nil {
		return 0, errctx.Wrap(err, "CountMembersWithRole", "projectID", projectID, "roleID", roleID)
	}
	return n, nil
}

func (r *projectRoleRepository) ListPermissions(ctx context.Context, roleID uuid.UUID) ([]db.RolePermission, error) {
	rows, err := r.q.ListProjRoleDefPermissions(ctx, roleID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListProjRoleDefPermissions", "roleID", roleID)
	}
	return rows, nil
}

func (r *projectRoleRepository) UpsertPermission(ctx context.Context, roleID uuid.UUID, area, access string) error {
	return errctx.Wrap(r.q.UpsertProjRoleDefPermission(ctx, db.UpsertProjRoleDefPermissionParams{RoleID: roleID, PermissionCode: area, Access: sql.NullString{String: access, Valid: access != ""}}), "UpsertProjRoleDefPermission", "roleID", roleID, "area", area)
}

func (r *projectRoleRepository) DeletePermissions(ctx context.Context, roleID uuid.UUID) error {
	return errctx.Wrap(r.q.DeleteProjRoleDefPermissionsByRoleID(ctx, roleID), "DeleteProjRoleDefPermissionsByRoleID", "roleID", roleID)
}
