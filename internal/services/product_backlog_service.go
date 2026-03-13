package services

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type ProductBacklogService struct {
	backlogRepo repositories.ProductBacklogRepository
	taskRepo    repositories.TaskRepository
}

func NewProductBacklogService(backlogRepo repositories.ProductBacklogRepository, taskRepo repositories.TaskRepository) *ProductBacklogService {
	return &ProductBacklogService{backlogRepo: backlogRepo, taskRepo: taskRepo}
}

func (s *ProductBacklogService) GetProductBacklog(ctx context.Context, projectID uuid.UUID) ([]domain.Task, error) {
	items, err := s.backlogRepo.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Task, 0, len(items))
	for _, item := range items {
		tid := item.TaskID
		task, err := s.taskRepo.GetByID(ctx, tid)
		if err != nil {
			return nil, err
		}
		result = append(result, *task)
	}
	return result, nil
}

func (s *ProductBacklogService) AddToProductBacklog(ctx context.Context, projectID, taskID uuid.UUID, order int32) error {
	_, err := s.backlogRepo.Add(ctx, projectID, taskID, order)
	return err
}

func (s *ProductBacklogService) RemoveFromProductBacklog(ctx context.Context, projectID, taskID uuid.UUID) error {
	return s.backlogRepo.Remove(ctx, projectID, taskID)
}

func (s *ProductBacklogService) ReorderProductBacklog(ctx context.Context, projectID uuid.UUID, orders map[uuid.UUID]int32) error {
	for taskID, ord := range orders {
		if err := s.backlogRepo.UpdateOrder(ctx, projectID, taskID, ord); err != nil {
			return err
		}
	}
	return nil
}

