package services

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type ClassOfServiceService struct {
	classRepo    repositories.ClassOfServiceRepository
	swimlaneRepo repositories.SwimlaneConfigRepository
	boardRepo    repositories.BoardRepository
}

func NewClassOfServiceService(classRepo repositories.ClassOfServiceRepository, swimlaneRepo repositories.SwimlaneConfigRepository, boardRepo repositories.BoardRepository) *ClassOfServiceService {
	return &ClassOfServiceService{
		classRepo:    classRepo,
		swimlaneRepo: swimlaneRepo,
		boardRepo:    boardRepo,
	}
}

func (s *ClassOfServiceService) GetDefaultClasses() []domain.ClassOfService {
	return domain.GetAllDefaultClasses()
}

func (s *ClassOfServiceService) SetTaskClass(ctx context.Context, taskID uuid.UUID, classStr string) error {
	classStr = strings.ToLower(strings.TrimSpace(classStr))
	class := domain.ClassOfService(classStr)

	valid := false
	for _, c := range domain.GetAllDefaultClasses() {
		if c == class {
			valid = true
			break
		}
	}
	if !valid {
		return domain.ErrInvalidInput
	}

	return s.classRepo.UpdateTaskClass(ctx, taskID, class)
}

func (s *ClassOfServiceService) ConfigureSwimlanes(ctx context.Context, boardID uuid.UUID, sourceTypeStr string, customFieldID *uuid.UUID, valueMappings map[string]string) error {
	src := domain.SwimlaneSourceType(strings.ToLower(strings.TrimSpace(sourceTypeStr)))
	switch src {
	case domain.SwimlaneSourceClassOfService, domain.SwimlaneSourceCustomField:
	default:
		return domain.ErrInvalidInput
	}

	boardIDStr := boardID.String()
	lanes, err := s.boardRepo.ListSwimlanes(ctx, boardIDStr)
	if err != nil {
		return err
	}

	// Если доска ещё не имеет дорожек и выбран источник по классам обслуживания —
	// создаём по одной дорожке на каждый класс обслуживания.
	if len(lanes) == 0 && src == domain.SwimlaneSourceClassOfService {
		var order int16 = 1
		for _, c := range domain.GetAllDefaultClasses() {
			name := strings.Title(strings.ReplaceAll(string(c), "_", " "))
			_, err := s.boardRepo.CreateSwimlane(ctx, &domain.Swimlane{
				BoardID: boardIDStr,
				Name:    name,
				Order:   order,
			})
			if err != nil {
				return err
			}
			order++
		}
		lanes, err = s.boardRepo.ListSwimlanes(ctx, boardIDStr)
		if err != nil {
			return err
		}
	}

	cfgTemplate := domain.SwimlaneConfig{
		SourceType:  src,
		CustomFieldID: customFieldID,
		ValueMappings: valueMappings,
	}

	for _, lane := range lanes {
		cfg := cfgTemplate
		cfg.BoardID = boardID
		laneID, err := uuid.Parse(lane.ID)
		if err != nil {
			return err
		}
		if _, err := s.swimlaneRepo.UpdateConfig(ctx, laneID, cfg); err != nil {
			return err
		}
	}

	return nil
}

