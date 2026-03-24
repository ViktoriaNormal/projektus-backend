package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type ProjectService struct {
	repo           repositories.ProjectRepository
	scrumRoleSVC   *ScrumRoleService
}

func NewProjectService(repo repositories.ProjectRepository, scrumRoleSVC *ScrumRoleService) *ProjectService {
	return &ProjectService{repo: repo, scrumRoleSVC: scrumRoleSVC}
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

	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return nil, err
	}

	// Инициализация Scrum-ролей для Scrum-проектов
	if pt == domain.ProjectTypeScrum && s.scrumRoleSVC != nil {
		if err := s.scrumRoleSVC.InitializeScrumRoles(ctx, created.ID); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *ProjectService) GetProject(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProjectService) ListProjects(ctx context.Context, userID uuid.UUID, query *string, status, projectType *string) ([]domain.Project, error) {
	return s.repo.ListUserProjects(ctx, userID, query, status, projectType)
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

// cyrToLat — таблица транслитерации кириллицы в латиницу.
var cyrToLat = map[rune]string{
	'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ё': "E",
	'Ж': "ZH", 'З': "Z", 'И': "I", 'Й': "Y", 'К': "K", 'Л': "L", 'М': "M",
	'Н': "N", 'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'У': "U",
	'Ф': "F", 'Х': "KH", 'Ц': "TS", 'Ч': "CH", 'Ш': "SH", 'Щ': "SHCH",
	'Ъ': "", 'Ы': "Y", 'Ь': "", 'Э': "E", 'Ю': "YU", 'Я': "YA",
}

// transliterate переводит строку из кириллицы в латиницу. Латинские символы остаются как есть.
func transliterate(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if lat, ok := cyrToLat[r]; ok {
			b.WriteString(lat)
		} else if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func generateProjectKeyPrefix(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "PRJ"
	}

	transliterated := transliterate(trimmed)
	if transliterated == "" {
		return "PRJ"
	}

	// Берём первые 3-5 латинских символов (только буквы)
	var letters []rune
	for _, r := range transliterated {
		if r >= 'A' && r <= 'Z' {
			letters = append(letters, r)
		}
		if len(letters) >= 4 {
			break
		}
	}
	if len(letters) == 0 {
		return "PRJ"
	}
	return string(letters)
}
