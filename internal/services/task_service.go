package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type TaskService struct {
	taskRepo        repositories.TaskRepository
	projectRepo     repositories.ProjectRepository
	tagRepo         repositories.TagRepository
	commentRepo     repositories.CommentRepository
	attachmentRepo  repositories.AttachmentRepository
	checklistRepo   repositories.ChecklistRepository
	dependencyRepo  repositories.TaskDependencyRepository
	watcherRepo     repositories.TaskWatcherRepository
	fieldValueRepo  repositories.TaskFieldValueRepository
	tagSvc          *TagService
	queries         *db.Queries
	conn            *sql.DB
	notificationSvc NotificationService
}

func NewTaskService(
	taskRepo repositories.TaskRepository,
	projectRepo repositories.ProjectRepository,
	tagRepo repositories.TagRepository,
	commentRepo repositories.CommentRepository,
	attachmentRepo repositories.AttachmentRepository,
	checklistRepo repositories.ChecklistRepository,
	dependencyRepo repositories.TaskDependencyRepository,
	watcherRepo repositories.TaskWatcherRepository,
	fieldValueRepo repositories.TaskFieldValueRepository,
	tagSvc *TagService,
	conn *sql.DB,
	queries *db.Queries,
	notificationSvc NotificationService,
) *TaskService {
	return &TaskService{
		taskRepo:        taskRepo,
		projectRepo:     projectRepo,
		tagRepo:         tagRepo,
		commentRepo:     commentRepo,
		attachmentRepo:  attachmentRepo,
		checklistRepo:   checklistRepo,
		dependencyRepo:  dependencyRepo,
		watcherRepo:     watcherRepo,
		fieldValueRepo:  fieldValueRepo,
		tagSvc:          tagSvc,
		conn:            conn,
		queries:         queries,
		notificationSvc: notificationSvc,
	}
}

