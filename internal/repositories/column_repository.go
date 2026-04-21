package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type ColumnRepository interface {
	List(ctx context.Context, boardID uuid.UUID) ([]domain.Column, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Column, error)
	Create(ctx context.Context, c *domain.Column) (*domain.Column, error)
	Update(ctx context.Context, c *domain.Column) (*domain.Column, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateOrder(ctx context.Context, id uuid.UUID, order int16) error
	CountTasks(ctx context.Context, id uuid.UUID) (int, error)
}

type columnRepository struct {
	q *db.Queries
}

func NewColumnRepository(q *db.Queries) ColumnRepository {
	return &columnRepository{q: q}
}

func (r *columnRepository) List(ctx context.Context, boardID uuid.UUID) ([]domain.Column, error) {
	rows, err := r.q.ListBoardColumns(ctx, boardID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListBoardColumns", "boardID", boardID)
	}
	result := make([]domain.Column, len(rows))
	for i, row := range rows {
		result[i] = mapDBColumnToDomain(row)
	}
	return result, nil
}

func (r *columnRepository) Create(ctx context.Context, c *domain.Column) (*domain.Column, error) {
	var systemType sql.NullString
	if c.SystemType != nil {
		systemType = sql.NullString{String: string(*c.SystemType), Valid: true}
	}
	var wipLimit sql.NullInt16
	if c.WipLimit != nil {
		wipLimit = sql.NullInt16{Int16: *c.WipLimit, Valid: true}
	}
	row, err := r.q.CreateColumn(ctx, db.CreateColumnParams{
		BoardID:    c.BoardID,
		Name:       c.Name,
		SystemType: systemType,
		WipLimit:   wipLimit,
		SortOrder:  c.Order,
		IsLocked:   c.IsLocked,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateColumn", "boardID", c.BoardID, "name", c.Name)
	}
	created := mapDBColumnToDomain(row)
	return &created, nil
}

func (r *columnRepository) Update(ctx context.Context, c *domain.Column) (*domain.Column, error) {
	params := db.UpdateColumnParams{ID: c.ID}
	if c.Name != "" {
		params.Name = sql.NullString{String: c.Name, Valid: true}
	}
	if c.SystemType != nil {
		params.SystemType = sql.NullString{String: string(*c.SystemType), Valid: true}
	}
	if c.WipLimit != nil {
		params.WipLimit = sql.NullInt16{Int16: *c.WipLimit, Valid: true}
	}
	params.SortOrder = sql.NullInt16{Int16: c.Order, Valid: true}
	row, err := r.q.UpdateColumn(ctx, params)
	if err != nil {
		return nil, errctx.Wrap(err, "UpdateColumn", "id", c.ID)
	}
	updated := mapDBColumnToDomain(row)
	return &updated, nil
}

func (r *columnRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return errctx.Wrap(r.q.DeleteColumn(ctx, id), "DeleteColumn", "id", id)
}

func (r *columnRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Column, error) {
	row, err := r.q.GetColumnByID(ctx, id)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetColumnByID", "id", id)
	}
	c := mapDBColumnToDomain(row)
	return &c, nil
}

func (r *columnRepository) UpdateOrder(ctx context.Context, id uuid.UUID, order int16) error {
	return errctx.Wrap(r.q.UpdateColumnOrder(ctx, db.UpdateColumnOrderParams{ID: id, SortOrder: order}), "UpdateColumnOrder", "id", id)
}

func (r *columnRepository) CountTasks(ctx context.Context, id uuid.UUID) (int, error) {
	count, err := r.q.CountTasksInColumn(ctx, uuid.NullUUID{UUID: id, Valid: true})
	if err != nil {
		return 0, errctx.Wrap(err, "CountTasksInColumn", "id", id)
	}
	return int(count), nil
}

func mapDBColumnToDomain(c db.Column) domain.Column {
	var systemType *domain.SystemStatusType
	if c.SystemType.Valid {
		st := domain.SystemStatusType(c.SystemType.String)
		systemType = &st
	}
	var wip *int16
	if c.WipLimit.Valid {
		v := c.WipLimit.Int16
		wip = &v
	}
	return domain.Column{
		ID:         c.ID,
		BoardID:    c.BoardID,
		Name:       c.Name,
		SystemType: systemType,
		WipLimit:   wip,
		Order:      c.SortOrder,
		IsLocked:   c.IsLocked,
	}
}
