package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type TaskService struct {
	taskRepo    repositories.TaskRepository
	projectRepo repositories.ProjectRepository
	tagRepo     repositories.TagRepository
	conn        *sql.DB
}

func NewTaskService(taskRepo repositories.TaskRepository, projectRepo repositories.ProjectRepository, tagRepo repositories.TagRepository, conn *sql.DB) *TaskService {
	return &TaskService{taskRepo: taskRepo, projectRepo: projectRepo, tagRepo: tagRepo, conn: conn}
}

func (s *TaskService) generateTaskKey(ctx context.Context, projectID uuid.UUID) (string, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return "", err
	}
	keys, err := s.taskRepo.ListProjectTaskKeys(ctx, projectID)
	if err != nil {
		return "", err
	}
	maxNum := 0
	prefix := project.Key + "-"
	for _, k := range keys {
		if strings.HasPrefix(k, prefix) {
			rest := strings.TrimPrefix(k, prefix)
			if n, err := strconv.Atoi(rest); err == nil && n > maxNum {
				maxNum = n
			}
		}
	}
	return prefix + strconv.Itoa(maxNum+1), nil
}

func (s *TaskService) CreateTask(ctx context.Context, projectID, ownerMemberID uuid.UUID, name, description string, executorMemberID *uuid.UUID, columnID uuid.UUID, boardID *uuid.UUID, swimlaneID *uuid.UUID, deadline *time.Time, priority *string, estimation *string) (*domain.Task, error) {
	if name == "" {
		return nil, domain.ErrInvalidInput
	}

	key, err := s.generateTaskKey(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var descPtr *string
	if strings.TrimSpace(description) != "" {
		d := strings.TrimSpace(description)
		descPtr = &d
	}
	var execPtr *string
	if executorMemberID != nil {
		id := executorMemberID.String()
		execPtr = &id
	}
	var deadlinePtr *time.Time
	if deadline != nil {
		deadlinePtr = deadline
	}
	var swimlanePtr *string
	if swimlaneID != nil {
		id := swimlaneID.String()
		swimlanePtr = &id
	}

	var boardIDStr string
	if boardID != nil {
		boardIDStr = boardID.String()
	}

	t := &domain.Task{
		ProjectID:   projectID.String(),
		BoardID:     boardIDStr,
		OwnerID:     ownerMemberID.String(),
		ExecutorID:  execPtr,
		Name:        name,
		Description: descPtr,
		Deadline:    deadlinePtr,
		ColumnID:    uuidToStringPtr(columnID),
		SwimlaneID:  swimlanePtr,
		Priority:    priority,
		Estimation:  estimation,
	}
	t.Key = key

	return s.taskRepo.Create(ctx, t)
}

// CreateTaskFullParams содержит все параметры для создания задачи с вложенными сущностями.
type CreateTaskFullParams struct {
	ProjectID        uuid.UUID
	OwnerMemberID    uuid.UUID
	Name             string
	Description      string
	ExecutorMemberID *uuid.UUID
	ColumnID         uuid.UUID
	BoardID          *uuid.UUID
	SwimlaneID       *uuid.UUID
	Deadline         *time.Time
	Priority         *string
	Estimation       *string

	Checklists       []CreateChecklistParam
	Tags             []string
	WatcherMemberIDs []uuid.UUID
	FieldValues      []CreateFieldValueParam
	Dependencies     []CreateDependencyParam
	AddToBacklog     bool
}

type CreateChecklistParam struct {
	Name  string
	Items []CreateChecklistItemParam
}

type CreateChecklistItemParam struct {
	Content   string
	IsChecked bool
	Order     int16
}

type CreateFieldValueParam struct {
	FieldID       uuid.UUID
	ValueText     *string
	ValueNumber   *string
	ValueDatetime *time.Time
}

type CreateDependencyParam struct {
	DependsOnTaskID uuid.UUID
	Type            domain.TaskDependencyType
}

// CreateTaskFull создаёт задачу со всеми вложенными сущностями в одной транзакции.
func (s *TaskService) CreateTaskFull(ctx context.Context, p CreateTaskFullParams) (*domain.Task, error) {
	if p.Name == "" {
		return nil, domain.ErrInvalidInput
	}

	key, err := s.generateTaskKey(ctx, p.ProjectID)
	if err != nil {
		return nil, err
	}

	// Validate dependencies before starting transaction
	for i, dep := range p.Dependencies {
		if dep.DependsOnTaskID == uuid.Nil {
			return nil, fmt.Errorf("dependencies[%d]: %w", i, domain.ErrInvalidInput)
		}
		switch dep.Type {
		case domain.TaskDependencyBlocks, domain.TaskDependencyIsBlockedBy, domain.TaskDependencyRelatesTo, domain.TaskDependencyParent, domain.TaskDependencySubtask:
		default:
			return nil, fmt.Errorf("dependencies[%d]: некорректный тип зависимости %q", i, dep.Type)
		}
	}

	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	txQ := db.New(tx)
	txTaskRepo := repositories.NewTaskRepository(txQ)
	txTagRepo := repositories.NewTagRepository(txQ)
	txBacklogRepo := repositories.NewProductBacklogRepository(txQ)

	// 1. Create the task
	var descPtr *string
	if strings.TrimSpace(p.Description) != "" {
		d := strings.TrimSpace(p.Description)
		descPtr = &d
	}
	var execPtr *string
	if p.ExecutorMemberID != nil {
		id := p.ExecutorMemberID.String()
		execPtr = &id
	}
	var swimlanePtr *string
	if p.SwimlaneID != nil {
		id := p.SwimlaneID.String()
		swimlanePtr = &id
	}
	var boardIDStr string
	if p.BoardID != nil {
		boardIDStr = p.BoardID.String()
	}

	t := &domain.Task{
		Key:         key,
		ProjectID:   p.ProjectID.String(),
		BoardID:     boardIDStr,
		OwnerID:     p.OwnerMemberID.String(),
		ExecutorID:  execPtr,
		Name:        p.Name,
		Description: descPtr,
		Deadline:    p.Deadline,
		ColumnID:    uuidToStringPtr(p.ColumnID),
		SwimlaneID:  swimlanePtr,
		Priority:    p.Priority,
		Estimation:  p.Estimation,
	}

	task, err := txTaskRepo.Create(ctx, t)
	if err != nil {
		return nil, fmt.Errorf("task: %w", err)
	}
	taskID := uuid.MustParse(task.ID)

	// 2. Checklists
	for i, cl := range p.Checklists {
		name := strings.TrimSpace(cl.Name)
		if name == "" {
			return nil, fmt.Errorf("checklists[%d]: имя не может быть пустым", i)
		}
		checklist, err := txTaskRepo.CreateChecklist(ctx, taskID, name)
		if err != nil {
			return nil, fmt.Errorf("checklists[%d]: %w", i, err)
		}
		checklistID := uuid.MustParse(checklist.ID)
		for j, item := range cl.Items {
			content := strings.TrimSpace(item.Content)
			if content == "" {
				return nil, fmt.Errorf("checklists[%d].items[%d]: содержимое не может быть пустым", i, j)
			}
			if _, err := txTaskRepo.CreateChecklistItem(ctx, checklistID, content, item.Order); err != nil {
				return nil, fmt.Errorf("checklists[%d].items[%d]: %w", i, j, err)
			}
		}
	}

	// 3. Tags (find or create on board, then attach)
	if len(p.Tags) > 0 {
		if p.BoardID == nil {
			return nil, fmt.Errorf("tags: невозможно привязать теги без board_id")
		}
		boardID := *p.BoardID
		for i, tagName := range p.Tags {
			tagName = strings.TrimSpace(tagName)
			if tagName == "" {
				continue
			}
			tag, err := txTagRepo.GetByBoardAndName(ctx, boardID, tagName)
			if err != nil && !errors.Is(err, domain.ErrNotFound) {
				return nil, fmt.Errorf("tags[%d]: %w", i, err)
			}
			if tag == nil {
				tag, err = txTagRepo.Create(ctx, boardID, tagName)
				if err != nil {
					return nil, fmt.Errorf("tags[%d]: %w", i, err)
				}
			}
			tagUUID := uuid.MustParse(tag.ID)
			if err := txTagRepo.AddTagToTask(ctx, taskID, tagUUID); err != nil {
				return nil, fmt.Errorf("tags[%d]: %w", i, err)
			}
		}
	}

	// 4. Watchers
	for i, memberID := range p.WatcherMemberIDs {
		if err := txTaskRepo.AddWatcher(ctx, taskID, memberID); err != nil {
			return nil, fmt.Errorf("watcher_member_ids[%d]: %w", i, err)
		}
	}

	// 5. Field values
	for i, fv := range p.FieldValues {
		if err := txTaskRepo.UpsertTaskFieldValue(ctx, taskID, fv.FieldID, fv.ValueText, fv.ValueNumber, fv.ValueDatetime); err != nil {
			return nil, fmt.Errorf("field_values[%d]: %w", i, err)
		}
	}

	// 6. Dependencies (с обратными связями)
	for i, dep := range p.Dependencies {
		if dep.DependsOnTaskID == taskID {
			return nil, fmt.Errorf("dependencies[%d]: задача не может зависеть сама от себя", i)
		}
		if _, err := txTaskRepo.AddDependency(ctx, taskID, dep.DependsOnTaskID, dep.Type); err != nil {
			return nil, fmt.Errorf("dependencies[%d]: %w", i, err)
		}
		// Обратная связь
		if _, err := txTaskRepo.AddDependency(ctx, dep.DependsOnTaskID, taskID, inverseDepType(dep.Type)); err != nil {
			return nil, fmt.Errorf("dependencies[%d] inverse: %w", i, err)
		}
	}

	// 7. Add to product backlog
	if p.AddToBacklog {
		if _, err := txBacklogRepo.Add(ctx, p.ProjectID, taskID, 0); err != nil {
			return nil, fmt.Errorf("add_to_backlog: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskService) GetTask(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	checklists, err := s.ListChecklistsWithItems(ctx, id)
	if err != nil {
		return nil, err
	}
	task.Checklists = checklists
	tags, err := s.tagRepo.ListTaskTags(ctx, id)
	if err != nil {
		return nil, err
	}
	task.Tags = tags
	return task, nil
}

func (s *TaskService) ListProjectTasks(ctx context.Context, projectID uuid.UUID) ([]domain.Task, error) {
	tasks, err := s.taskRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return s.enrichTasksWithTags(ctx, tasks)
}

func (s *TaskService) SearchTasks(ctx context.Context, userID uuid.UUID, projectID, columnID *uuid.UUID) ([]domain.Task, error) {
	tasks, err := s.taskRepo.Search(ctx, userID, projectID, columnID)
	if err != nil {
		return nil, err
	}
	return s.enrichTasksWithTags(ctx, tasks)
}

func (s *TaskService) enrichTasksWithTags(ctx context.Context, tasks []domain.Task) ([]domain.Task, error) {
	if len(tasks) == 0 {
		return tasks, nil
	}
	ids := make([]uuid.UUID, len(tasks))
	for i, t := range tasks {
		ids[i] = uuid.MustParse(t.ID)
	}
	tagMap, err := s.tagRepo.ListTagsByTaskIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		if tags, ok := tagMap[tasks[i].ID]; ok {
			tasks[i].Tags = tags
		}
	}
	return tasks, nil
}

func (s *TaskService) UpdateTask(ctx context.Context, t *domain.Task) (*domain.Task, error) {
	if t.ID == "" {
		return nil, domain.ErrInvalidInput
	}

	// Получаем текущую задачу
	_, err := s.taskRepo.GetByID(ctx, uuid.MustParse(t.ID))
	if err != nil {
		return nil, err
	}

	updated, err := s.taskRepo.Update(ctx, t)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, id uuid.UUID) error {
	return s.taskRepo.SoftDelete(ctx, id)
}

func (s *TaskService) AddWatcher(ctx context.Context, taskID, memberID uuid.UUID) error {
	return s.taskRepo.AddWatcher(ctx, taskID, memberID)
}

func (s *TaskService) RemoveWatcher(ctx context.Context, taskID, memberID uuid.UUID) error {
	return s.taskRepo.RemoveWatcher(ctx, taskID, memberID)
}

func (s *TaskService) ListWatchers(ctx context.Context, taskID uuid.UUID) ([]domain.TaskWatcher, error) {
	return s.taskRepo.ListWatchers(ctx, taskID)
}

// Comments

func (s *TaskService) ListComments(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error) {
	return s.taskRepo.ListComments(ctx, taskID)
}

func (s *TaskService) CreateComment(ctx context.Context, taskID, authorID uuid.UUID, content string, parentCommentID *uuid.UUID) (*domain.Comment, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.taskRepo.CreateComment(ctx, taskID, authorID, content, parentCommentID)
}

func (s *TaskService) GetCommentByID(ctx context.Context, commentID uuid.UUID) (*domain.Comment, error) {
	return s.taskRepo.GetCommentByID(ctx, commentID)
}

func (s *TaskService) DeleteComment(ctx context.Context, commentID uuid.UUID) error {
	return s.taskRepo.DeleteComment(ctx, commentID)
}

// Attachments

func (s *TaskService) ListAttachments(ctx context.Context, taskID uuid.UUID) ([]domain.Attachment, error) {
	return s.taskRepo.ListAttachments(ctx, taskID)
}

func (s *TaskService) CreateAttachment(ctx context.Context, taskID, uploadedBy uuid.UUID, fileName, filePath, contentType string, fileSize int64) (*domain.Attachment, error) {
	return s.taskRepo.CreateAttachment(ctx, taskID, uploadedBy, fileName, filePath, contentType, fileSize)
}

func (s *TaskService) GetAttachmentByID(ctx context.Context, attachmentID uuid.UUID) (*domain.Attachment, error) {
	return s.taskRepo.GetAttachmentByID(ctx, attachmentID)
}

func (s *TaskService) DeleteAttachment(ctx context.Context, attachmentID uuid.UUID) error {
	return s.taskRepo.DeleteAttachment(ctx, attachmentID)
}

// Field values

func (s *TaskService) GetTaskFieldValues(ctx context.Context, taskID uuid.UUID) ([]domain.TaskFieldValue, error) {
	return s.taskRepo.GetTaskFieldValues(ctx, taskID)
}

func (s *TaskService) UpsertTaskFieldValue(ctx context.Context, taskID, fieldID uuid.UUID, valueText, valueNumber *string, valueDatetime *time.Time) error {
	return s.taskRepo.UpsertTaskFieldValue(ctx, taskID, fieldID, valueText, valueNumber, valueDatetime)
}

func (s *TaskService) AddDependency(ctx context.Context, taskID, dependsOnID uuid.UUID, depType domain.TaskDependencyType) (*domain.TaskDependency, error) {
	if taskID == dependsOnID {
		return nil, domain.ErrInvalidInput
	}
	switch depType {
	case domain.TaskDependencyBlocks, domain.TaskDependencyIsBlockedBy, domain.TaskDependencyRelatesTo, domain.TaskDependencyParent, domain.TaskDependencySubtask:
	default:
		return nil, domain.ErrInvalidInput
	}
	// Проверка: между парой задач может быть только одна связь (в любом направлении)
	deps, err := s.taskRepo.ListDependencies(ctx, taskID)
	if err != nil {
		return nil, err
	}
	for _, d := range deps {
		if d.DependsOnTaskID == dependsOnID.String() {
			return nil, domain.ErrConflict
		}
	}
	dependants, err := s.taskRepo.ListDependants(ctx, taskID)
	if err != nil {
		return nil, err
	}
	for _, d := range dependants {
		if d.TaskID == dependsOnID.String() {
			return nil, domain.ErrConflict
		}
	}

	// Обе связи в одной транзакции
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	txRepo := repositories.NewTaskRepository(db.New(tx))

	dep, err := txRepo.AddDependency(ctx, taskID, dependsOnID, depType)
	if err != nil {
		return nil, err
	}

	inverseType := inverseDepType(depType)
	if _, err := txRepo.AddDependency(ctx, dependsOnID, taskID, inverseType); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return dep, nil
}

func inverseDepType(t domain.TaskDependencyType) domain.TaskDependencyType {
	switch t {
	case domain.TaskDependencyBlocks:
		return domain.TaskDependencyIsBlockedBy
	case domain.TaskDependencyIsBlockedBy:
		return domain.TaskDependencyBlocks
	case domain.TaskDependencyParent:
		return domain.TaskDependencySubtask
	case domain.TaskDependencySubtask:
		return domain.TaskDependencyParent
	default:
		return domain.TaskDependencyRelatesTo
	}
}

func (s *TaskService) RemoveDependency(ctx context.Context, dependencyID uuid.UUID) error {
	dep, err := s.taskRepo.GetDependencyByID(ctx, dependencyID)
	if err != nil {
		return err
	}

	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	txRepo := repositories.NewTaskRepository(db.New(tx))

	if err := txRepo.RemoveDependency(ctx, dependencyID); err != nil {
		return err
	}

	taskID, _ := uuid.Parse(dep.TaskID)
	dependsOnID, _ := uuid.Parse(dep.DependsOnTaskID)
	if err := txRepo.RemoveInverseDependency(ctx, dependsOnID, taskID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *TaskService) ListDependencies(ctx context.Context, taskID uuid.UUID) ([]domain.TaskDependency, error) {
	return s.taskRepo.ListDependencies(ctx, taskID)
}

func (s *TaskService) CreateChecklist(ctx context.Context, taskID uuid.UUID, name string) (*domain.Checklist, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.taskRepo.CreateChecklist(ctx, taskID, name)
}

func (s *TaskService) ListChecklistsWithItems(ctx context.Context, taskID uuid.UUID) ([]domain.Checklist, error) {
	checklists, err := s.taskRepo.ListChecklists(ctx, taskID)
	if err != nil {
		return nil, err
	}
	for i := range checklists {
		cid, err := uuid.Parse(checklists[i].ID)
		if err != nil {
			return nil, err
		}
		items, err := s.taskRepo.ListChecklistItems(ctx, cid)
		if err != nil {
			return nil, err
		}
		checklists[i].Items = items
	}
	return checklists, nil
}

func (s *TaskService) AddChecklistItem(ctx context.Context, checklistID uuid.UUID, content string, order int16) (*domain.ChecklistItem, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.taskRepo.CreateChecklistItem(ctx, checklistID, content, order)
}

func (s *TaskService) SetChecklistItemStatus(ctx context.Context, itemID uuid.UUID, isChecked bool) (*domain.ChecklistItem, error) {
	return s.taskRepo.UpdateChecklistItemStatus(ctx, itemID, isChecked)
}

func (s *TaskService) UpdateChecklistName(ctx context.Context, checklistID uuid.UUID, name string) (*domain.Checklist, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.taskRepo.UpdateChecklistName(ctx, checklistID, name)
}

func (s *TaskService) DeleteChecklist(ctx context.Context, checklistID uuid.UUID) error {
	return s.taskRepo.DeleteChecklist(ctx, checklistID)
}

func (s *TaskService) UpdateChecklistItemContent(ctx context.Context, itemID uuid.UUID, content string) (*domain.ChecklistItem, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.taskRepo.UpdateChecklistItemContent(ctx, itemID, content)
}

func (s *TaskService) DeleteChecklistItem(ctx context.Context, itemID uuid.UUID) error {
	return s.taskRepo.DeleteChecklistItem(ctx, itemID)
}

func uuidToStringPtr(id uuid.UUID) *string {
	if id == uuid.Nil {
		return nil
	}
	s := id.String()
	return &s
}

