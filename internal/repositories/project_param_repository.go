package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type ProjectParamRepository interface {
	List(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectParam, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectParam, error)
	Create(ctx context.Context, params db.CreateProjectParamParams) (*domain.ProjectParam, error)
	Update(ctx context.Context, params db.UpdateProjectParamParams) (*domain.ProjectParam, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type projectParamRepository struct {
	q *db.Queries
}

func NewProjectParamRepository(q *db.Queries) ProjectParamRepository {
	return &projectParamRepository{q: q}
}

func (r *projectParamRepository) List(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectParam, error) {
	rows, err := r.q.ListProjectParams(ctx, uuid.NullUUID{UUID: projectID, Valid: true})
	if err != nil {
		return nil, errctx.Wrap(err, "ListProjectParams", "projectID", projectID)
	}
	result := make([]domain.ProjectParam, len(rows))
	for i, row := range rows {
		result[i] = mapProjectParamRow(row.ID, row.ProjectID, row.Name, row.FieldType, row.IsRequired, row.Options, row.Value)
	}
	return result, nil
}

func (r *projectParamRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectParam, error) {
	row, err := r.q.GetProjectParamByID(ctx, id)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetProjectParamByID", "id", id)
	}
	p := mapProjectParamRow(row.ID, row.ProjectID, row.Name, row.FieldType, row.IsRequired, row.Options, row.Value)
	return &p, nil
}

func (r *projectParamRepository) Create(ctx context.Context, params db.CreateProjectParamParams) (*domain.ProjectParam, error) {
	row, err := r.q.CreateProjectParam(ctx, params)
	if err != nil {
		return nil, errctx.Wrap(err, "CreateProjectParam", "name", params.Name)
	}
	p := mapProjectParamRow(row.ID, row.ProjectID, row.Name, row.FieldType, row.IsRequired, row.Options, row.Value)
	return &p, nil
}

func (r *projectParamRepository) Update(ctx context.Context, params db.UpdateProjectParamParams) (*domain.ProjectParam, error) {
	row, err := r.q.UpdateProjectParam(ctx, params)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "UpdateProjectParam", "id", params.ID)
	}
	p := mapProjectParamRow(row.ID, row.ProjectID, row.Name, row.FieldType, row.IsRequired, row.Options, row.Value)
	return &p, nil
}

func (r *projectParamRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return errctx.Wrap(r.q.DeleteProjectParamByID(ctx, id), "DeleteProjectParamByID", "id", id)
}

func mapProjectParamRow(id uuid.UUID, projectID uuid.NullUUID, name, fieldType string, isRequired bool, options pqtype.NullRawMessage, value sql.NullString) domain.ProjectParam {
	var val *string
	if value.Valid {
		v := value.String
		val = &v
	}
	var pid uuid.UUID
	if projectID.Valid {
		pid = projectID.UUID
	}
	return domain.ProjectParam{
		ID:         id,
		ProjectID:  pid,
		Name:       name,
		FieldType:  fieldType,
		IsSystem:   false,
		IsRequired: isRequired,
		Options:    JSONToOptions(options),
		Value:      val,
	}
}
