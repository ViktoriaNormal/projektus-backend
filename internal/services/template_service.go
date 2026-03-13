package services

import (
	"context"
	"strings"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type TemplateService struct {
	repo repositories.TemplateRepository
}

func NewTemplateService(repo repositories.TemplateRepository) *TemplateService {
	return &TemplateService{repo: repo}
}

func (s *TemplateService) List(ctx context.Context) ([]domain.ProjectTemplate, error) {
	return s.repo.List(ctx)
}

func (s *TemplateService) Create(ctx context.Context, name, description, projectType string) (*domain.ProjectTemplate, error) {
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	pt := domain.ProjectType(strings.ToLower(projectType))
	if pt != domain.ProjectTypeScrum && pt != domain.ProjectTypeKanban {
		return nil, domain.ErrInvalidInput
	}
	var descPtr *string
	if description != "" {
		descPtr = &description
	}
	t := &domain.ProjectTemplate{
		Name:        name,
		Description: descPtr,
		Type:        pt,
	}
	return s.repo.Create(ctx, t)
}

