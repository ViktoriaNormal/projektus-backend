package repositories

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type TaskDependencyRepository interface {
	Add(ctx context.Context, taskID, dependsOnID uuid.UUID, depType domain.TaskDependencyType) (*domain.TaskDependency, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.TaskDependency, error)
	Remove(ctx context.Context, dependencyID uuid.UUID) error
	RemoveInverse(ctx context.Context, taskID, dependsOnTaskID uuid.UUID) error
	RemoveAllForTask(ctx context.Context, taskID uuid.UUID) error
	ListFor(ctx context.Context, taskID uuid.UUID) ([]domain.TaskDependency, error)
	ListDependants(ctx context.Context, taskID uuid.UUID) ([]domain.TaskDependency, error)
}

type taskDependencyRepository struct {
	q *db.Queries
}

func NewTaskDependencyRepository(q *db.Queries) TaskDependencyRepository {
	return &taskDependencyRepository{q: q}
}

func (r *taskDependencyRepository) Add(ctx context.Context, taskID, dependsOnID uuid.UUID, depType domain.TaskDependencyType) (*domain.TaskDependency, error) {
	row, err := r.q.AddTaskDependency(ctx, db.AddTaskDependencyParams{
		TaskID:          taskID,
		DependsOnTaskID: dependsOnID,
		DependencyType:  string(depType),
	})
	if err != nil {
		return nil, errctx.Wrap(err, "AddTaskDependency", "taskID", taskID, "dependsOnID", dependsOnID)
	}
	d := domain.TaskDependency{
		ID:              row.ID,
		TaskID:          row.TaskID,
		DependsOnTaskID: row.DependsOnTaskID,
		Type:            domain.TaskDependencyType(row.DependencyType),
	}
	return &d, nil
}

func (r *taskDependencyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.TaskDependency, error) {
	row, err := r.q.GetTaskDependencyByID(ctx, id)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetTaskDependencyByID", "id", id)
	}
	return &domain.TaskDependency{
		ID:              row.ID,
		TaskID:          row.TaskID,
		DependsOnTaskID: row.DependsOnTaskID,
		Type:            domain.TaskDependencyType(row.DependencyType),
	}, nil
}

func (r *taskDependencyRepository) Remove(ctx context.Context, dependencyID uuid.UUID) error {
	return errctx.Wrap(r.q.RemoveTaskDependency(ctx, dependencyID), "RemoveTaskDependency", "dependencyID", dependencyID)
}

func (r *taskDependencyRepository) RemoveInverse(ctx context.Context, taskID, dependsOnTaskID uuid.UUID) error {
	return errctx.Wrap(r.q.RemoveInverseDependency(ctx, db.RemoveInverseDependencyParams{
		TaskID:          taskID,
		DependsOnTaskID: dependsOnTaskID,
	}), "RemoveInverseDependency", "taskID", taskID, "dependsOnTaskID", dependsOnTaskID)
}

func (r *taskDependencyRepository) RemoveAllForTask(ctx context.Context, taskID uuid.UUID) error {
	return errctx.Wrap(r.q.RemoveAllTaskDependencies(ctx, taskID), "RemoveAllTaskDependencies", "taskID", taskID)
}

func (r *taskDependencyRepository) ListFor(ctx context.Context, taskID uuid.UUID) ([]domain.TaskDependency, error) {
	rows, err := r.q.ListTaskDependencies(ctx, taskID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListTaskDependencies", "taskID", taskID)
	}
	result := make([]domain.TaskDependency, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.TaskDependency{
			ID:              row.ID,
			TaskID:          row.TaskID,
			DependsOnTaskID: row.DependsOnTaskID,
			Type:            domain.TaskDependencyType(row.DependencyType),
		})
	}
	return result, nil
}

func (r *taskDependencyRepository) ListDependants(ctx context.Context, taskID uuid.UUID) ([]domain.TaskDependency, error) {
	rows, err := r.q.ListTaskDependants(ctx, taskID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListTaskDependants", "taskID", taskID)
	}
	result := make([]domain.TaskDependency, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.TaskDependency{
			ID:              row.ID,
			TaskID:          row.TaskID,
			DependsOnTaskID: row.DependsOnTaskID,
			Type:            domain.TaskDependencyType(row.DependencyType),
		})
	}
	return result, nil
}
