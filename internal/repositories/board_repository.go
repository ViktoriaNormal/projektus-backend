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

type BoardRepository interface {
	CreateBoard(ctx context.Context, b *domain.Board) (*domain.Board, error)
	GetBoardByID(ctx context.Context, id uuid.UUID) (*domain.Board, error)
	ListProjectBoards(ctx context.Context, projectID uuid.UUID) ([]domain.Board, error)
	UpdateBoard(ctx context.Context, b *domain.Board) (*domain.Board, error)
	DeleteBoard(ctx context.Context, id uuid.UUID) error
	UpdateBoardOrder(ctx context.Context, id uuid.UUID, order int16) error
	UnsetDefaultBoardByProjectID(ctx context.Context, projectID uuid.UUID) error
}

type boardRepository struct {
	q *db.Queries
}

func NewBoardRepository(q *db.Queries) BoardRepository {
	return &boardRepository{q: q}
}

func (r *boardRepository) CreateBoard(ctx context.Context, b *domain.Board) (*domain.Board, error) {
	projectID := ptrToNullUUID(b.ProjectID)
	templateID := ptrToNullUUID(b.TemplateID)
	desc := sql.NullString{}
	if b.Description != nil {
		desc = sql.NullString{String: *b.Description, Valid: true}
	}
	row, err := r.q.CreateBoard(ctx, db.CreateBoardParams{
		ProjectID:       projectID,
		TemplateID:      templateID,
		Name:            b.Name,
		Description:     desc,
		IsDefault:       b.IsDefault,
		SortOrder:       b.Order,
		PriorityType:    b.PriorityType,
		EstimationUnit:  b.EstimationUnit,
		SwimlaneGroupBy: b.SwimlaneGroupBy,
		PriorityOptions: OptionsToJSON(b.PriorityOptions),
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateBoard", "name", b.Name)
	}
	dbBoard := mapBoardRowToDomain(row.ID, row.ProjectID, row.TemplateID, row.Name, row.Description, row.IsDefault, row.SortOrder, row.PriorityType, row.EstimationUnit, row.SwimlaneGroupBy, row.PriorityOptions)
	return &dbBoard, nil
}

func (r *boardRepository) GetBoardByID(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
	row, err := r.q.GetBoardByID(ctx, id)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetBoardByID", "id", id)
	}
	b := mapBoardRowToDomain(row.ID, row.ProjectID, row.TemplateID, row.Name, row.Description, row.IsDefault, row.SortOrder, row.PriorityType, row.EstimationUnit, row.SwimlaneGroupBy, row.PriorityOptions)
	return &b, nil
}

func (r *boardRepository) ListProjectBoards(ctx context.Context, projectID uuid.UUID) ([]domain.Board, error) {
	rows, err := r.q.ListProjectBoards(ctx, uuid.NullUUID{UUID: projectID, Valid: true})
	if err != nil {
		return nil, errctx.Wrap(err, "ListProjectBoards", "projectID", projectID)
	}
	result := make([]domain.Board, len(rows))
	for i, row := range rows {
		result[i] = mapBoardRowToDomain(row.ID, row.ProjectID, row.TemplateID, row.Name, row.Description, row.IsDefault, row.SortOrder, row.PriorityType, row.EstimationUnit, row.SwimlaneGroupBy, row.PriorityOptions)
	}
	return result, nil
}

func (r *boardRepository) UpdateBoard(ctx context.Context, b *domain.Board) (*domain.Board, error) {
	params := db.UpdateBoardParams{ID: b.ID}
	if b.Name != "" {
		params.Name = sql.NullString{String: b.Name, Valid: true}
	}
	if b.Description != nil {
		params.Description = sql.NullString{String: *b.Description, Valid: true}
	}
	params.SortOrder = sql.NullInt16{Int16: b.Order, Valid: true}
	if b.PriorityType != "" {
		params.PriorityType = sql.NullString{String: b.PriorityType, Valid: true}
	}
	if b.EstimationUnit != "" {
		params.EstimationUnit = sql.NullString{String: b.EstimationUnit, Valid: true}
	}
	params.SwimlaneGroupBy = sql.NullString{String: b.SwimlaneGroupBy, Valid: true}
	params.IsDefault = sql.NullBool{Bool: b.IsDefault, Valid: true}
	if len(b.PriorityOptions) > 0 {
		params.PriorityOptions = OptionsToJSON(b.PriorityOptions)
	}

	row, err := r.q.UpdateBoard(ctx, params)
	if err != nil {
		return nil, errctx.Wrap(err, "UpdateBoard", "id", b.ID)
	}
	updated := mapBoardRowToDomain(row.ID, row.ProjectID, row.TemplateID, row.Name, row.Description, row.IsDefault, row.SortOrder, row.PriorityType, row.EstimationUnit, row.SwimlaneGroupBy, row.PriorityOptions)
	return &updated, nil
}

func (r *boardRepository) DeleteBoard(ctx context.Context, id uuid.UUID) error {
	return errctx.Wrap(r.q.DeleteBoard(ctx, id), "DeleteBoard", "id", id)
}

func (r *boardRepository) UnsetDefaultBoardByProjectID(ctx context.Context, projectID uuid.UUID) error {
	return errctx.Wrap(r.q.UnsetDefaultBoardByProjectID(ctx, uuid.NullUUID{UUID: projectID, Valid: true}), "UnsetDefaultBoardByProjectID", "projectID", projectID)
}

func (r *boardRepository) UpdateBoardOrder(ctx context.Context, id uuid.UUID, order int16) error {
	return errctx.Wrap(r.q.UpdateBoardOrder(ctx, db.UpdateBoardOrderParams{ID: id, SortOrder: order}), "UpdateBoardOrder", "id", id)
}

func mapBoardRowToDomain(id uuid.UUID, projectID, templateID uuid.NullUUID, name string, desc sql.NullString, isDefault bool, sortOrder int16, priorityType, estimationUnit, swimlaneGroupBy string, priorityOptions pqtype.NullRawMessage) domain.Board {
	var d *string
	if desc.Valid {
		v := desc.String
		d = &v
	}
	return domain.Board{
		ID:              id,
		ProjectID:       nullUUIDToPtr(projectID),
		TemplateID:      nullUUIDToPtr(templateID),
		Name:            name,
		Description:     d,
		IsDefault:       isDefault,
		Order:           sortOrder,
		PriorityType:    priorityType,
		EstimationUnit:  estimationUnit,
		SwimlaneGroupBy: swimlaneGroupBy,
		PriorityOptions: JSONToOptions(priorityOptions),
	}
}
