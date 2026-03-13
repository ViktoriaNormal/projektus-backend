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
	return s.taskRepo.GetByID(ctx, id)
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
	return s.taskRepo.Update(ctx, t)
}

func (s *TaskService) DeleteTask(ctx context.Context, id uuid.UUID, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return domain.ErrInvalidInput
	}
	return s.taskRepo.SoftDelete(ctx, id, reason)
}


