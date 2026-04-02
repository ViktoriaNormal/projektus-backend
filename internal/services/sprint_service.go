package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type SprintService struct {
	repo           repositories.SprintRepository
	sprintTaskRepo repositories.SprintTaskRepository
	backlogRepo    repositories.ProductBacklogRepository
	taskRepo       repositories.TaskRepository
	boardRepo      repositories.BoardRepository
	projectRepo    repositories.ProjectRepository
}

func NewSprintService(repo repositories.SprintRepository, sprintTaskRepo repositories.SprintTaskRepository, backlogRepo repositories.ProductBacklogRepository, taskRepo repositories.TaskRepository, boardRepo repositories.BoardRepository, projectRepo repositories.ProjectRepository) *SprintService {
	return &SprintService{
		repo:           repo,
		sprintTaskRepo: sprintTaskRepo,
		backlogRepo:    backlogRepo,
		taskRepo:       taskRepo,
		boardRepo:      boardRepo,
		projectRepo:    projectRepo,
	}
}

func (s *SprintService) CreateSprint(ctx context.Context, projectID uuid.UUID, name string, goal *string, startDate time.Time, durationDays int) (*domain.Sprint, error) {
	if name == "" || durationDays <= 0 {
		return nil, domain.ErrInvalidInput
	}
	endDate := startDate.AddDate(0, 0, durationDays-1)

	// Валидация пересечений с незавершёнными спринтами
	existing, err := s.repo.GetNonCompletedSprints(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, ex := range existing {
		if datesOverlap(startDate, endDate, ex.StartDate, ex.EndDate) {
			return nil, domain.ErrSprintDatesOverlap
		}
	}

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

func datesOverlap(s1, e1, s2, e2 time.Time) bool {
	return !s1.After(e2) && !s2.After(e1)
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

	// 1. Валидация: нельзя запустить, если уже есть active спринт
	activeSprint, err := s.repo.GetActiveSprint(ctx, sprint.ProjectID)
	if err != nil && err != domain.ErrNotFound {
		return nil, err
	}
	if activeSprint != nil && activeSprint.ID != id {
		return nil, domain.ErrActiveSprintExists
	}

	// 2. start_date = сегодня, end_date пересчитываем по sprint_duration_weeks проекта
	today := time.Now().Truncate(24 * time.Hour)
	sprint.StartDate = today

	project, err := s.projectRepo.GetByID(ctx, sprint.ProjectID)
	if err != nil {
		return nil, err
	}
	durationDays := 14 // fallback 2 недели
	if project.SprintDurationWeeks != nil {
		durationDays = *project.SprintDurationWeeks * 7
	}
	sprint.EndDate = sprint.StartDate.AddDate(0, 0, durationDays-1)

	// 3. Каскадный сдвиг пересекающихся planned-спринтов
	if err := s.shiftOverlappingSprints(ctx, sprint); err != nil {
		return nil, err
	}

	// 4. Assign columns to all sprint tasks that don't have one yet.
	tasksWithoutColumn, err := s.taskRepo.ListSprintTasksWithoutColumn(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, t := range tasksWithoutColumn {
		if t.BoardID == "" {
			return nil, domain.ErrInvalidInput
		}
		columns, err := s.boardRepo.ListColumns(ctx, t.BoardID)
		if err != nil {
			return nil, err
		}
		for _, col := range columns {
			if col.SystemType != nil && *col.SystemType == domain.StatusInitial {
				taskID, _ := uuid.Parse(t.TaskID)
				colID, _ := uuid.Parse(col.ID)
				if err := s.taskRepo.AssignColumnToTask(ctx, taskID, colID); err != nil {
					return nil, err
				}
				break
			}
		}
	}

	sprint.Status = domain.SprintStatusActive
	return s.repo.Update(ctx, sprint)
}

// shiftOverlappingSprints каскадно сдвигает planned-спринты, пересекающиеся с запускаемым.
func (s *SprintService) shiftOverlappingSprints(ctx context.Context, started *domain.Sprint) error {
	planned, err := s.repo.GetPlannedSprints(ctx, started.ProjectID)
	if err != nil {
		return err
	}

	prevEnd := started.EndDate
	for _, sp := range planned {
		if sp.ID == started.ID {
			continue
		}
		// Сдвигаем только если есть реальное пересечение
		if !sp.StartDate.After(prevEnd) {
			newStart := prevEnd.AddDate(0, 0, 1)
			duration := int(sp.EndDate.Sub(sp.StartDate).Hours()/24) + 1
			sp.StartDate = newStart
			sp.EndDate = newStart.AddDate(0, 0, duration-1)
			if _, err := s.repo.Update(ctx, &sp); err != nil {
				return err
			}
			prevEnd = sp.EndDate
		} else {
			break // дальше пересечений нет
		}
	}
	return nil
}

func (s *SprintService) CompleteSprint(ctx context.Context, id uuid.UUID, incompleteTasksAction string) (*domain.Sprint, error) {
	sprint, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sprint.Status == domain.SprintStatusCompleted {
		return sprint, nil
	}

	// Собираем незавершённые задачи (column_system_type != 'completed')
	tasks, err := s.sprintTaskRepo.ListSprintTasksFull(ctx, id)
	if err != nil {
		return nil, err
	}

	var incompleteTasks []domain.Task
	for _, t := range tasks {
		if t.ColumnSystemType == nil || *t.ColumnSystemType != string(domain.StatusCompleted) {
			incompleteTasks = append(incompleteTasks, t)
		}
	}

	if len(incompleteTasks) > 0 {
		switch incompleteTasksAction {
		case "next_sprint":
			err = s.moveIncompleteTasksToNextSprint(ctx, sprint.ProjectID, id, incompleteTasks)
		case "backlog":
			err = s.moveIncompleteTasksToBacklog(ctx, sprint.ProjectID, id, incompleteTasks)
		default:
			err = s.moveIncompleteTasksToBacklog(ctx, sprint.ProjectID, id, incompleteTasks)
		}
		if err != nil {
			return nil, err
		}
	}

	sprint.Status = domain.SprintStatusCompleted
	return s.repo.Update(ctx, sprint)
}

func (s *SprintService) moveIncompleteTasksToNextSprint(ctx context.Context, projectID uuid.UUID, currentSprintID uuid.UUID, tasks []domain.Task) error {
	nextSprint, err := s.repo.GetNextPlannedSprint(ctx, projectID)
	if err != nil {
		if err == domain.ErrNotFound {
			return domain.ErrNoNextSprintForMove
		}
		return err
	}

	// Получаем текущий max sort_order в следующем спринте
	nextSprintTasks, err := s.sprintTaskRepo.ListBySprint(ctx, nextSprint.ID)
	if err != nil {
		return err
	}
	var maxOrder int32
	for _, t := range nextSprintTasks {
		if int32(t.Order) > maxOrder {
			maxOrder = int32(t.Order)
		}
	}

	for _, t := range tasks {
		taskID := uuid.MustParse(t.ID)
		maxOrder++
		if _, err := s.sprintTaskRepo.AddTask(ctx, nextSprint.ID, taskID, &maxOrder); err != nil {
			return err
		}
		if err := s.sprintTaskRepo.RemoveTask(ctx, currentSprintID, taskID); err != nil {
			return err
		}
		// Сбрасываем column_id, чтобы при старте следующего спринта они получили initial-колонку
		if err := s.taskRepo.ClearColumnFromTask(ctx, taskID); err != nil {
			return err
		}
	}
	return nil
}

func (s *SprintService) moveIncompleteTasksToBacklog(ctx context.Context, projectID uuid.UUID, sprintID uuid.UUID, tasks []domain.Task) error {
	for _, t := range tasks {
		taskID := uuid.MustParse(t.ID)
		if err := s.sprintTaskRepo.RemoveTask(ctx, sprintID, taskID); err != nil {
			return err
		}
		if _, err := s.backlogRepo.Add(ctx, projectID, taskID, 0); err != nil {
			return err
		}
		// Сбрасываем column_id
		if err := s.taskRepo.ClearColumnFromTask(ctx, taskID); err != nil {
			return err
		}
	}
	return nil
}

func (s *SprintService) GetSprintTasks(ctx context.Context, sprintID uuid.UUID) ([]domain.Task, error) {
	return s.sprintTaskRepo.ListSprintTasksFull(ctx, sprintID)
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
		// Удаляем из всех спринтов (если задача уже в другом спринте)
		if err := s.sprintTaskRepo.RemoveTaskFromAllSprints(ctx, tid); err != nil {
			return err
		}
		// Удаляем из product backlog (если была там)
		_ = s.backlogRepo.Remove(ctx, projectID, tid)
		maxOrder++
		if _, err := s.sprintTaskRepo.AddTask(ctx, sprintID, tid, &maxOrder); err != nil {
			return err
		}
	}
	return nil
}