// estimationPattern задаёт единый формат оценки трудозатрат — неотрицательное
// число с опциональной десятичной частью (до 2 знаков). Интерпретация единиц
// (story points или часы) определяется настройкой доски `estimation_unit`.
var estimationPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]{1,2})?$`)

// validateEstimation возвращает ErrInvalidEstimation, если переданная оценка
// задана, но не соответствует числовому формату. Пустая строка нормализуется
// в nil, чтобы не тащить в БД "" как валидное значение.
func validateEstimation(est *string) (*string, error) {
	if est == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*est)
	if trimmed == "" {
		return nil, nil
	}
	if !estimationPattern.MatchString(trimmed) {
		return nil, domain.ErrInvalidEstimation
	}
	return &trimmed, nil
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

	estimation, err := validateEstimation(estimation)
	if err != nil {
		return nil, err
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
	var deadlinePtr *time.Time
	if deadline != nil {
		deadlinePtr = deadline
	}

	var boardIDVal uuid.UUID
	if boardID != nil {
		boardIDVal = *boardID
	}

	var colPtr *uuid.UUID
	if columnID != uuid.Nil {
		cid := columnID
		colPtr = &cid
	}

	t := &domain.Task{
		ProjectID:   projectID,
		BoardID:     boardIDVal,
		OwnerID:     ownerMemberID,
		ExecutorID:  executorMemberID,
		Name:        name,
		Description: descPtr,
		Deadline:    deadlinePtr,
		ColumnID:    colPtr,
		SwimlaneID:  swimlaneID,
		Priority:    priority,
		Estimation:  estimation,
	}
	t.Key = key

	created, err := s.taskRepo.Create(ctx, t)
	if err != nil {
		return nil, err
	}

	// Записываем начальную запись в историю
	if columnID != uuid.Nil {
		s.recordColumnChange(ctx, created.ID, columnID)
	}

	// Уведомление при назначении исполнителя
	if executorMemberID != nil {
		ownerMember, _ := s.queries.GetProjectMember(ctx, ownerMemberID)
		actorUID := ""
		if ownerMember.UserID != uuid.Nil {
			actorUID = ownerMember.UserID.String()
		}
		s.notifyTaskAssigned(ctx, created, *executorMemberID, actorUID)
	}

	return created, nil
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

	est, err := validateEstimation(p.Estimation)
	if err != nil {
		return nil, err
	}
	p.Estimation = est

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

	// Всё создание задачи со вложенными сущностями (чек-листы, теги, наблюдатели,
	// field values, зависимости, backlog) — одной транзакцией. Если любой шаг
	// упал, откатываем всё и наружу возвращаем исходную ошибку с контекстом.
	task, err := repositories.InTxT(ctx, s.conn, func(qtx *db.Queries) (*domain.Task, error) {
		txTaskRepo := repositories.NewTaskRepository(qtx)
		txTagRepo := repositories.NewTagRepository(qtx)
		txBacklogRepo := repositories.NewProductBacklogRepository(qtx)
		txChecklistRepo := repositories.NewChecklistRepository(qtx)
		txDependencyRepo := repositories.NewTaskDependencyRepository(qtx)
		txWatcherRepo := repositories.NewTaskWatcherRepository(qtx)
		txFieldValueRepo := repositories.NewTaskFieldValueRepository(qtx)

	// 1. Create the task
	var descPtr *string
	if strings.TrimSpace(p.Description) != "" {
		d := strings.TrimSpace(p.Description)
		descPtr = &d
	}
	var boardIDVal uuid.UUID
	if p.BoardID != nil {
		boardIDVal = *p.BoardID
	}
	var colPtr *uuid.UUID
	if p.ColumnID != uuid.Nil {
		cid := p.ColumnID
		colPtr = &cid
	}

	t := &domain.Task{
		Key:         key,
		ProjectID:   p.ProjectID,
		BoardID:     boardIDVal,
		OwnerID:     p.OwnerMemberID,
		ExecutorID:  p.ExecutorMemberID,
		Name:        p.Name,
		Description: descPtr,
		Deadline:    p.Deadline,
		ColumnID:    colPtr,
		SwimlaneID:  p.SwimlaneID,
		Priority:    p.Priority,
		Estimation:  p.Estimation,
	}

	task, err := txTaskRepo.Create(ctx, t)
	if err != nil {
		return nil, fmt.Errorf("task: %w", err)
	}
	taskID := task.ID

	// Записываем начальную запись в историю
	if p.ColumnID != uuid.Nil {
		s.recordColumnChange(ctx, taskID, p.ColumnID)
	}

	// 2. Checklists
	for i, cl := range p.Checklists {
		name := strings.TrimSpace(cl.Name)
		if name == "" {
			return nil, fmt.Errorf("checklists[%d]: имя не может быть пустым", i)
		}
		checklist, err := txChecklistRepo.Create(ctx, taskID, name)
		if err != nil {
			return nil, fmt.Errorf("checklists[%d]: %w", i, err)
		}
		checklistID := checklist.ID
		for j, item := range cl.Items {
			content := strings.TrimSpace(item.Content)
			if content == "" {
				return nil, fmt.Errorf("checklists[%d].items[%d]: содержимое не может быть пустым", i, j)
			}
			if _, err := txChecklistRepo.CreateItem(ctx, checklistID, content, item.Order); err != nil {
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
			if err := txTagRepo.AddTagToTask(ctx, taskID, tag.ID); err != nil {
				return nil, fmt.Errorf("tags[%d]: %w", i, err)
			}
		}
	}

	// 4. Watchers
	for i, memberID := range p.WatcherMemberIDs {
		if err := txWatcherRepo.Add(ctx, taskID, memberID); err != nil {
			return nil, fmt.Errorf("watcher_member_ids[%d]: %w", i, err)
		}
	}

	// 5. Field values
	for i, fv := range p.FieldValues {
		if err := txFieldValueRepo.Upsert(ctx, taskID, fv.FieldID, fv.ValueText, fv.ValueNumber, fv.ValueDatetime); err != nil {
			return nil, fmt.Errorf("field_values[%d]: %w", i, err)
		}
	}

	// 6. Dependencies (с обратными связями)
	for i, dep := range p.Dependencies {
		if dep.DependsOnTaskID == taskID {
			return nil, fmt.Errorf("dependencies[%d]: задача не может зависеть сама от себя", i)
		}
		if _, err := txDependencyRepo.Add(ctx, taskID, dep.DependsOnTaskID, dep.Type); err != nil {
			return nil, fmt.Errorf("dependencies[%d]: %w", i, err)
		}
		// Обратная связь
		if _, err := txDependencyRepo.Add(ctx, dep.DependsOnTaskID, taskID, inverseDepType(dep.Type)); err != nil {
			return nil, fmt.Errorf("dependencies[%d] inverse: %w", i, err)
		}
	}

	// 7. Add to product backlog
	if p.AddToBacklog {
		if _, err := txBacklogRepo.Add(ctx, p.ProjectID, taskID, 0); err != nil {
			return nil, fmt.Errorf("add_to_backlog: %w", err)
		}
	}

		return task, nil
	})
	if err != nil {
		return nil, err
	}

	// Уведомление при назначении исполнителя
	if p.ExecutorMemberID != nil {
		ownerMember, _ := s.queries.GetProjectMember(ctx, p.OwnerMemberID)
		actorUID := ""
		if ownerMember.UserID != uuid.Nil {
			actorUID = ownerMember.UserID.String()
		}
		s.notifyTaskAssigned(ctx, task, *p.ExecutorMemberID, actorUID)
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

// SearchAllTasks возвращает задачи без фильтра по участию пользователя.
// Используется для пользователей с system.projects.manage ∈ {full, view}.
func (s *TaskService) SearchAllTasks(ctx context.Context, projectID, columnID *uuid.UUID) ([]domain.Task, error) {
	tasks, err := s.taskRepo.SearchAll(ctx, projectID, columnID)
	if err != nil {
		return nil, err
	}
	return s.enrichTasksWithTags(ctx, tasks)
}

// enrichTasksWithTags — thin wrapper над TagService.EnrichTasksWithTags,
// чтобы не плодить прямое обращение к tagSvc во всех методах поиска.
func (s *TaskService) enrichTasksWithTags(ctx context.Context, tasks []domain.Task) ([]domain.Task, error) {
	return s.tagSvc.EnrichTasksWithTags(ctx, tasks)
}

func (s *TaskService) UpdateTask(ctx context.Context, t *domain.Task, actorUserID string) (*domain.Task, error) {
	if t.ID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	est, err := validateEstimation(t.Estimation)
	if err != nil {
		return nil, err
	}
	t.Estimation = est

	taskID := t.ID

	// Получаем текущую задачу для сравнения column_id и executor_id.
	// existing имеет OwnerUserID/ExecutorUserID (из JOIN), а updated — нет.
	existing, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	updated, err := s.taskRepo.Update(ctx, t)
	if err != nil {
		return nil, err
	}

	// Записываем историю при смене колонки
	var oldCol uuid.UUID
	if existing.ColumnID != nil {
		oldCol = *existing.ColumnID
	}
	var newCol uuid.UUID
	if updated.ColumnID != nil {
		newCol = *updated.ColumnID
	}
	columnChanged := newCol != uuid.Nil && newCol != oldCol
	if columnChanged {
		s.recordColumnChange(ctx, taskID, newCol)
	}

	// Уведомление при смене исполнителя
	var oldExec uuid.UUID
	if existing.ExecutorID != nil {
		oldExec = *existing.ExecutorID
	}
	var newExec uuid.UUID
	if updated.ExecutorID != nil {
		newExec = *updated.ExecutorID
	}
	if newExec != uuid.Nil && newExec != oldExec {
		s.notifyTaskAssigned(ctx, existing, newExec, actorUserID)
	}

	// Уведомления при смене колонки (статуса) — используем existing для user IDs
	if columnChanged {
		s.notifyStatusChange(ctx, existing, newCol, actorUserID)
	}

	return updated, nil
}

// recordColumnChange закрывает текущую запись в task_status_history и создаёт новую
func (s *TaskService) recordColumnChange(ctx context.Context, taskID, newColumnID uuid.UUID) {
	now := time.Now()
	// Закрываем предыдущую открытую запись
	_ = s.queries.CloseTaskStatusHistory(ctx, db.CloseTaskStatusHistoryParams{
		TaskID: taskID,
		LeftAt: sql.NullTime{Time: now, Valid: true},
	})
	// Создаём новую запись
	_, _ = s.queries.RecordTaskStatusChange(ctx, db.RecordTaskStatusChangeParams{
		TaskID:    taskID,
		ColumnID:  newColumnID,
		EnteredAt: now,
	})
}

func (s *TaskService) DeleteTask(ctx context.Context, id uuid.UUID) error {
	// Soft delete не запускает ON DELETE CASCADE, поэтому парные строки в
	// task_dependencies нужно подчистить руками — иначе на другой стороне
	// связи в GET /tasks/{other}/dependencies останутся «висящие» ссылки на
	// удалённую задачу.
	return repositories.InTx(ctx, s.conn, func(qtx *db.Queries) error {
		depRepo := repositories.NewTaskDependencyRepository(qtx)
		if err := depRepo.RemoveAllForTask(ctx, id); err != nil {
			return err
		}
		taskRepo := repositories.NewTaskRepository(qtx)
		return taskRepo.SoftDelete(ctx, id)
	})
}

func (s *TaskService) AddWatcher(ctx context.Context, taskID, memberID uuid.UUID) error {
	return s.watcherRepo.Add(ctx, taskID, memberID)
}

func (s *TaskService) RemoveWatcher(ctx context.Context, taskID, memberID uuid.UUID) error {
	return s.watcherRepo.Remove(ctx, taskID, memberID)
}

func (s *TaskService) ListWatchers(ctx context.Context, taskID uuid.UUID) ([]domain.TaskWatcher, error) {
	return s.watcherRepo.List(ctx, taskID)
}

// Comments

func (s *TaskService) ListComments(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error) {
	return s.commentRepo.List(ctx, taskID)
}

func (s *TaskService) CreateComment(ctx context.Context, taskID, authorID uuid.UUID, content string, parentCommentID *uuid.UUID) (*domain.Comment, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, domain.ErrInvalidInput
	}
	comment, err := s.commentRepo.Create(ctx, taskID, authorID, content, parentCommentID)
	if err != nil {
		return nil, err
	}

	// Уведомления об @-упоминаниях
	s.notifyMentions(ctx, taskID, authorID, content)

	return comment, nil
}

func (s *TaskService) GetCommentByID(ctx context.Context, commentID uuid.UUID) (*domain.Comment, error) {
	return s.commentRepo.GetByID(ctx, commentID)
}

func (s *TaskService) DeleteComment(ctx context.Context, commentID uuid.UUID) error {
	return s.commentRepo.Delete(ctx, commentID)
}

// Attachments

func (s *TaskService) ListAttachments(ctx context.Context, taskID uuid.UUID) ([]domain.Attachment, error) {
	return s.attachmentRepo.List(ctx, taskID)
}

func (s *TaskService) CreateAttachment(ctx context.Context, taskID, uploadedBy uuid.UUID, fileName, filePath, contentType string, fileSize int64) (*domain.Attachment, error) {
	return s.attachmentRepo.Create(ctx, taskID, uploadedBy, fileName, filePath, contentType, fileSize)
}

func (s *TaskService) GetAttachmentByID(ctx context.Context, attachmentID uuid.UUID) (*domain.Attachment, error) {
	return s.attachmentRepo.GetByID(ctx, attachmentID)
}

func (s *TaskService) DeleteAttachment(ctx context.Context, attachmentID uuid.UUID) error {
	return s.attachmentRepo.Delete(ctx, attachmentID)
}

// Field values

func (s *TaskService) GetTaskFieldValues(ctx context.Context, taskID uuid.UUID) ([]domain.TaskFieldValue, error) {
	return s.fieldValueRepo.ListByTask(ctx, taskID)
}

func (s *TaskService) UpsertTaskFieldValue(ctx context.Context, taskID, fieldID uuid.UUID, valueText, valueNumber *string, valueDatetime *time.Time) error {
	return s.fieldValueRepo.Upsert(ctx, taskID, fieldID, valueText, valueNumber, valueDatetime)
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
	deps, err := s.dependencyRepo.ListFor(ctx, taskID)
	if err != nil {
		return nil, err
	}
	for _, d := range deps {
		if d.DependsOnTaskID == dependsOnID {
			return nil, domain.ErrConflict
		}
	}
	dependants, err := s.dependencyRepo.ListDependants(ctx, taskID)
	if err != nil {
		return nil, err
	}
	for _, d := range dependants {
		if d.TaskID == dependsOnID {
			return nil, domain.ErrConflict
		}
	}

	// Обе связи ставим атомарно в одной транзакции: если вторая вставка
	// упала, прямую связь тоже откатываем.
	return repositories.InTxT(ctx, s.conn, func(qtx *db.Queries) (*domain.TaskDependency, error) {
		txRepo := repositories.NewTaskDependencyRepository(qtx)
		dep, err := txRepo.Add(ctx, taskID, dependsOnID, depType)
		if err != nil {
			return nil, err
		}
		if _, err := txRepo.Add(ctx, dependsOnID, taskID, inverseDepType(depType)); err != nil {
			return nil, err
		}
		return dep, nil
	})
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

func (s *TaskService) RemoveDependency(ctx context.Context, taskID, dependencyID uuid.UUID) error {
	dep, err := s.dependencyRepo.GetByID(ctx, dependencyID)
	if err != nil {
		return err
	}
	// Связь должна принадлежать именно этой задаче. Обратная парная строка
	// (task_id = dep.DependsOnTaskID) удаляется отдельным эндпоинтом на
	// противоположной стороне — здесь мы принимаем только «прямой» id.
	if dep.TaskID != taskID {
		return domain.ErrNotFound
	}

	return repositories.InTx(ctx, s.conn, func(qtx *db.Queries) error {
		txRepo := repositories.NewTaskDependencyRepository(qtx)
		if err := txRepo.Remove(ctx, dependencyID); err != nil {
			return err
		}
		return txRepo.RemoveInverse(ctx, dep.DependsOnTaskID, dep.TaskID)
	})
}

func (s *TaskService) ListDependencies(ctx context.Context, taskID uuid.UUID) ([]domain.TaskDependency, error) {
	return s.dependencyRepo.ListFor(ctx, taskID)
}

func (s *TaskService) CreateChecklist(ctx context.Context, taskID uuid.UUID, name string) (*domain.Checklist, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.checklistRepo.Create(ctx, taskID, name)
}

func (s *TaskService) ListChecklistsWithItems(ctx context.Context, taskID uuid.UUID) ([]domain.Checklist, error) {
	checklists, err := s.checklistRepo.ListByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	for i := range checklists {
		items, err := s.checklistRepo.ListItems(ctx, checklists[i].ID)
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
	return s.checklistRepo.CreateItem(ctx, checklistID, content, order)
}

func (s *TaskService) SetChecklistItemStatus(ctx context.Context, itemID uuid.UUID, isChecked bool) (*domain.ChecklistItem, error) {
	return s.checklistRepo.UpdateItemStatus(ctx, itemID, isChecked)
}

func (s *TaskService) UpdateChecklistName(ctx context.Context, checklistID uuid.UUID, name string) (*domain.Checklist, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.checklistRepo.UpdateName(ctx, checklistID, name)
}

func (s *TaskService) DeleteChecklist(ctx context.Context, checklistID uuid.UUID) error {
	return s.checklistRepo.Delete(ctx, checklistID)
}

func (s *TaskService) UpdateChecklistItemContent(ctx context.Context, itemID uuid.UUID, content string) (*domain.ChecklistItem, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.checklistRepo.UpdateItemContent(ctx, itemID, content)
}

func (s *TaskService) DeleteChecklistItem(ctx context.Context, itemID uuid.UUID) error {
	return s.checklistRepo.DeleteItem(ctx, itemID)
}

// ── Notification helpers ────────────────────────────────────

func (s *TaskService) taskPayload(task *domain.Task) []byte {
	tid := task.ID.String()
	p := domain.NotificationPayload{TaskID: &tid, TaskKey: &task.Key}
	data, _ := json.Marshal(p)
	return data
}

// notifyTaskAssigned sends a task_assigned notification to the executor.
// actorUserID is the user_id of the person who made the assignment — they are excluded.
func (s *TaskService) notifyTaskAssigned(ctx context.Context, task *domain.Task, executorMemberID uuid.UUID, actorUserID string) {
	member, err := s.queries.GetProjectMember(ctx, executorMemberID)
	if err != nil {
		log.Printf("[NOTIFY] task_assigned: GetProjectMember(%s) error: %v", executorMemberID, err)
		return
	}
	executorUserID := member.UserID.String()
	if executorUserID == actorUserID {
		return
	}
	title := fmt.Sprintf("Вам назначена задача %s", task.Key)
	if err := s.notificationSvc.SendEvent(ctx, domain.EventTaskAssigned, []string{executorUserID}, title, "", s.taskPayload(task)); err != nil {
		log.Printf("[NOTIFY] task_assigned: SendEvent error: %v", err)
	}
}

// notifyStatusChange sends task_status_change_* notifications to author, executor, and watchers.
// actorUserID is the user who moved the task — they are excluded from all notifications.
func (s *TaskService) notifyStatusChange(ctx context.Context, task *domain.Task, newColumnID uuid.UUID, actorUserID string) {
	col, err := s.queries.GetColumnByID(ctx, newColumnID)
	if err != nil {
		log.Printf("[NOTIFY] status_change: GetColumnByID(%s) error: %v", newColumnID, err)
		return
	}

	payload := s.taskPayload(task)
	title := fmt.Sprintf("Задача %s перемещена в «%s»", task.Key, col.Name)

	ownerUserIDStr := ""
	if task.OwnerUserID != nil {
		ownerUserIDStr = task.OwnerUserID.String()
	}
	executorUserIDStr := ""
	if task.ExecutorUserID != nil {
		executorUserIDStr = task.ExecutorUserID.String()
	}

	// Notify author (if not the actor)
	if ownerUserIDStr != "" && ownerUserIDStr != actorUserID {
		_ = s.notificationSvc.SendEvent(ctx, domain.EventTaskStatusChangeAuthor, []string{ownerUserIDStr}, title, "", payload)
	}

	// Notify executor (if assigned, not the actor, and not the author)
	if executorUserIDStr != "" && executorUserIDStr != actorUserID &&
		(ownerUserIDStr == "" || executorUserIDStr != ownerUserIDStr) {
		_ = s.notificationSvc.SendEvent(ctx, domain.EventTaskStatusChangeAssignee, []string{executorUserIDStr}, title, "", payload)
	}

	// Notify watchers
	watchers, err := s.watcherRepo.List(ctx, task.ID)
	if err != nil || len(watchers) == 0 {
		return
	}
	// Resolve watcher member_ids → user_ids, excluding author, executor, and actor
	exclude := make(map[string]bool)
	exclude[actorUserID] = true
	if ownerUserIDStr != "" {
		exclude[ownerUserIDStr] = true
	}
	if executorUserIDStr != "" {
		exclude[executorUserIDStr] = true
	}
	var watcherUserIDs []string
	for _, w := range watchers {
		m, err := s.queries.GetProjectMember(ctx, w.MemberID)
		if err != nil {
			continue
		}
		uid := m.UserID.String()
		if !exclude[uid] {
			watcherUserIDs = append(watcherUserIDs, uid)
		}
	}
	if len(watcherUserIDs) > 0 {
		_ = s.notificationSvc.SendEvent(ctx, domain.EventTaskStatusChangeWatcher, watcherUserIDs, title, "", payload)
	}
}

var mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_.]+)`)

// notifyMentions parses @username mentions from comment text and sends notifications.
func (s *TaskService) notifyMentions(ctx context.Context, taskID, authorID uuid.UUID, content string) {
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return
	}

	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return
	}

	authorStr := authorID.String()
	seen := make(map[string]bool)
	var userIDs []string
	for _, m := range matches {
		username := m[1]
		if seen[username] {
			continue
		}
		seen[username] = true
		user, err := s.queries.GetUserByUsername(ctx, username)
		if err != nil {
			continue
		}
		uid := user.ID.String()
		if uid != authorStr {
			userIDs = append(userIDs, uid)
		}
	}

	if len(userIDs) > 0 {
		title := fmt.Sprintf("Вас упомянули в комментарии к задаче %s", task.Key)
		_ = s.notificationSvc.SendEvent(ctx, domain.EventCommentMention, userIDs, title, "", s.taskPayload(task))
	}
}

