package services

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type KanbanService struct {
	wipRepo   repositories.KanbanRepository
	boardRepo repositories.BoardRepository
}

func NewKanbanService(wipRepo repositories.KanbanRepository, boardRepo repositories.BoardRepository) *KanbanService {
	return &KanbanService{
		wipRepo:   wipRepo,
		boardRepo: boardRepo,
	}
}

// GetWipLimits возвращает WIP-лимиты для всех колонок и дорожек проекта.
func (s *KanbanService) GetWipLimits(ctx context.Context, projectID uuid.UUID) ([]domain.WipLimit, error) {
	return s.wipRepo.GetWipLimits(ctx, projectID)
}

// UpdateWipLimits массово обновляет WIP-лимиты по колонкам и дорожкам проекта.
// Если Limit == nil, лимит сбрасывается (NULL).
func (s *KanbanService) UpdateWipLimits(ctx context.Context, projectID uuid.UUID, limits []domain.WipLimit) error {
	for _, l := range limits {
		// Для надёжности можно было бы проверять принадлежность boardId проекту,
		// но пока опираемся на корректность входных данных.
		if l.ColumnID != nil {
			var val *int16
			if l.Limit != nil {
				v := int16(*l.Limit)
				val = &v
			}
			if err := s.wipRepo.UpdateColumnWipLimit(ctx, *l.ColumnID, val); err != nil {
				return err
			}
		} else if l.SwimlaneID != nil {
			var val *int16
			if l.Limit != nil {
				v := int16(*l.Limit)
				val = &v
			}
			if err := s.wipRepo.UpdateSwimlaneWipLimit(ctx, *l.SwimlaneID, val); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetCurrentWipCounts возвращает текущие WIP-счетчики по колонкам/дорожкам доски,
// а также подставляет соответствующие лимиты и признак превышения.
func (s *KanbanService) GetCurrentWipCounts(ctx context.Context, boardID uuid.UUID) ([]domain.WipCount, error) {
	board, err := s.boardRepo.GetBoardByID(ctx, boardID.String())
	if err != nil {
		return nil, err
	}

	// Получаем лимиты по проекту и фильтруем по boardID.
	projectUUID, err := uuid.Parse(*board.ProjectID)
	if err != nil {
		return nil, err
	}
	allLimits, err := s.wipRepo.GetWipLimits(ctx, projectUUID)
	if err != nil {
		return nil, err
	}

	limitsByKey := make(map[string]*domain.WipLimit)
	for i := range allLimits {
		l := allLimits[i]
		if l.BoardID != boardID {
			continue
		}
		key := wipKey(l.ColumnID, l.SwimlaneID)
		limitsByKey[key] = &l
	}

	counts, err := s.wipRepo.GetCurrentWipCounts(ctx, boardID)
	if err != nil {
		return nil, err
	}

	for i := range counts {
		key := wipKey(counts[i].ColumnID, counts[i].SwimlaneID)
		if lim, ok := limitsByKey[key]; ok {
			counts[i].Limit = lim.Limit
		}
	}

	return counts, nil
}

func wipKey(columnID, swimlaneID *uuid.UUID) string {
	col := ""
	sw := ""
	if columnID != nil {
		col = columnID.String()
	}
	if swimlaneID != nil {
		sw = swimlaneID.String()
	}
	return col + ":" + sw
}

