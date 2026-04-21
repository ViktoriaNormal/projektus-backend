package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type TaskFieldValueRepository interface {
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.TaskFieldValue, error)
	Upsert(ctx context.Context, taskID, fieldID uuid.UUID, valueText, valueNumber *string, valueDatetime *time.Time) error
}

type taskFieldValueRepository struct {
	q *db.Queries
}

func NewTaskFieldValueRepository(q *db.Queries) TaskFieldValueRepository {
	return &taskFieldValueRepository{q: q}
}

func (r *taskFieldValueRepository) ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.TaskFieldValue, error) {
	rows, err := r.q.GetTaskFieldValues(ctx, taskID)
	if err != nil {
		return nil, errctx.Wrap(err, "GetTaskFieldValues", "taskID", taskID)
	}
	result := make([]domain.TaskFieldValue, 0, len(rows))
	for _, row := range rows {
		fv := domain.TaskFieldValue{
			TaskID:  row.TaskID,
			FieldID: row.FieldID,
		}
		if row.ValueText.Valid {
			v := row.ValueText.String
			fv.ValueText = &v
		}
		if row.ValueNumber.Valid {
			v := row.ValueNumber.String
			fv.ValueNumber = &v
		}
		if row.ValueDatetime.Valid {
			t := row.ValueDatetime.Time
			fv.ValueDatetime = &t
		}
		result = append(result, fv)
	}
	return result, nil
}

func (r *taskFieldValueRepository) Upsert(ctx context.Context, taskID, fieldID uuid.UUID, valueText, valueNumber *string, valueDatetime *time.Time) error {
	params := db.UpsertTaskFieldValueParams{
		TaskID:  taskID,
		FieldID: fieldID,
	}
	if valueText != nil {
		params.ValueText = sql.NullString{String: *valueText, Valid: true}
	}
	if valueNumber != nil {
		params.ValueNumber = sql.NullString{String: *valueNumber, Valid: true}
	}
	if valueDatetime != nil {
		params.ValueDatetime = sql.NullTime{Time: *valueDatetime, Valid: true}
	}
	return errctx.Wrap(r.q.UpsertTaskFieldValue(ctx, params), "UpsertTaskFieldValue", "taskID", taskID, "fieldID", fieldID)
}
