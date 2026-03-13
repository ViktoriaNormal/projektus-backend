package services

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

var mentionPattern = regexp.MustCompile(`@([A-Za-z0-9_-]+)`)

type CommentService struct {
	commentRepo       repositories.CommentRepository
	projectMemberRepo repositories.ProjectMemberRepository
}

func NewCommentService(commentRepo repositories.CommentRepository, projectMemberRepo repositories.ProjectMemberRepository) *CommentService {
	return &CommentService{commentRepo: commentRepo, projectMemberRepo: projectMemberRepo}
}

func (s *CommentService) CreateComment(ctx context.Context, taskID, authorMemberID uuid.UUID, content string) (*domain.Comment, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, domain.ErrInvalidInput
	}

	comment, err := s.commentRepo.CreateComment(ctx, taskID, authorMemberID, content)
	if err != nil {
		return nil, err
	}

	// Простая обработка упоминаний по username'ам вида @username.
	matches := mentionPattern.FindAllStringSubmatch(content, -1)
	seen := make(map[string]struct{})
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		username := strings.ToLower(m[1])
		if _, ok := seen[username]; ok {
			continue
		}
		seen[username] = struct{}{}

		// Здесь можно было бы найти project_member по username пользователя.
		// Для простоты пока пропускаем реальное разрешение username -> project_member_id.
		_ = username
		// В будущем: s.projectMemberRepo.FindByUsernameInProject(...)
	}

	return comment, nil
}

func (s *CommentService) ListTaskComments(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error) {
	return s.commentRepo.ListTaskComments(ctx, taskID)
}

