package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type KanbanRepository interface {
	GetWipLimits(ctx context.Context, projectID uuid.UUID) ([]domain.WipLimit, error)
	UpdateColumnWipLimit(ctx context.Context, columnID uuid.UUID, limit *int16) error
	UpdateSwimlaneWipLimit(ctx context.Context, swimlaneID uuid.UUID, limit *int16) error
	GetCurrentWipCounts(ctx context.Context, boardID uuid.UUID) ([]domain.WipCount, error)
}

type kanbanRepository struct {
	q *db.Queries
}

func NewKanbanRepository(q *db.Queries) KanbanRepository {
	return &kanbanRepository{q: q}
}

func (r *kanbanRepository) GetWipLimits(ctx context.Context, projectID uuid.UUID) ([]domain.WipLimit, error) {
	rows, err := r.q.GetWipLimits(ctx, uuid.NullUUID{UUID: projectID, Valid: true})
	if err != nil {
		return nil, err
	}
	result := make([]domain.WipLimit, 0, len(rows))
	for _, row := range rows {
		var colID *uuid.UUID
		if row.ColumnID != uuid.Nil {
			id := row.ColumnID
			colID = &id
		}
		var swimID *uuid.UUID
		if row.SwimlaneID.Valid {
			id := row.SwimlaneID.UUID
			swimID = &id
		}
		limitVal := int(row.Limit)
		limitPtr := &limitVal
		result = append(result, domain.WipLimit{
			BoardID:   row.BoardID,
			ColumnID:  colID,
			SwimlaneID: swimID,
			Limit:     limitPtr,
		})
	}
	return result, nil
}

func (r *kanbanRepository) UpdateColumnWipLimit(ctx context.Context, columnID uuid.UUID, limit *int16) error {
	param := db.UpdateColumnWipLimitParams{
		ID:       columnID,
		WipLimit: sql.NullInt16{},
	}
	if limit != nil {
		param.WipLimit = sql.NullInt16{Int16: *limit, Valid: true}
	}
	return r.q.UpdateColumnWipLimit(ctx, param)
}

func (r *kanbanRepository) UpdateSwimlaneWipLimit(ctx context.Context, swimlaneID uuid.UUID, limit *int16) error {
	param := db.UpdateSwimlaneWipLimitParams{
		ID:       swimlaneID,
		WipLimit: sql.NullInt16{},
	}
	if limit != nil {
		param.WipLimit = sql.NullInt16{Int16: *limit, Valid: true}
	}
	return r.q.UpdateSwimlaneWipLimit(ctx, param)
}

func (r *kanbanRepository) GetCurrentWipCounts(ctx context.Context, boardID uuid.UUID) ([]domain.WipCount, error) {
	rows, err := r.q.GetCurrentWipCounts(ctx, boardID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.WipCount, 0, len(rows))
	for _, row := range rows {
		boardID := row.BoardID
		var colID *uuid.UUID
		if row.ColumnID != uuid.Nil {
			id := row.ColumnID
			colID = &id
		}
		var swimID *uuid.UUID
		if row.SwimlaneID.Valid {
			id := row.SwimlaneID.UUID
			swimID = &id
		}
		countVal := int(row.Count)
		result = append(result, domain.WipCount{
			BoardID:   boardID,
			ColumnID:  colID,
			SwimlaneID: swimID,
			Count:     countVal,
		})
	}
	return result, nil
}

