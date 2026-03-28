package services

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type TaskService struct {
	taskRepo    repositories.TaskRepository
	projectRepo repositories.ProjectRepository
}

func NewTaskService(taskRepo repositories.TaskRepository, projectRepo repositories.ProjectRepository) *TaskService {
	return &TaskService{taskRepo: taskRepo, projectRepo: projectRepo}
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

func (s *TaskService) CreateTask(ctx context.Context, projectID, ownerMemberID uuid.UUID, name, description string, executorMemberID *uuid.UUID, columnID uuid.UUID, swimlaneID *uuid.UUID, deadline *time.Time) (*domain.Task, error) {
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

	t := &domain.Task{
		ProjectID:  projectID.String(),
		OwnerID:    ownerMemberID.String(),
		ExecutorID: execPtr,
		Name:       name,
		Description: descPtr,
		Deadline:   deadlinePtr,
		ColumnID:   columnID.String(),
		SwimlaneID: swimlanePtr,
	}
	t.Key = key

	return s.taskRepo.Create(ctx, t)
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
	return task, nil
}

func (s *TaskService) ListProjectTasks(ctx context.Context, projectID uuid.UUID) ([]domain.Task, error) {
	return s.taskRepo.ListByProject(ctx, projectID)
}

func (s *TaskService) SearchTasks(ctx context.Context, projectID, ownerID, executorID, columnID *uuid.UUID) ([]domain.Task, error) {
	return s.taskRepo.Search(ctx, projectID, ownerID, executorID, columnID)
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

func (s *TaskService) DeleteTask(ctx context.Context, id uuid.UUID, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return domain.ErrInvalidInput
	}
	return s.taskRepo.SoftDelete(ctx, id, reason)
}

func (s *TaskService) AddWatcher(ctx context.Context, taskID, projectMemberID uuid.UUID) (*domain.TaskWatcher, error) {
	return s.taskRepo.AddWatcher(ctx, taskID, projectMemberID)
}

func (s *TaskService) RemoveWatcher(ctx context.Context, watcherID uuid.UUID) error {
	return s.taskRepo.RemoveWatcher(ctx, watcherID)
}

func (s *TaskService) ListWatchers(ctx context.Context, taskID uuid.UUID) ([]domain.TaskWatcher, error) {
	return s.taskRepo.ListWatchers(ctx, taskID)
}

func (s *TaskService) AddDependency(ctx context.Context, taskID, dependsOnID uuid.UUID, depType domain.TaskDependencyType) (*domain.TaskDependency, error) {
	if taskID == dependsOnID {
		return nil, domain.ErrInvalidInput
	}
	switch depType {
	case domain.TaskDependencyBlocks, domain.TaskDependencyRelated, domain.TaskDependencyParent, domain.TaskDependencyChild:
	default:
		return nil, domain.ErrInvalidInput
	}
	// TODO: в будущем добавить проверку циклов зависимостей
	return s.taskRepo.AddDependency(ctx, taskID, dependsOnID, depType)
}

func (s *TaskService) RemoveDependency(ctx context.Context, dependencyID uuid.UUID) error {
	return s.taskRepo.RemoveDependency(ctx, dependencyID)
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



