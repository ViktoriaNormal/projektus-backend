package services

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type AttachmentService struct {
	repo repositories.AttachmentRepository
}

func NewAttachmentService(repo repositories.AttachmentRepository) *AttachmentService {
	return &AttachmentService{repo: repo}
}

func (s *AttachmentService) CreateTaskAttachment(ctx context.Context, taskID uuid.UUID, fileName, filePath string, uploadedBy uuid.UUID) (*domain.Attachment, error) {
	if !isAllowedFile(fileName) {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.CreateForTask(ctx, taskID, fileName, filePath, uploadedBy)
}

func (s *AttachmentService) CreateCommentAttachment(ctx context.Context, commentID uuid.UUID, fileName, filePath string, uploadedBy uuid.UUID) (*domain.Attachment, error) {
	if !isAllowedFile(fileName) {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.CreateForComment(ctx, commentID, fileName, filePath, uploadedBy)
}

func (s *AttachmentService) ListTaskAttachments(ctx context.Context, taskID uuid.UUID) ([]domain.Attachment, error) {
	return s.repo.ListForTask(ctx, taskID)
}

func (s *AttachmentService) ListCommentAttachments(ctx context.Context, commentID uuid.UUID) ([]domain.Attachment, error) {
	return s.repo.ListForComment(ctx, commentID)
}

func isAllowedFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".pdf", ".docx", ".xlsx", ".txt":
		return true
	default:
		return false
	}
}

