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

type TaskRepository interface {
	Create(ctx context.Context, t *domain.Task) (*domain.Task, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.Task, error)
	Search(ctx context.Context, userID uuid.UUID, projectID, columnID *uuid.UUID) ([]domain.Task, error)
	SearchAll(ctx context.Context, projectID, columnID *uuid.UUID) ([]domain.Task, error)
	Update(ctx context.Context, t *domain.Task) (*domain.Task, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	ListProjectTaskKeys(ctx context.Context, projectID uuid.UUID) ([]string, error)

	// Sprint start: assign columns to backlog tasks
	ListSprintTasksWithoutColumn(ctx context.Context, sprintID uuid.UUID) ([]domain.SprintTaskWithoutColumn, error)
	AssignColumnToTask(ctx context.Context, taskID, columnID uuid.UUID) error
	ClearColumnFromTask(ctx context.Context, taskID uuid.UUID) error

	// Board priority catalog maintenance: вызывается при смене priority_type
	// / priority_options на доске, чтобы синхронизировать tasks.priority с
	// новым каталогом значений.
	ClearPriorityByBoard(ctx context.Context, boardID uuid.UUID) error
	ClearPriorityByBoardNotIn(ctx context.Context, boardID uuid.UUID, allowedOptions []string) error
}

type taskRepository struct {
	q *db.Queries
}

func NewTaskRepository(q *db.Queries) TaskRepository {
	return &taskRepository{q: q}
}

func (r *taskRepository) Create(ctx context.Context, t *domain.Task) (*domain.Task, error) {
	executor := ptrToNullUUID(t.ExecutorID)
	desc := sql.NullString{}
	if t.Description != nil {
		desc = sql.NullString{String: *t.Description, Valid: true}
	}
	var deadline sql.NullTime
	if t.Deadline != nil {
		deadline = sql.NullTime{Time: *t.Deadline, Valid: true}
	}
	columnID := ptrToNullUUID(t.ColumnID)
	swimlane := ptrToNullUUID(t.SwimlaneID)

	var priority sql.NullString
	if t.Priority != nil {
		priority = sql.NullString{String: *t.Priority, Valid: true}
	}
	var estimation sql.NullString
	if t.Estimation != nil {
		estimation = sql.NullString{String: *t.Estimation, Valid: true}
	}

	row, err := r.q.CreateTask(ctx, db.CreateTaskParams{
		Key:         t.Key,
		ProjectID:   t.ProjectID,
		OwnerID:     t.OwnerID,
		ExecutorID:  executor,
		Name:        t.Name,
		Description: desc,
		Deadline:    deadline,
		ColumnID:    columnID,
		SwimlaneID:  swimlane,
		Priority:    priority,
		Estimation:  estimation,
		BoardID:     t.BoardID,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateTask", "key", t.Key, "projectID", t.ProjectID)
	}
	d := mapDBTaskToDomain(row)
	return &d, nil
}

func (r *taskRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	row, err := r.q.GetTaskByID(ctx, id)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetTaskByID", "id", id)
	}
	d := mapTaskRowToDomain(db.Task{
		ID: row.ID, Key: row.Key, ProjectID: row.ProjectID, OwnerID: row.OwnerID,
		ExecutorID: row.ExecutorID, Name: row.Name, Description: row.Description,
		Deadline: row.Deadline, ColumnID: row.ColumnID, SwimlaneID: row.SwimlaneID,
		DeletedAt: row.DeletedAt, CreatedAt: row.CreatedAt, Priority: row.Priority,
		Estimation: row.Estimation, BoardID: row.BoardID,
	}, row.ColumnName, row.ColumnSystemType, row.OwnerUserID, row.ExecutorUserID)
	return &d, nil
}

func (r *taskRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.Task, error) {
	rows, err := r.q.ListProjectTasks(ctx, projectID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListProjectTasks", "projectID", projectID)
	}
	result := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapTaskRowToDomain(db.Task{
			ID: row.ID, Key: row.Key, ProjectID: row.ProjectID, OwnerID: row.OwnerID,
			ExecutorID: row.ExecutorID, Name: row.Name, Description: row.Description,
			Deadline: row.Deadline, ColumnID: row.ColumnID, SwimlaneID: row.SwimlaneID,
			DeletedAt: row.DeletedAt, CreatedAt: row.CreatedAt, Priority: row.Priority,
			Estimation: row.Estimation, BoardID: row.BoardID,
		}, row.ColumnName, row.ColumnSystemType, row.OwnerUserID, row.ExecutorUserID))
	}
	return result, nil
}

