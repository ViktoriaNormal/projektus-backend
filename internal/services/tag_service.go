package services

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type TagService struct {
	repo repositories.TagRepository
}

func NewTagService(repo repositories.TagRepository) *TagService {
	return &TagService{repo: repo}
}

// ListBoardTags возвращает все теги доски (для автокомплита).
func (s *TagService) ListBoardTags(ctx context.Context, boardID uuid.UUID) ([]domain.Tag, error) {
	return s.repo.ListByBoard(ctx, boardID)
}

// ListTaskTags возвращает теги конкретной задачи.
func (s *TagService) ListTaskTags(ctx context.Context, taskID uuid.UUID) ([]domain.Tag, error) {
	return s.repo.ListTaskTags(ctx, taskID)
}

// AddTagToTask создаёт тег (если не существует) и привязывает к задаче.
func (s *TagService) AddTagToTask(ctx context.Context, boardID, taskID uuid.UUID, name string) (*domain.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrInvalidInput
	}

	// Ищем существующий тег на доске
	tag, err := s.repo.GetByBoardAndName(ctx, boardID, name)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	if tag == nil {
		// Создаём новый тег
		tag, err = s.repo.Create(ctx, boardID, name)
		if err != nil {
			return nil, err
		}
	}

	tagID, _ := uuid.Parse(tag.ID)
	if err := s.repo.AddTagToTask(ctx, taskID, tagID); err != nil {
		return nil, err
	}

	return tag, nil
}

// RemoveTagFromTask убирает тег с задачи. Если тег больше не используется, удаляет его.
func (s *TagService) RemoveTagFromTask(ctx context.Context, taskID, tagID uuid.UUID) error {
	if err := s.repo.RemoveTagFromTask(ctx, taskID, tagID); err != nil {
		return err
	}

	// Если тег больше никем не используется — удаляем
	count, err := s.repo.CountTasksWithTag(ctx, tagID)
	if err != nil {
		return nil // не критично
	}
	if count == 0 {
		_ = s.repo.Delete(ctx, tagID)
	}

	return nil
}

// SetTaskTags заменяет все теги задачи на новый набор.
func (s *TagService) SetTaskTags(ctx context.Context, boardID, taskID uuid.UUID, tagNames []string) ([]domain.Tag, error) {
	if err := s.repo.RemoveAllTagsFromTask(ctx, taskID); err != nil {
		return nil, err
	}

	result := make([]domain.Tag, 0, len(tagNames))
	for _, name := range tagNames {
		tag, err := s.AddTagToTask(ctx, boardID, taskID, name)
		if err != nil {
			return nil, err
		}
		result = append(result, *tag)
	}

	return result, nil
}
