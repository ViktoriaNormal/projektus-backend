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

	AddWatcher(ctx context.Context, taskID, projectMemberID uuid.UUID) (*domain.TaskWatcher, error)
	RemoveWatcher(ctx context.Context, watcherID uuid.UUID) error
	ListWatchers(ctx context.Context, taskID uuid.UUID) ([]domain.TaskWatcher, error)

	AddDependency(ctx context.Context, taskID, dependsOnID uuid.UUID, depType domain.TaskDependencyType) (*domain.TaskDependency, error)
	RemoveDependency(ctx context.Context, dependencyID uuid.UUID) error
	ListDependencies(ctx context.Context, taskID uuid.UUID) ([]domain.TaskDependency, error)
	ListDependants(ctx context.Context, taskID uuid.UUID) ([]domain.TaskDependency, error)

	CreateChecklist(ctx context.Context, taskID uuid.UUID, name string) (*domain.Checklist, error)
	ListChecklists(ctx context.Context, taskID uuid.UUID) ([]domain.Checklist, error)
	CreateChecklistItem(ctx context.Context, checklistID uuid.UUID, content string, order int16) (*domain.ChecklistItem, error)
	ListChecklistItems(ctx context.Context, checklistID uuid.UUID) ([]domain.ChecklistItem, error)
	UpdateChecklistItemStatus(ctx context.Context, itemID uuid.UUID, isChecked bool) (*domain.ChecklistItem, error)
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

func (r *taskRepository) AddWatcher(ctx context.Context, taskID, projectMemberID uuid.UUID) (*domain.TaskWatcher, error) {
	row, err := r.q.AddTaskWatcher(ctx, db.AddTaskWatcherParams{
		TaskID:          taskID,
		ProjectMemberID: projectMemberID,
	})
	if err != nil {
		return nil, err
	}
	w := domain.TaskWatcher{
		ID:              row.ID.String(),
		TaskID:          row.TaskID.String(),
		ProjectMemberID: row.ProjectMemberID.String(),
		CreatedAt:       row.CreatedAt,
	}
	return &w, nil
}

func (r *taskRepository) RemoveWatcher(ctx context.Context, watcherID uuid.UUID) error {
	return r.q.RemoveTaskWatcher(ctx, watcherID)
}

func (r *taskRepository) ListWatchers(ctx context.Context, taskID uuid.UUID) ([]domain.TaskWatcher, error) {
	rows, err := r.q.ListTaskWatchers(ctx, taskID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.TaskWatcher, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.TaskWatcher{
			ID:              row.ID.String(),
			TaskID:          row.TaskID.String(),
			ProjectMemberID: row.ProjectMemberID.String(),
			CreatedAt:       row.CreatedAt,
		})
	}
	return result, nil
}

func (r *taskRepository) AddDependency(ctx context.Context, taskID, dependsOnID uuid.UUID, depType domain.TaskDependencyType) (*domain.TaskDependency, error) {
	row, err := r.q.AddTaskDependency(ctx, db.AddTaskDependencyParams{
		TaskID:          taskID,
		DependsOnTaskID: dependsOnID,
		DependencyType:  string(depType),
	})
	if err != nil {
		return nil, err
	}
	d := domain.TaskDependency{
		ID:              row.ID.String(),
		TaskID:          row.TaskID.String(),
		DependsOnTaskID: row.DependsOnTaskID.String(),
		Type:            domain.TaskDependencyType(row.DependencyType),
		CreatedAt:       row.CreatedAt,
	}
	return &d, nil
}

func (r *taskRepository) RemoveDependency(ctx context.Context, dependencyID uuid.UUID) error {
	return r.q.RemoveTaskDependency(ctx, dependencyID)
}

func (r *taskRepository) ListDependencies(ctx context.Context, taskID uuid.UUID) ([]domain.TaskDependency, error) {
	rows, err := r.q.ListTaskDependencies(ctx, taskID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.TaskDependency, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.TaskDependency{
			ID:              row.ID.String(),
			TaskID:          row.TaskID.String(),
			DependsOnTaskID: row.DependsOnTaskID.String(),
			Type:            domain.TaskDependencyType(row.DependencyType),
			CreatedAt:       row.CreatedAt,
		})
	}
	return result, nil
}

func (r *taskRepository) ListDependants(ctx context.Context, taskID uuid.UUID) ([]domain.TaskDependency, error) {
	rows, err := r.q.ListTaskDependants(ctx, taskID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.TaskDependency, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.TaskDependency{
			ID:              row.ID.String(),
			TaskID:          row.TaskID.String(),
			DependsOnTaskID: row.DependsOnTaskID.String(),
			Type:            domain.TaskDependencyType(row.DependencyType),
			CreatedAt:       row.CreatedAt,
		})
	}
	return result, nil
}

func (r *taskRepository) CreateChecklist(ctx context.Context, taskID uuid.UUID, name string) (*domain.Checklist, error) {
	row, err := r.q.CreateChecklist(ctx, db.CreateChecklistParams{
		TaskID: taskID,
		Name:   name,
	})
	if err != nil {
		return nil, err
	}
	ch := domain.Checklist{
		ID:     row.ID.String(),
		TaskID: row.TaskID.String(),
		Name:   row.Name,
	}
	return &ch, nil
}

func (r *taskRepository) ListChecklists(ctx context.Context, taskID uuid.UUID) ([]domain.Checklist, error) {
	rows, err := r.q.ListTaskChecklists(ctx, taskID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Checklist, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.Checklist{
			ID:     row.ID.String(),
			TaskID: row.TaskID.String(),
			Name:   row.Name,
		})
	}
	return result, nil
}

func (r *taskRepository) CreateChecklistItem(ctx context.Context, checklistID uuid.UUID, content string, order int16) (*domain.ChecklistItem, error) {
	row, err := r.q.CreateChecklistItem(ctx, db.CreateChecklistItemParams{
		ChecklistID: checklistID,
		Content:     content,
		Order:       order,
	})
	if err != nil {
		return nil, err
	}
	item := domain.ChecklistItem{
		ID:          row.ID.String(),
		ChecklistID: row.ChecklistID.String(),
		Content:     row.Content,
		IsChecked:   row.IsChecked,
		Order:       row.Order,
	}
	return &item, nil
}

func (r *taskRepository) ListChecklistItems(ctx context.Context, checklistID uuid.UUID) ([]domain.ChecklistItem, error) {
	rows, err := r.q.ListChecklistItems(ctx, checklistID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.ChecklistItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.ChecklistItem{
			ID:          row.ID.String(),
			ChecklistID: row.ChecklistID.String(),
			Content:     row.Content,
			IsChecked:   row.IsChecked,
			Order:       row.Order,
		})
	}
	return result, nil
}

func (r *taskRepository) UpdateChecklistItemStatus(ctx context.Context, itemID uuid.UUID, isChecked bool) (*domain.ChecklistItem, error) {
	row, err := r.q.UpdateChecklistItemStatus(ctx, db.UpdateChecklistItemStatusParams{
		ID:        itemID,
		IsChecked: isChecked,
	})
	if err != nil {
		return nil, err
	}
	item := domain.ChecklistItem{
		ID:          row.ID.String(),
		ChecklistID: row.ChecklistID.String(),
		Content:     row.Content,
		IsChecked:   row.IsChecked,
		Order:       row.Order,
	}
	return &item, nil
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
