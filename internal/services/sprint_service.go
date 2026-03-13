package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type SprintService struct {
	repo          repositories.SprintRepository
	sprintTaskRepo repositories.SprintTaskRepository
	backlogRepo    repositories.ProductBacklogRepository
	taskRepo       repositories.TaskRepository
}

func NewSprintService(repo repositories.SprintRepository, sprintTaskRepo repositories.SprintTaskRepository, backlogRepo repositories.ProductBacklogRepository, taskRepo repositories.TaskRepository) *SprintService {
	return &SprintService{
		repo:           repo,
		sprintTaskRepo: sprintTaskRepo,
		backlogRepo:    backlogRepo,
		taskRepo:       taskRepo,
	}
}

func (s *SprintService) CreateSprint(ctx context.Context, projectID uuid.UUID, name string, goal *string, startDate time.Time, durationDays int) (*domain.Sprint, error) {
	if name == "" || durationDays <= 0 {
		return nil, domain.ErrInvalidInput
	}
	endDate := startDate.AddDate(0, 0, durationDays-1)
	tmp := &domain.Sprint{
		ProjectID: projectID,
		Name:      name,
		Goal:      goal,
		StartDate: startDate,
		EndDate:   endDate,
		Status:    domain.SprintStatusPlanned,
	}
	return s.repo.Create(ctx, tmp)
}

func (s *SprintService) GetSprint(ctx context.Context, id uuid.UUID) (*domain.Sprint, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *SprintService) GetProjectSprints(ctx context.Context, projectID uuid.UUID) ([]domain.Sprint, error) {
	return s.repo.GetProjectSprints(ctx, projectID)
}

func (s *SprintService) GetActiveSprint(ctx context.Context, projectID uuid.UUID) (*domain.Sprint, error) {
	return s.repo.GetActiveSprint(ctx, projectID)
}

func (s *SprintService) UpdateSprint(ctx context.Context, sprint *domain.Sprint, name, goal *string, startDate *time.Time, durationDays *int) (*domain.Sprint, error) {
	if name != nil {
		sprint.Name = *name
	}
	if goal != nil {
		sprint.Goal = goal
	}
	if startDate != nil {
		sprint.StartDate = *startDate
	}
	if durationDays != nil && *durationDays > 0 {
		sprint.EndDate = sprint.StartDate.AddDate(0, 0, *durationDays-1)
	}
	return s.repo.Update(ctx, sprint)
}

func (s *SprintService) DeleteSprint(ctx context.Context, id uuid.UUID) error {
	// Дополнительные проверки (нет задач, статус planned) будут добавлены позже
	return s.repo.Delete(ctx, id)
}

func (s *SprintService) StartSprint(ctx context.Context, id uuid.UUID) (*domain.Sprint, error) {
	sprint, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sprint.Status == domain.SprintStatusCompleted {
		return nil, domain.ErrInvalidInput
	}
	sprint.Status = domain.SprintStatusActive
	return s.repo.Update(ctx, sprint)
}

func (s *SprintService) CompleteSprint(ctx context.Context, id uuid.UUID) (*domain.Sprint, error) {
	sprint, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sprint.Status == domain.SprintStatusCompleted {
		return sprint, nil
	}
	sprint.Status = domain.SprintStatusCompleted
	return s.repo.Update(ctx, sprint)
}

func (s *SprintService) GetSprintBacklog(ctx context.Context, sprintID uuid.UUID) ([]domain.Task, error) {
	sprintTasks, err := s.sprintTaskRepo.ListBySprint(ctx, sprintID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Task, 0, len(sprintTasks))
	for _, st := range sprintTasks {
		task, err := s.taskRepo.GetByID(ctx, st.TaskID)
		if err != nil {
			return nil, err
		}
		result = append(result, *task)
	}
	return result, nil
}

func (s *SprintService) AddTaskToSprint(ctx context.Context, sprintID, taskID uuid.UUID) error {
	tasks, err := s.sprintTaskRepo.ListBySprint(ctx, sprintID)
	if err != nil {
		return err
	}
	var maxOrder int32
	for _, t := range tasks {
		if int32(t.Order) > maxOrder {
			maxOrder = int32(t.Order)
		}
	}
	next := maxOrder + 1
	if _, err := s.sprintTaskRepo.AddTask(ctx, sprintID, taskID, &next); err != nil {
		return err
	}
	return nil
}

func (s *SprintService) RemoveTaskFromSprint(ctx context.Context, sprintID, taskID uuid.UUID) error {
	return s.sprintTaskRepo.RemoveTask(ctx, sprintID, taskID)
}

func (s *SprintService) MoveTasksToSprint(ctx context.Context, sprintID uuid.UUID, projectID uuid.UUID, taskIDs []uuid.UUID) error {
	tasks, err := s.sprintTaskRepo.ListBySprint(ctx, sprintID)
	if err != nil {
		return err
	}
	var maxOrder int32
	for _, t := range tasks {
		if int32(t.Order) > maxOrder {
			maxOrder = int32(t.Order)
		}
	}
	for _, tid := range taskIDs {
		maxOrder++
		if _, err := s.sprintTaskRepo.AddTask(ctx, sprintID, tid, &maxOrder); err != nil {
			return err
		}
		if err := s.backlogRepo.Remove(ctx, projectID, tid); err != nil {
			return err
		}
	}
	return nil
}