func (r *taskRepository) Search(ctx context.Context, userID uuid.UUID, projectID, columnID *uuid.UUID) ([]domain.Task, error) {
	params := db.SearchTasksParams{UserID: userID}
	if projectID != nil {
		params.ProjectID = uuid.NullUUID{UUID: *projectID, Valid: true}
	}
	if columnID != nil {
		params.ColumnID = uuid.NullUUID{UUID: *columnID, Valid: true}
	}
	rows, err := r.q.SearchTasks(ctx, params)
	if err != nil {
		return nil, errctx.Wrap(err, "SearchTasks", "userID", userID)
	}
	result := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapTaskRowToDomain(db.Task{
			ID: row.ID, Key: row.Key, ProjectID: row.ProjectID, OwnerID: row.OwnerID,
			ExecutorID: row.ExecutorID, Name: row.Name, Description: row.Description,
			Deadline: row.Deadline, ColumnID: row.ColumnID, SwimlaneID: row.SwimlaneID,
			DeletedAt: row.DeletedAt, CreatedAt: row.CreatedAt, Priority: row.Priority,
			Estimation: row.Estimation, BoardID: row.BoardID,
		}, row.ColumnName, row.ColumnSystemType, row.OwnerUserID, row.ExecutorUserID))
	}
	return result, nil
}

func (r *taskRepository) SearchAll(ctx context.Context, projectID, columnID *uuid.UUID) ([]domain.Task, error) {
	params := db.SearchTasksAllParams{}
	if projectID != nil {
		params.ProjectID = uuid.NullUUID{UUID: *projectID, Valid: true}
	}
	if columnID != nil {
		params.ColumnID = uuid.NullUUID{UUID: *columnID, Valid: true}
	}
	rows, err := r.q.SearchTasksAll(ctx, params)
	if err != nil {
		return nil, errctx.Wrap(err, "SearchTasksAll")
	}
	result := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapTaskRowToDomain(db.Task{
			ID: row.ID, Key: row.Key, ProjectID: row.ProjectID, OwnerID: row.OwnerID,
			ExecutorID: row.ExecutorID, Name: row.Name, Description: row.Description,
			Deadline: row.Deadline, ColumnID: row.ColumnID, SwimlaneID: row.SwimlaneID,
			DeletedAt: row.DeletedAt, CreatedAt: row.CreatedAt, Priority: row.Priority,
			Estimation: row.Estimation, BoardID: row.BoardID,
		}, row.ColumnName, row.ColumnSystemType, row.OwnerUserID, row.ExecutorUserID))
	}
	return result, nil
}

func (r *taskRepository) Update(ctx context.Context, t *domain.Task) (*domain.Task, error) {
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
	executor := ptrToNullUUID(t.ExecutorID)
	column := ptrToNullUUID(t.ColumnID)
	swimlane := ptrToNullUUID(t.SwimlaneID)
	var priority sql.NullString
	if t.Priority != nil {
		priority = sql.NullString{String: *t.Priority, Valid: true}
	}
	var estimation sql.NullString
	if t.Estimation != nil {
		estimation = sql.NullString{String: *t.Estimation, Valid: true}
	}

	row, err := r.q.UpdateTask(ctx, db.UpdateTaskParams{
		Name:        name,
		Description: desc,
		Deadline:    deadline,
		ExecutorID:  executor,
		ColumnID:    column,
		SwimlaneID:  swimlane,
		Priority:    priority,
		Estimation:  estimation,
		ID:          t.ID,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "UpdateTask", "id", t.ID)
	}
	d := mapDBTaskToDomain(row)
	return &d, nil
}

