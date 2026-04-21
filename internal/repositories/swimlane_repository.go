package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type SwimlaneRepository interface {
	List(ctx context.Context, boardID uuid.UUID) ([]domain.Swimlane, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Swimlane, error)
	Create(ctx context.Context, s *domain.Swimlane) (*domain.Swimlane, error)
	Update(ctx context.Context, s *domain.Swimlane) (*domain.Swimlane, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateOrder(ctx context.Context, id uuid.UUID, order int16) error
	CountTasks(ctx context.Context, id uuid.UUID) (int, error)
	ClearFromTasks(ctx context.Context, swimlaneID uuid.UUID) error
}

type swimlaneRepository struct {
	q *db.Queries
}

func NewSwimlaneRepository(q *db.Queries) SwimlaneRepository {
	return &swimlaneRepository{q: q}
}

func (r *swimlaneRepository) List(ctx context.Context, boardID uuid.UUID) ([]domain.Swimlane, error) {
	rows, err := r.q.ListBoardSwimlanes(ctx, boardID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListBoardSwimlanes", "boardID", boardID)
	}
	result := make([]domain.Swimlane, len(rows))
	for i, row := range rows {
		result[i] = mapDBSwimlaneToDomain(row)
	}
	return result, nil
}

func (r *swimlaneRepository) Create(ctx context.Context, s *domain.Swimlane) (*domain.Swimlane, error) {
	var wipLimit sql.NullInt16
	if s.WipLimit != nil {
		wipLimit = sql.NullInt16{Int16: *s.WipLimit, Valid: true}
	}
	row, err := r.q.CreateSwimlane(ctx, db.CreateSwimlaneParams{
		BoardID:   s.BoardID,
		Name:      s.Name,
		WipLimit:  wipLimit,
		SortOrder: s.Order,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateSwimlane", "boardID", s.BoardID, "name", s.Name)
	}
	created := mapDBSwimlaneToDomain(row)
	return &created, nil
}

func (r *swimlaneRepository) Update(ctx context.Context, s *domain.Swimlane) (*domain.Swimlane, error) {
	params := db.UpdateSwimlaneParams{ID: s.ID}
	if s.Name != "" {
		params.Name = sql.NullString{String: s.Name, Valid: true}
	}
	if s.WipLimit != nil {
		params.WipLimit = sql.NullInt16{Int16: *s.WipLimit, Valid: true}
	}
	params.SortOrder = sql.NullInt16{Int16: s.Order, Valid: true}
	row, err := r.q.UpdateSwimlane(ctx, params)
	if err != nil {
		return nil, errctx.Wrap(err, "UpdateSwimlane", "id", s.ID)
	}
	updated := mapDBSwimlaneToDomain(row)
	return &updated, nil
}

func (r *swimlaneRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return errctx.Wrap(r.q.DeleteSwimlane(ctx, id), "DeleteSwimlane", "id", id)
}

func (r *swimlaneRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Swimlane, error) {
	row, err := r.q.GetSwimlaneByID(ctx, id)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetSwimlaneByID", "id", id)
	}
	s := mapDBSwimlaneToDomain(row)
	return &s, nil
}

func (r *swimlaneRepository) UpdateOrder(ctx context.Context, id uuid.UUID, order int16) error {
	return errctx.Wrap(r.q.UpdateSwimlaneOrder(ctx, db.UpdateSwimlaneOrderParams{ID: id, SortOrder: order}), "UpdateSwimlaneOrder", "id", id)
}

func (r *swimlaneRepository) CountTasks(ctx context.Context, id uuid.UUID) (int, error) {
	count, err := r.q.CountTasksInSwimlane(ctx, uuid.NullUUID{UUID: id, Valid: true})
	if err != nil {
		return 0, errctx.Wrap(err, "CountTasksInSwimlane", "id", id)
	}
	return int(count), nil
}

func (r *swimlaneRepository) ClearFromTasks(ctx context.Context, swimlaneID uuid.UUID) error {
	return errctx.Wrap(r.q.ClearSwimlaneFromTasks(ctx, uuid.NullUUID{UUID: swimlaneID, Valid: true}), "ClearSwimlaneFromTasks", "swimlaneID", swimlaneID)
}

func mapDBSwimlaneToDomain(s db.Swimlane) domain.Swimlane {
	var wip *int16
	if s.WipLimit.Valid {
		v := s.WipLimit.Int16
		wip = &v
	}
	return domain.Swimlane{
		ID:       s.ID,
		BoardID:  s.BoardID,
		Name:     s.Name,
		WipLimit: wip,
		Order:    s.SortOrder,
	}
}
