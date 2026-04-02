package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type TaskRepository interface {
	Create(ctx context.Context, t *domain.Task) (*domain.Task, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.Task, error)
	Search(ctx context.Context, userID uuid.UUID, projectID, columnID *uuid.UUID) ([]domain.Task, error)
	Update(ctx context.Context, t *domain.Task) (*domain.Task, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	ListProjectTaskKeys(ctx context.Context, projectID uuid.UUID) ([]string, error)

	AddWatcher(ctx context.Context, taskID, memberID uuid.UUID) error
	RemoveWatcher(ctx context.Context, taskID, memberID uuid.UUID) error
	ListWatchers(ctx context.Context, taskID uuid.UUID) ([]domain.TaskWatcher, error)

	// Comments
	ListComments(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error)
	CreateComment(ctx context.Context, taskID, authorID uuid.UUID, content string, parentCommentID *uuid.UUID) (*domain.Comment, error)
	GetCommentByID(ctx context.Context, commentID uuid.UUID) (*domain.Comment, error)
	DeleteComment(ctx context.Context, commentID uuid.UUID) error

	// Attachments
	ListAttachments(ctx context.Context, taskID uuid.UUID) ([]domain.Attachment, error)
	CreateAttachment(ctx context.Context, taskID, uploadedBy uuid.UUID, fileName, filePath, contentType string, fileSize int64) (*domain.Attachment, error)
	GetAttachmentByID(ctx context.Context, attachmentID uuid.UUID) (*domain.Attachment, error)
	DeleteAttachment(ctx context.Context, attachmentID uuid.UUID) error

	// Field values
	GetTaskFieldValues(ctx context.Context, taskID uuid.UUID) ([]domain.TaskFieldValue, error)
	UpsertTaskFieldValue(ctx context.Context, taskID, fieldID uuid.UUID, valueText, valueNumber *string, valueDatetime *time.Time) error

	// Sprint start: assign columns to backlog tasks
	ListSprintTasksWithoutColumn(ctx context.Context, sprintID uuid.UUID) ([]domain.SprintTaskWithoutColumn, error)
	AssignColumnToTask(ctx context.Context, taskID, columnID uuid.UUID) error
	ClearColumnFromTask(ctx context.Context, taskID uuid.UUID) error

	AddDependency(ctx context.Context, taskID, dependsOnID uuid.UUID, depType domain.TaskDependencyType) (*domain.TaskDependency, error)
	GetDependencyByID(ctx context.Context, id uuid.UUID) (*domain.TaskDependency, error)
	RemoveDependency(ctx context.Context, dependencyID uuid.UUID) error
	RemoveInverseDependency(ctx context.Context, taskID, dependsOnTaskID uuid.UUID) error
	ListDependencies(ctx context.Context, taskID uuid.UUID) ([]domain.TaskDependency, error)
	ListDependants(ctx context.Context, taskID uuid.UUID) ([]domain.TaskDependency, error)

	CreateChecklist(ctx context.Context, taskID uuid.UUID, name string) (*domain.Checklist, error)
	UpdateChecklistName(ctx context.Context, checklistID uuid.UUID, name string) (*domain.Checklist, error)
	DeleteChecklist(ctx context.Context, checklistID uuid.UUID) error
	ListChecklists(ctx context.Context, taskID uuid.UUID) ([]domain.Checklist, error)
	CreateChecklistItem(ctx context.Context, checklistID uuid.UUID, content string, order int16) (*domain.ChecklistItem, error)
	ListChecklistItems(ctx context.Context, checklistID uuid.UUID) ([]domain.ChecklistItem, error)
	UpdateChecklistItemStatus(ctx context.Context, itemID uuid.UUID, isChecked bool) (*domain.ChecklistItem, error)
	UpdateChecklistItemContent(ctx context.Context, itemID uuid.UUID, content string) (*domain.ChecklistItem, error)
	DeleteChecklistItem(ctx context.Context, itemID uuid.UUID) error
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
	var columnID uuid.NullUUID
	if t.ColumnID != nil {
		if cid, err := uuid.Parse(*t.ColumnID); err == nil {
			columnID = uuid.NullUUID{UUID: cid, Valid: true}
		}
	}
	var swimlane uuid.NullUUID
	if t.SwimlaneID != nil {
		if id, err := uuid.Parse(*t.SwimlaneID); err == nil {
			swimlane = uuid.NullUUID{UUID: id, Valid: true}
		}
	}

	var priority sql.NullString
	if t.Priority != nil {
		priority = sql.NullString{String: *t.Priority, Valid: true}
	}
	var estimation sql.NullString
	if t.Estimation != nil {
		estimation = sql.NullString{String: *t.Estimation, Valid: true}
	}

	boardID, err := uuid.Parse(t.BoardID)
	if err != nil {
		return nil, err
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
		Priority:    priority,
		Estimation:  estimation,
		BoardID:     boardID,
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
		return nil, err
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
		return nil, err
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
	if t.ColumnID != nil {
		if cid, err := uuid.Parse(*t.ColumnID); err == nil {
			column = uuid.NullUUID{UUID: cid, Valid: true}
		}
	}
	var swimlane uuid.NullUUID
	if t.SwimlaneID != nil {
		if sid, err := uuid.Parse(*t.SwimlaneID); err == nil {
			swimlane = uuid.NullUUID{UUID: sid, Valid: true}
		}
	}
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
		ID:          id,
	})
	if err != nil {
		return nil, err
	}
	d := mapDBTaskToDomain(row)
	return &d, nil
}

