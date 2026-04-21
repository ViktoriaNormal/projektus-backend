package repositories

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type TaskWatcherRepository interface {
	Add(ctx context.Context, taskID, memberID uuid.UUID) error
	Remove(ctx context.Context, taskID, memberID uuid.UUID) error
	List(ctx context.Context, taskID uuid.UUID) ([]domain.TaskWatcher, error)
}

type taskWatcherRepository struct {
	q *db.Queries
}

func NewTaskWatcherRepository(q *db.Queries) TaskWatcherRepository {
	return &taskWatcherRepository{q: q}
}

func (r *taskWatcherRepository) Add(ctx context.Context, taskID, memberID uuid.UUID) error {
	return errctx.Wrap(r.q.AddTaskWatcher(ctx, db.AddTaskWatcherParams{
		TaskID:   taskID,
		MemberID: memberID,
	}), "AddTaskWatcher", "taskID", taskID, "memberID", memberID)
}

func (r *taskWatcherRepository) Remove(ctx context.Context, taskID, memberID uuid.UUID) error {
	return errctx.Wrap(r.q.RemoveTaskWatcher(ctx, db.RemoveTaskWatcherParams{
		TaskID:   taskID,
		MemberID: memberID,
	}), "RemoveTaskWatcher", "taskID", taskID, "memberID", memberID)
}

func (r *taskWatcherRepository) List(ctx context.Context, taskID uuid.UUID) ([]domain.TaskWatcher, error) {
	rows, err := r.q.ListTaskWatchers(ctx, taskID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListTaskWatchers", "taskID", taskID)
	}
	result := make([]domain.TaskWatcher, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.TaskWatcher{
			TaskID:   row.TaskID,
			MemberID: row.MemberID,
		})
	}
	return result, nil
}
