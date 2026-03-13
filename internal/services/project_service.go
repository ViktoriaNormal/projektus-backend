package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type ProjectService struct {
	repo repositories.ProjectRepository
}

func NewProjectService(repo repositories.ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) CreateProject(ctx context.Context, ownerID uuid.UUID, name, description, projectType string) (*domain.Project, error) {
	if name == "" {
		return nil, domain.ErrInvalidInput
	}

	pt := domain.ProjectType(strings.ToLower(projectType))
	if pt != domain.ProjectTypeScrum && pt != domain.ProjectTypeKanban {
		return nil, domain.ErrInvalidInput
	}

	keyPrefix := generateProjectKeyPrefix(name)
	key, err := s.generateUniqueProjectKey(ctx, keyPrefix)
	if err != nil {
		return nil, err
	}

	var descPtr *string
	if description != "" {
		descPtr = &description
	}

	p := &domain.Project{
		Key:         key,
		Name:        name,
		Description: descPtr,
		Type:        pt,
		OwnerID:     ownerID,
		Status:      domain.ProjectStatusActive,
	}

	return s.repo.Create(ctx, p)
}

func (s *ProjectService) GetProject(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProjectService) ListProjects(ctx context.Context, ownerID uuid.UUID, status, projectType *string) ([]domain.Project, error) {
	return s.repo.ListByOwner(ctx, ownerID, status, projectType)
}

func (s *ProjectService) UpdateProject(ctx context.Context, p *domain.Project) (*domain.Project, error) {
	return s.repo.Update(ctx, p)
}

func (s *ProjectService) DeleteProject(ctx context.Context, id uuid.UUID, confirm bool) error {
	if !confirm {
		return domain.ErrInvalidInput
	}
	return s.repo.Delete(ctx, id)
}

func (s *ProjectService) generateUniqueProjectKey(ctx context.Context, prefix string) (string, error) {
	for i := 1; i < 1000000; i++ {
		key := fmt.Sprintf("%s-%d", prefix, i)
		_, err := s.repo.GetByKey(ctx, key)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return key, nil
			}
			return "", err
		}
	}
	return "", domain.ErrConflict
}

func generateProjectKeyPrefix(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "PRJ"
	}

	var letters []rune
	for _, r := range trimmed {
		if unicode.IsLetter(r) {
			letters = append(letters, unicode.ToUpper(r))
		}
		if len(letters) >= 3 {
			break
		}
	}
	if len(letters) == 0 {
		return "PRJ"
	}
	return string(letters)
}
