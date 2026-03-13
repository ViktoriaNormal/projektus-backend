package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type SprintTaskRepository interface {
	AddTask(ctx context.Context, sprintID, taskID uuid.UUID, order *int32) (*domain.SprintTask, error)
	RemoveTask(ctx context.Context, sprintID, taskID uuid.UUID) error
	ListBySprint(ctx context.Context, sprintID uuid.UUID) ([]domain.SprintTask, error)
	UpdateTaskOrder(ctx context.Context, sprintID, taskID uuid.UUID, order int32) error
}

type sprintTaskRepository struct {
	q *db.Queries
}

func NewSprintTaskRepository(q *db.Queries) SprintTaskRepository {
	return &sprintTaskRepository{q: q}
}

func (r *sprintTaskRepository) AddTask(ctx context.Context, sprintID, taskID uuid.UUID, order *int32) (*domain.SprintTask, error) {
	var ord sql.NullInt32
	if order != nil {
		ord = sql.NullInt32{Int32: *order, Valid: true}
	}
	row, err := r.q.AddTaskToSprint(ctx, db.AddTaskToSprintParams{
		SprintID: sprintID,
		TaskID:   taskID,
		Order:    ord,
	})
	if err != nil {
		return nil, err
	}
	st := mapDBSprintTask(row)
	return &st, nil
}

func (r *sprintTaskRepository) RemoveTask(ctx context.Context, sprintID, taskID uuid.UUID) error {
	return r.q.RemoveTaskFromSprint(ctx, db.RemoveTaskFromSprintParams{
		SprintID: sprintID,
		TaskID:   taskID,
	})
}

func (r *sprintTaskRepository) ListBySprint(ctx context.Context, sprintID uuid.UUID) ([]domain.SprintTask, error) {
	rows, err := r.q.GetSprintTasks(ctx, sprintID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.SprintTask, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapDBSprintTask(row))
	}
	return result, nil
}

func (r *sprintTaskRepository) UpdateTaskOrder(ctx context.Context, sprintID, taskID uuid.UUID, order int32) error {
	return r.q.UpdateTaskOrder(ctx, db.UpdateTaskOrderParams{
		SprintID: sprintID,
		TaskID:   taskID,
		Order:    sql.NullInt32{Int32: order, Valid: true},
	})
}

func mapDBSprintTask(row db.SprintTask) domain.SprintTask {
	var ord int
	if row.Order.Valid {
		ord = int(row.Order.Int32)
	}
	return domain.SprintTask{
		ID:       row.ID,
		SprintID: row.SprintID,
		TaskID:   row.TaskID,
		Order:    ord,
		AddedAt:  row.AddedAt,
	}
}

