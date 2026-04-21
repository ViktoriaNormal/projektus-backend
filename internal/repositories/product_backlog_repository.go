package repositories

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
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
		SortOrder: order,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "AddToProductBacklog", "projectID", projectID, "taskID", taskID)
	}
	return mapDBBacklog(row), nil
}

func (r *productBacklogRepository) Remove(ctx context.Context, projectID, taskID uuid.UUID) error {
	return errctx.Wrap(r.q.RemoveFromProductBacklog(ctx, db.RemoveFromProductBacklogParams{
		ProjectID: projectID,
		TaskID:    taskID,
	}), "RemoveFromProductBacklog", "projectID", projectID, "taskID", taskID)
}

func (r *productBacklogRepository) List(ctx context.Context, projectID uuid.UUID) ([]domain.ProductBacklogItem, error) {
	rows, err := r.q.GetProductBacklog(ctx, projectID)
	if err != nil {
		return nil, errctx.Wrap(err, "GetProductBacklog", "projectID", projectID)
	}
	result := make([]domain.ProductBacklogItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, *mapDBBacklog(row))
	}
	return result, nil
}

func (r *productBacklogRepository) UpdateOrder(ctx context.Context, projectID, taskID uuid.UUID, order int32) error {
	return errctx.Wrap(r.q.UpdateProductBacklogOrder(ctx, db.UpdateProductBacklogOrderParams{
		ProjectID: projectID,
		TaskID:    taskID,
		SortOrder: order,
	}), "UpdateProductBacklogOrder", "projectID", projectID, "taskID", taskID)
}

func mapDBBacklog(row db.Backlog) *domain.ProductBacklogItem {
	return &domain.ProductBacklogItem{
		ProjectID: row.ProjectID,
		TaskID:    row.TaskID,
		Order:     int(row.SortOrder),
	}
}
