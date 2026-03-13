package repositories

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type TaskRepository interface {
	Create(ctx context.Context, t *domain.Task) (*domain.Task, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.Task, error)
	Search(ctx context.Context, projectID, ownerID, executorID, columnID *uuid.UUID) ([]domain.Task, error)
	Update(ctx context.Context, t *domain.Task) (*domain.Task, error)
	SoftDelete(ctx context.Context, id uuid.UUID, reason string) error
	ListProjectTaskKeys(ctx context.Context, projectID uuid.UUID) ([]string, error)
}

type taskRepository struct {
	q *db.Queries
}

func NewTaskRepository(q *db.Queries) TaskRepository {
	return &taskRepository{q: q}
}

func (r *taskRepository) Create(ctx context.Context, t *domain.Task) (*domain.Task, error) {
	projectID, err := uuid.Parse(t.ProjectID)
	if err != nil {
		return nil, err
	}
	ownerID, err := uuid.Parse(t.OwnerID)
	if err != nil {
		return nil, err
	}
	var executor uuid.NullUUID
	if t.ExecutorID != nil {
		if id, err := uuid.Parse(*t.ExecutorID); err == nil {
			executor = uuid.NullUUID{UUID: id, Valid: true}
		}
	}
	desc := sql.NullString{}
	if t.Description != nil {
		desc = sql.NullString{String: *t.Description, Valid: true}
	}
	var deadline sql.NullTime
	if t.Deadline != nil {
		deadline = sql.NullTime{Time: *t.Deadline, Valid: true}
	}
	columnID, err := uuid.Parse(t.ColumnID)
	if err != nil {
		return nil, err
	}
	var swimlane uuid.NullUUID
	if t.SwimlaneID != nil {
		if id, err := uuid.Parse(*t.SwimlaneID); err == nil {
			swimlane = uuid.NullUUID{UUID: id, Valid: true}
		}
	}

	row, err := r.q.CreateTask(ctx, db.CreateTaskParams{
		Key:         t.Key,
		ProjectID:   projectID,
		OwnerID:     ownerID,
		ExecutorID:  executor,
		Name:        t.Name,
		Description: desc,
		Deadline:    deadline,
		ColumnID:    columnID,
		SwimlaneID:  swimlane,
	})
	if err != nil {
		return nil, err
	}
	d := mapDBTaskToDomain(row)
	return &d, nil
}

func (r *taskRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	row, err := r.q.GetTaskByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	d := mapDBTaskToDomain(row)
	return &d, nil
}

func (r *taskRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.Task, error) {
	rows, err := r.q.ListProjectTasks(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapDBTaskToDomain(row))
	}
	return result, nil
}

func (r *taskRepository) Search(ctx context.Context, projectID, ownerID, executorID, columnID *uuid.UUID) ([]domain.Task, error) {
	params := db.SearchTasksParams{}
	if projectID != nil {
		params.Column1 = *projectID
	}
	if ownerID != nil {
		params.Column2 = *ownerID
	}
	if executorID != nil {
		params.Column3 = *executorID
	}
	if columnID != nil {
		params.Column4 = *columnID
	}
	rows, err := r.q.SearchTasks(ctx, params)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapDBTaskToDomain(row))
	}
	return result, nil
}

func (r *taskRepository) Update(ctx context.Context, t *domain.Task) (*domain.Task, error) {
	id, err := uuid.Parse(t.ID)
	if err != nil {
		return nil, err
	}
	name := sql.NullString{}
	if t.Name != "" {
		name = sql.NullString{String: t.Name, Valid: true}
	}
	var desc sql.NullString
	if t.Description != nil {
		desc = sql.NullString{String: *t.Description, Valid: true}
	}
	var deadline sql.NullTime
	if t.Deadline != nil {
		deadline = sql.NullTime{Time: *t.Deadline, Valid: true}
	}
	var executor uuid.NullUUID
	if t.ExecutorID != nil {
		if eid, err := uuid.Parse(*t.ExecutorID); err == nil {
			executor = uuid.NullUUID{UUID: eid, Valid: true}
		}
	}
	var column uuid.NullUUID
	if t.ColumnID != "" {
		if cid, err := uuid.Parse(t.ColumnID); err == nil {
			column = uuid.NullUUID{UUID: cid, Valid: true}
		}
	}
	var swimlane uuid.NullUUID
	if t.SwimlaneID != nil {
		if sid, err := uuid.Parse(*t.SwimlaneID); err == nil {
			swimlane = uuid.NullUUID{UUID: sid, Valid: true}
		}
	}

	row, err := r.q.UpdateTask(ctx, db.UpdateTaskParams{
		Name:        name,
		Description: desc,
		Deadline:    deadline,
		ExecutorID:  executor,
		ColumnID:    column,
		SwimlaneID:  swimlane,
		ID:          id,
	})
	if err != nil {
		return nil, err
	}
	d := mapDBTaskToDomain(row)
	return &d, nil
}

func (r *taskRepository) SoftDelete(ctx context.Context, id uuid.UUID, reason string) error {
	params := db.SoftDeleteTaskParams{
		ID:           id,
		DeleteReason: sql.NullString{String: strings.TrimSpace(reason), Valid: reason != ""},
	}
	return r.q.SoftDeleteTask(ctx, params)
}

func (r *taskRepository) ListProjectTaskKeys(ctx context.Context, projectID uuid.UUID) ([]string, error) {
	return r.q.ListProjectTaskKeys(ctx, projectID)
}

func mapDBTaskToDomain(t db.Task) domain.Task {
	var exec *string
	if t.ExecutorID.Valid {
		id := t.ExecutorID.UUID.String()
		exec = &id
	}
	var desc *string
	if t.Description.Valid {
		v := t.Description.String
		desc = &v
	}
	var deadline *time.Time
	if t.Deadline.Valid {
		rt := t.Deadline.Time
		deadline = &rt
	}
	var swimlane *string
	if t.SwimlaneID.Valid {
		id := t.SwimlaneID.UUID.String()
		swimlane = &id
	}
	var deletedAt *time.Time
	if t.DeletedAt.Valid {
		dt := t.DeletedAt.Time
		deletedAt = &dt
	}
	return domain.Task{
		ID:          t.ID.String(),
		Key:         t.Key,
		ProjectID:   t.ProjectID.String(),
		OwnerID:     t.OwnerID.String(),
		ExecutorID:  exec,
		Name:        t.Name,
		Description: desc,
		Deadline:    deadline,
		ColumnID:    t.ColumnID.String(),
		SwimlaneID:  swimlane,
		DeletedAt:   deletedAt,
	}
}
