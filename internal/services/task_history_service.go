package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"projektus-backend/internal/repositories"
)

type TaskHistoryService struct {
	historyRepo repositories.TaskHistoryRepository
}

func NewTaskHistoryService(historyRepo repositories.TaskHistoryRepository) *TaskHistoryService {
	return &TaskHistoryService{historyRepo: historyRepo}
}

// RecordStatusChange регистрирует изменение статуса (колонки) задачи.
func (s *TaskHistoryService) RecordStatusChange(ctx context.Context, taskID, columnID uuid.UUID, fromColumnID *uuid.UUID) error {
	now := time.Now().UTC()

	// Если был предыдущий статус — закрываем его интервал.
	if fromColumnID != nil {
		if err := s.historyRepo.RecordStatusChange(ctx, taskID, *fromColumnID, now, &now); err != nil {
			return err
		}
	}

	// Открываем новый интервал (left_at = NULL).
	return s.historyRepo.RecordStatusChange(ctx, taskID, columnID, now, nil)
}

