package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type PermissionRepository interface {
	GetSystemProjectManageAccess(ctx context.Context, userID uuid.UUID) (string, error)
	ListMemberProjectPermissions(ctx context.Context, userID, projectID uuid.UUID) ([]domain.ProjectRolePermission, error)
	GetMemberAreaMaxAccess(ctx context.Context, userID, projectID uuid.UUID, area string) (string, error)
}

type permissionRepository struct {
	q *db.Queries
}

func NewPermissionRepository(q *db.Queries) PermissionRepository {
	return &permissionRepository{q: q}
}

func (r *permissionRepository) GetSystemProjectManageAccess(ctx context.Context, userID uuid.UUID) (string, error) {
	access, err := r.q.GetSystemPermissionAccess(ctx, db.GetSystemPermissionAccessParams{
		UserID:         userID,
		PermissionCode: SystemPermissionManageProjects,
	})
	if err != nil {
		return "", errctx.Wrap(err, "GetSystemPermissionAccess", "userID", userID)
	}
	if !access.Valid {
		return "none", nil
	}
	return access.String, nil
}

func (r *permissionRepository) ListMemberProjectPermissions(ctx context.Context, userID, projectID uuid.UUID) ([]domain.ProjectRolePermission, error) {
	rows, err := r.q.GetMemberProjectPermissions(ctx, db.GetMemberProjectPermissionsParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "GetMemberProjectPermissions", "projectID", projectID, "userID", userID)
	}
	result := make([]domain.ProjectRolePermission, 0, len(rows))
	for _, row := range rows {
		if !row.Access.Valid {
			continue
		}
		result = append(result, domain.ProjectRolePermission{
			Area:   row.PermissionCode,
			Access: row.Access.String,
		})
	}
	return result, nil
}

func (r *permissionRepository) GetMemberAreaMaxAccess(ctx context.Context, userID, projectID uuid.UUID, area string) (string, error) {
	access, err := r.q.GetMemberAreaMaxAccess(ctx, db.GetMemberAreaMaxAccessParams{
		ProjectID:      projectID,
		UserID:         userID,
		PermissionCode: area,
	})
	if err == sql.ErrNoRows {
		return "none", nil
	}
	if err != nil {
		return "", errctx.Wrap(err, "GetMemberAreaMaxAccess", "projectID", projectID, "userID", userID, "area", area)
	}
	if !access.Valid {
		return "none", nil
	}
	return access.String, nil
}
