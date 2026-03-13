package repositories

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type ProductBacklogRepository interface {
	Add(ctx context.Context, projectID, taskID uuid.UUID, order int32) (*domain.ProductBacklogItem, error)
	Remove(ctx context.Context, projectID, taskID uuid.UUID) error
	List(ctx context.Context, projectID uuid.UUID) ([]domain.ProductBacklogItem, error)
	UpdateOrder(ctx context.Context, projectID, taskID uuid.UUID, order int32) error
}

type productBacklogRepository struct {
	q *db.Queries
}

func NewProductBacklogRepository(q *db.Queries) ProductBacklogRepository {
	return &productBacklogRepository{q: q}
}

func (r *productBacklogRepository) Add(ctx context.Context, projectID, taskID uuid.UUID, order int32) (*domain.ProductBacklogItem, error) {
	row, err := r.q.AddToProductBacklog(ctx, db.AddToProductBacklogParams{
		ProjectID: projectID,
		TaskID:    taskID,
		Order:     order,
	})
	if err != nil {
		return nil, err
	}
	return mapDBProductBacklog(row), nil
}

func (r *productBacklogRepository) Remove(ctx context.Context, projectID, taskID uuid.UUID) error {
	return r.q.RemoveFromProductBacklog(ctx, db.RemoveFromProductBacklogParams{
		ProjectID: projectID,
		TaskID:    taskID,
	})
}

func (r *productBacklogRepository) List(ctx context.Context, projectID uuid.UUID) ([]domain.ProductBacklogItem, error) {
	rows, err := r.q.GetProductBacklog(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.ProductBacklogItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, *mapDBProductBacklog(row))
	}
	return result, nil
}

func (r *productBacklogRepository) UpdateOrder(ctx context.Context, projectID, taskID uuid.UUID, order int32) error {
	return r.q.UpdateProductBacklogOrder(ctx, db.UpdateProductBacklogOrderParams{
		ProjectID: projectID,
		TaskID:    taskID,
		Order:     order,
	})
}

func mapDBProductBacklog(row db.ProductBacklog) *domain.ProductBacklogItem {
	return &domain.ProductBacklogItem{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		TaskID:    row.TaskID,
		Order:     int(row.Order),
		AddedAt:   row.AddedAt,
	}
}