func (r *taskRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return errctx.Wrap(r.q.SoftDeleteTask(ctx, id), "SoftDeleteTask", "id", id)
}

func (r *taskRepository) ListProjectTaskKeys(ctx context.Context, projectID uuid.UUID) ([]string, error) {
	keys, err := r.q.ListProjectTaskKeys(ctx, projectID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListProjectTaskKeys", "projectID", projectID)
	}
	return keys, nil
}

func mapDBTaskToDomain(t db.Task) domain.Task {
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
	var deletedAt *time.Time
	if t.DeletedAt.Valid {
		dt := t.DeletedAt.Time
		deletedAt = &dt
	}
	var priority *string
	if t.Priority.Valid {
		v := t.Priority.String
		priority = &v
	}
	var estimation *string
	if t.Estimation.Valid {
		v := t.Estimation.String
		estimation = &v
	}
	return domain.Task{
		ID:          t.ID,
		Key:         t.Key,
		ProjectID:   t.ProjectID,
		BoardID:     t.BoardID,
		OwnerID:     t.OwnerID,
		ExecutorID:  nullUUIDToPtr(t.ExecutorID),
		Name:        t.Name,
		Description: desc,
		Deadline:    deadline,
		ColumnID:    nullUUIDToPtr(t.ColumnID),
		SwimlaneID:  nullUUIDToPtr(t.SwimlaneID),
		DeletedAt:   deletedAt,
		CreatedAt:   t.CreatedAt,
		Priority:    priority,
		Estimation:  estimation,
	}
}

func mapTaskRowToDomain(t db.Task, colName, colSystemType sql.NullString, ownerUserID uuid.UUID, executorUserID uuid.NullUUID) domain.Task {
	d := mapDBTaskToDomain(t)
	if colName.Valid {
		d.ColumnName = &colName.String
	}
	if colSystemType.Valid {
		d.ColumnSystemType = &colSystemType.String
	}
	uid := ownerUserID
	d.OwnerUserID = &uid
	if executorUserID.Valid {
		euid := executorUserID.UUID
		d.ExecutorUserID = &euid
	}
	return d
}

func (r *taskRepository) ListSprintTasksWithoutColumn(ctx context.Context, sprintID uuid.UUID) ([]domain.SprintTaskWithoutColumn, error) {
	rows, err := r.q.ListSprintTasksWithoutColumn(ctx, sprintID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListSprintTasksWithoutColumn", "sprintID", sprintID)
	}
	result := make([]domain.SprintTaskWithoutColumn, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.SprintTaskWithoutColumn{
			TaskID:  row.ID,
			BoardID: row.BoardID,
		})
	}
	return result, nil
}

func (r *taskRepository) AssignColumnToTask(ctx context.Context, taskID, columnID uuid.UUID) error {
	return errctx.Wrap(r.q.AssignColumnToTask(ctx, db.AssignColumnToTaskParams{
		ID:       taskID,
		ColumnID: uuid.NullUUID{UUID: columnID, Valid: true},
	}), "AssignColumnToTask", "taskID", taskID, "columnID", columnID)
}

func (r *taskRepository) ClearColumnFromTask(ctx context.Context, taskID uuid.UUID) error {
	return errctx.Wrap(r.q.ClearColumnFromTask(ctx, taskID), "ClearColumnFromTask", "taskID", taskID)
}

func (r *taskRepository) ClearPriorityByBoard(ctx context.Context, boardID uuid.UUID) error {
	return errctx.Wrap(r.q.ClearTaskPriorityByBoard(ctx, boardID), "ClearTaskPriorityByBoard", "boardID", boardID)
}

func (r *taskRepository) ClearPriorityByBoardNotIn(ctx context.Context, boardID uuid.UUID, allowedOptions []string) error {
	return errctx.Wrap(r.q.ClearTaskPriorityByBoardNotIn(ctx, db.ClearTaskPriorityByBoardNotInParams{
		BoardID: boardID,
		Column2: allowedOptions,
	}), "ClearTaskPriorityByBoardNotIn", "boardID", boardID)
}