func (r *taskRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.q.SoftDeleteTask(ctx, id)
}

func (r *taskRepository) ListProjectTaskKeys(ctx context.Context, projectID uuid.UUID) ([]string, error) {
	return r.q.ListProjectTaskKeys(ctx, projectID)
}

func (r *taskRepository) AddWatcher(ctx context.Context, taskID, memberID uuid.UUID) error {
	return r.q.AddTaskWatcher(ctx, db.AddTaskWatcherParams{
		TaskID:   taskID,
		MemberID: memberID,
	})
}

func (r *taskRepository) RemoveWatcher(ctx context.Context, taskID, memberID uuid.UUID) error {
	return r.q.RemoveTaskWatcher(ctx, db.RemoveTaskWatcherParams{
		TaskID:   taskID,
		MemberID: memberID,
	})
}

func (r *taskRepository) ListWatchers(ctx context.Context, taskID uuid.UUID) ([]domain.TaskWatcher, error) {
	rows, err := r.q.ListTaskWatchers(ctx, taskID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.TaskWatcher, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.TaskWatcher{
			TaskID:   row.TaskID.String(),
			MemberID: row.MemberID.String(),
		})
	}
	return result, nil
}

// Comments

func (r *taskRepository) ListComments(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error) {
	rows, err := r.q.ListTaskComments(ctx, taskID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Comment, 0, len(rows))
	for _, row := range rows {
		c := domain.Comment{
			ID:        row.ID.String(),
			TaskID:    row.TaskID.String(),
			AuthorID:  row.AuthorID.String(),
			Content:   row.Content,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
		if row.ParentCommentID.Valid {
			s := row.ParentCommentID.UUID.String()
			c.ParentCommentID = &s
		}
		result = append(result, c)
	}
	return result, nil
}

func (r *taskRepository) CreateComment(ctx context.Context, taskID, authorID uuid.UUID, content string, parentCommentID *uuid.UUID) (*domain.Comment, error) {
	var parentID uuid.NullUUID
	if parentCommentID != nil {
		parentID = uuid.NullUUID{UUID: *parentCommentID, Valid: true}
	}
	row, err := r.q.CreateComment(ctx, db.CreateCommentParams{
		TaskID:          taskID,
		AuthorID:        authorID,
		Content:         content,
		ParentCommentID: parentID,
	})
	if err != nil {
		return nil, err
	}
	c := &domain.Comment{
		ID:        row.ID.String(),
		TaskID:    row.TaskID.String(),
		AuthorID:  row.AuthorID.String(),
		Content:   row.Content,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.ParentCommentID.Valid {
		s := row.ParentCommentID.UUID.String()
		c.ParentCommentID = &s
	}
	return c, nil
}

func (r *taskRepository) GetCommentByID(ctx context.Context, commentID uuid.UUID) (*domain.Comment, error) {
	row, err := r.q.GetCommentByID(ctx, commentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	c := &domain.Comment{
		ID:        row.ID.String(),
		TaskID:    row.TaskID.String(),
		AuthorID:  row.AuthorID.String(),
		Content:   row.Content,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.ParentCommentID.Valid {
		s := row.ParentCommentID.UUID.String()
		c.ParentCommentID = &s
	}
	return c, nil
}

func (r *taskRepository) DeleteComment(ctx context.Context, commentID uuid.UUID) error {
	return r.q.DeleteComment(ctx, commentID)
}

// Attachments

func (r *taskRepository) ListAttachments(ctx context.Context, taskID uuid.UUID) ([]domain.Attachment, error) {
	rows, err := r.q.ListTaskAttachments(ctx, uuid.NullUUID{UUID: taskID, Valid: true})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Attachment, 0, len(rows))
	for _, row := range rows {
		a := domain.Attachment{
			ID:          row.ID.String(),
			FileName:    row.FileName,
			FilePath:    row.FilePath,
			FileSize:    row.FileSize,
			ContentType: row.ContentType,
			UploadedBy:  row.UploadedBy.String(),
			UploadedAt:  row.UploadedAt,
		}
		if row.TaskID.Valid {
			id := row.TaskID.UUID.String()
			a.TaskID = &id
		}
		if row.CommentID.Valid {
			id := row.CommentID.UUID.String()
			a.CommentID = &id
		}
		result = append(result, a)
	}
	return result, nil
}

func (r *taskRepository) CreateAttachment(ctx context.Context, taskID, uploadedBy uuid.UUID, fileName, filePath, contentType string, fileSize int64) (*domain.Attachment, error) {
	row, err := r.q.CreateAttachment(ctx, db.CreateAttachmentParams{
		TaskID:      uuid.NullUUID{UUID: taskID, Valid: true},
		CommentID:   uuid.NullUUID{},
		FileName:    fileName,
		FilePath:    filePath,
		FileSize:    fileSize,
		ContentType: contentType,
		UploadedBy:  uploadedBy,
	})
	if err != nil {
		return nil, err
	}
	a := &domain.Attachment{
		ID:          row.ID.String(),
		FileName:    row.FileName,
		FilePath:    row.FilePath,
		FileSize:    row.FileSize,
		ContentType: row.ContentType,
		UploadedBy:  row.UploadedBy.String(),
		UploadedAt:  row.UploadedAt,
	}
	if row.TaskID.Valid {
		id := row.TaskID.UUID.String()
		a.TaskID = &id
	}
	return a, nil
}

func (r *taskRepository) GetAttachmentByID(ctx context.Context, attachmentID uuid.UUID) (*domain.Attachment, error) {
	row, err := r.q.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	a := &domain.Attachment{
		ID:          row.ID.String(),
		FileName:    row.FileName,
		FilePath:    row.FilePath,
		FileSize:    row.FileSize,
		ContentType: row.ContentType,
		UploadedBy:  row.UploadedBy.String(),
		UploadedAt:  row.UploadedAt,
	}
	if row.TaskID.Valid {
		id := row.TaskID.UUID.String()
		a.TaskID = &id
	}
	if row.CommentID.Valid {
		id := row.CommentID.UUID.String()
		a.CommentID = &id
	}
	return a, nil
}

func (r *taskRepository) DeleteAttachment(ctx context.Context, attachmentID uuid.UUID) error {
	return r.q.DeleteAttachment(ctx, attachmentID)
}

// Field values

func (r *taskRepository) GetTaskFieldValues(ctx context.Context, taskID uuid.UUID) ([]domain.TaskFieldValue, error) {
	rows, err := r.q.GetTaskFieldValues(ctx, taskID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.TaskFieldValue, 0, len(rows))
	for _, row := range rows {
		fv := domain.TaskFieldValue{
			TaskID:  row.TaskID.String(),
			FieldID: row.FieldID.String(),
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

func (r *taskRepository) UpsertTaskFieldValue(ctx context.Context, taskID, fieldID uuid.UUID, valueText, valueNumber *string, valueDatetime *time.Time) error {
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
	return r.q.UpsertTaskFieldValue(ctx, params)
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
	}
	return &d, nil
}

func (r *taskRepository) GetDependencyByID(ctx context.Context, id uuid.UUID) (*domain.TaskDependency, error) {
	row, err := r.q.GetTaskDependencyByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &domain.TaskDependency{
		ID:              row.ID.String(),
		TaskID:          row.TaskID.String(),
		DependsOnTaskID: row.DependsOnTaskID.String(),
		Type:            domain.TaskDependencyType(row.DependencyType),
	}, nil
}

func (r *taskRepository) RemoveDependency(ctx context.Context, dependencyID uuid.UUID) error {
	return r.q.RemoveTaskDependency(ctx, dependencyID)
}

func (r *taskRepository) RemoveInverseDependency(ctx context.Context, taskID, dependsOnTaskID uuid.UUID) error {
	return r.q.RemoveInverseDependency(ctx, db.RemoveInverseDependencyParams{
		TaskID:          taskID,
		DependsOnTaskID: dependsOnTaskID,
	})
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
	return &domain.Checklist{
		ID:     row.ID.String(),
		TaskID: row.TaskID.String(),
		Name:   row.Name,
	}, nil
}

func (r *taskRepository) ListChecklists(ctx context.Context, taskID uuid.UUID) ([]domain.Checklist, error) {
	rows, err := r.q.ListChecklistsByTask(ctx, taskID)
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
		IsChecked:   false,
		SortOrder:   order,
	})
	if err != nil {
		return nil, err
	}
	return &domain.ChecklistItem{
		ID:          row.ID.String(),
		ChecklistID: row.ChecklistID.String(),
		Content:     row.Content,
		IsChecked:   row.IsChecked,
		Order:       row.SortOrder,
	}, nil
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
			Order:       row.SortOrder,
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
	return &domain.ChecklistItem{
		ID:          row.ID.String(),
		ChecklistID: row.ChecklistID.String(),
		Content:     row.Content,
		IsChecked:   row.IsChecked,
		Order:       row.SortOrder,
	}, nil
}

func (r *taskRepository) UpdateChecklistName(ctx context.Context, checklistID uuid.UUID, name string) (*domain.Checklist, error) {
	row, err := r.q.UpdateChecklistName(ctx, db.UpdateChecklistNameParams{ID: checklistID, Name: name})
	if err != nil {
		return nil, err
	}
	return &domain.Checklist{ID: row.ID.String(), TaskID: row.TaskID.String(), Name: row.Name}, nil
}

func (r *taskRepository) DeleteChecklist(ctx context.Context, checklistID uuid.UUID) error {
	return r.q.DeleteChecklist(ctx, checklistID)
}

func (r *taskRepository) UpdateChecklistItemContent(ctx context.Context, itemID uuid.UUID, content string) (*domain.ChecklistItem, error) {
	row, err := r.q.UpdateChecklistItemContent(ctx, db.UpdateChecklistItemContentParams{ID: itemID, Content: content})
	if err != nil {
		return nil, err
	}
	return &domain.ChecklistItem{
		ID: row.ID.String(), ChecklistID: row.ChecklistID.String(),
		Content: row.Content, IsChecked: row.IsChecked, Order: row.SortOrder,
	}, nil
}

func (r *taskRepository) DeleteChecklistItem(ctx context.Context, itemID uuid.UUID) error {
	return r.q.DeleteChecklistItem(ctx, itemID)
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
		ID:          t.ID.String(),
		Key:         t.Key,
		ProjectID:   t.ProjectID.String(),
		BoardID:     t.BoardID.String(),
		OwnerID:     t.OwnerID.String(),
		ExecutorID:  exec,
		Name:        t.Name,
		Description: desc,
		Deadline:    deadline,
		ColumnID:    nullUUIDToStringPtr(t.ColumnID),
		SwimlaneID:  swimlane,
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
	uid := ownerUserID.String()
	d.OwnerUserID = &uid
	if executorUserID.Valid {
		euid := executorUserID.UUID.String()
		d.ExecutorUserID = &euid
	}
	return d
}

func nullUUIDToStringPtr(n uuid.NullUUID) *string {
	if !n.Valid {
		return nil
	}
	s := n.UUID.String()
	return &s
}

func (r *taskRepository) ListSprintTasksWithoutColumn(ctx context.Context, sprintID uuid.UUID) ([]domain.SprintTaskWithoutColumn, error) {
	rows, err := r.q.ListSprintTasksWithoutColumn(ctx, sprintID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.SprintTaskWithoutColumn, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.SprintTaskWithoutColumn{
			TaskID:  row.ID.String(),
			BoardID: row.BoardID.String(),
		})
	}
	return result, nil
}

func (r *taskRepository) AssignColumnToTask(ctx context.Context, taskID, columnID uuid.UUID) error {
	return r.q.AssignColumnToTask(ctx, db.AssignColumnToTaskParams{
		ID:       taskID,
		ColumnID: uuid.NullUUID{UUID: columnID, Valid: true},
	})
}

func (r *taskRepository) ClearColumnFromTask(ctx context.Context, taskID uuid.UUID) error {
	return r.q.ClearColumnFromTask(ctx, taskID)
}
