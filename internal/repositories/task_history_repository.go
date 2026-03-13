package repositories

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type TaskHistoryRepository interface {
	RecordStatusChange(ctx context.Context, taskID, columnID uuid.UUID, enteredAt time.Time, leftAt *time.Time) error
	GetTaskStatusHistory(ctx context.Context, taskID uuid.UUID) ([]db.TaskStatusHistory, error)
	GetProjectCycleTimes(ctx context.Context, projectID uuid.UUID) ([]domain.CycleTimeData, error)
}

type taskHistoryRepository struct {
	q *db.Queries
}

func NewTaskHistoryRepository(q *db.Queries) TaskHistoryRepository {
	return &taskHistoryRepository{q: q}
}

func (r *taskHistoryRepository) RecordStatusChange(ctx context.Context, taskID, columnID uuid.UUID, enteredAt time.Time, leftAt *time.Time) error {
	var left sql.NullTime
	if leftAt != nil {
		left = sql.NullTime{Time: *leftAt, Valid: true}
	}
	_, err := r.q.RecordTaskStatusChange(ctx, db.RecordTaskStatusChangeParams{
		TaskID:    taskID,
		ColumnID:  columnID,
		EnteredAt: enteredAt,
		LeftAt:    left,
	})
	return err
}

func (r *taskHistoryRepository) GetTaskStatusHistory(ctx context.Context, taskID uuid.UUID) ([]db.TaskStatusHistory, error) {
	return r.q.GetTaskStatusHistory(ctx, taskID)
}

func (r *taskHistoryRepository) GetProjectCycleTimes(ctx context.Context, projectID uuid.UUID) ([]domain.CycleTimeData, error) {
	rows, err := r.q.GetCompletedTasksCycleTime(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.CycleTimeData, 0, len(rows))
	for _, row := range rows {
		secs, err := strconv.ParseFloat(row.CycleTimeSeconds, 64)
		if err != nil {
			continue
		}
		hours := secs / 3600.0
		ct := domain.CycleTimeData{
			TaskID:         row.TaskID,
			CycleTimeHours: hours,
		}
		if t, ok := row.CompletedAt.(time.Time); ok {
			ct.CompletedAt = t
		}
		result = append(result, ct)
	}
	return result, nil
}

