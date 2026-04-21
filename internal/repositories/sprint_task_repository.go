package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type SprintTaskRepository interface {
	AddTask(ctx context.Context, sprintID, taskID uuid.UUID, order *int32) (*domain.SprintTask, error)
	RemoveTask(ctx context.Context, sprintID, taskID uuid.UUID) error
	RemoveTaskFromAllSprints(ctx context.Context, taskID uuid.UUID) error
	ListBySprint(ctx context.Context, sprintID uuid.UUID) ([]domain.SprintTask, error)
	UpdateTaskOrder(ctx context.Context, sprintID, taskID uuid.UUID, order int32) error
	ListSprintTasksFull(ctx context.Context, sprintID uuid.UUID) ([]domain.Task, error)
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
		SprintID:  sprintID,
		TaskID:    taskID,
		SortOrder: ord,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "AddTaskToSprint", "sprintID", sprintID, "taskID", taskID)
	}
	st := mapDBSprintTask(row)
	return &st, nil
}

func (r *sprintTaskRepository) RemoveTask(ctx context.Context, sprintID, taskID uuid.UUID) error {
	return errctx.Wrap(r.q.RemoveTaskFromSprint(ctx, db.RemoveTaskFromSprintParams{
		SprintID: sprintID,
		TaskID:   taskID,
	}), "RemoveTaskFromSprint", "sprintID", sprintID, "taskID", taskID)
}

func (r *sprintTaskRepository) RemoveTaskFromAllSprints(ctx context.Context, taskID uuid.UUID) error {
	return errctx.Wrap(r.q.RemoveTaskFromAllSprints(ctx, taskID), "RemoveTaskFromAllSprints", "taskID", taskID)
}

func (r *sprintTaskRepository) ListBySprint(ctx context.Context, sprintID uuid.UUID) ([]domain.SprintTask, error) {
	rows, err := r.q.GetSprintTasks(ctx, sprintID)
	if err != nil {
		return nil, errctx.Wrap(err, "GetSprintTasks", "sprintID", sprintID)
	}
	result := make([]domain.SprintTask, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapDBSprintTask(row))
	}
	return result, nil
}

func (r *sprintTaskRepository) UpdateTaskOrder(ctx context.Context, sprintID, taskID uuid.UUID, order int32) error {
	return errctx.Wrap(r.q.UpdateTaskOrder(ctx, db.UpdateTaskOrderParams{
		SprintID:  sprintID,
		TaskID:    taskID,
		SortOrder: sql.NullInt32{Int32: order, Valid: true},
	}), "UpdateTaskOrder", "sprintID", sprintID, "taskID", taskID)
}

func (r *sprintTaskRepository) ListSprintTasksFull(ctx context.Context, sprintID uuid.UUID) ([]domain.Task, error) {
	rows, err := r.q.ListSprintTasksFull(ctx, sprintID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListSprintTasksFull", "sprintID", sprintID)
	}
	result := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapSprintTaskFullRowToDomain(row))
	}
	return result, nil
}

func mapSprintTaskFullRowToDomain(row db.ListSprintTasksFullRow) domain.Task {
	t := domain.Task{
		ID:        row.ID,
		Key:       row.Key,
		ProjectID: row.ProjectID,
		BoardID:   row.BoardID,
		OwnerID:   row.OwnerID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
	}
	uid := row.OwnerUserID
	t.OwnerUserID = &uid
	if row.ExecutorID.Valid {
		id := row.ExecutorID.UUID
		t.ExecutorID = &id
	}
	if row.ExecutorUserID.Valid {
		euid := row.ExecutorUserID.UUID
		t.ExecutorUserID = &euid
	}
	if row.Description.Valid {
		t.Description = &row.Description.String
	}
	if row.Deadline.Valid {
		d := row.Deadline.Time
		t.Deadline = &d
	}
	if row.ColumnID.Valid {
		id := row.ColumnID.UUID
		t.ColumnID = &id
	}
	if row.SwimlaneID.Valid {
		id := row.SwimlaneID.UUID
		t.SwimlaneID = &id
	}
	if row.DeletedAt.Valid {
		d := row.DeletedAt.Time
		t.DeletedAt = &d
	}
	if row.Priority.Valid {
		t.Priority = &row.Priority.String
	}
	if row.Estimation.Valid {
		t.Estimation = &row.Estimation.String
	}
	if row.ColumnName.Valid {
		t.ColumnName = &row.ColumnName.String
	}
	if row.ColumnSystemType.Valid {
		t.ColumnSystemType = &row.ColumnSystemType.String
	}
	return t
}

func mapDBSprintTask(row db.SprintTask) domain.SprintTask {
	var ord int
	if row.SortOrder.Valid {
		ord = int(row.SortOrder.Int32)
	}
	return domain.SprintTask{
		SprintID: row.SprintID,
		TaskID:   row.TaskID,
		Order:    ord,
	}
}
