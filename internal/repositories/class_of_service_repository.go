package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type ClassOfServiceRepository interface {
	UpdateTaskClass(ctx context.Context, taskID uuid.UUID, class domain.ClassOfService) error
	GetTasksByClass(ctx context.Context, projectID uuid.UUID, class domain.ClassOfService) ([]domain.Task, error)
}

type classOfServiceRepository struct {
	q *db.Queries
}

func NewClassOfServiceRepository(q *db.Queries) ClassOfServiceRepository {
	return &classOfServiceRepository{q: q}
}

func (r *classOfServiceRepository) UpdateTaskClass(ctx context.Context, taskID uuid.UUID, class domain.ClassOfService) error {
	return r.q.UpdateTaskClassOfService(ctx, db.UpdateTaskClassOfServiceParams{
		ID:             taskID,
		ClassOfService: sql.NullString{String: string(class), Valid: true},
	})
}

func (r *classOfServiceRepository) GetTasksByClass(ctx context.Context, projectID uuid.UUID, class domain.ClassOfService) ([]domain.Task, error) {
	rows, err := r.q.GetTasksByClassOfService(ctx, db.GetTasksByClassOfServiceParams{
		ProjectID:      projectID,
		ClassOfService: sql.NullString{String: string(class), Valid: true},
	})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapDBTaskToDomain(row))
	}
	return result, nil
}

