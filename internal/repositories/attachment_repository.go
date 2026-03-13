package repositories

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type AttachmentRepository interface {
	CreateForTask(ctx context.Context, taskID uuid.UUID, fileName, filePath string, uploadedBy uuid.UUID) (*domain.Attachment, error)
	CreateForComment(ctx context.Context, commentID uuid.UUID, fileName, filePath string, uploadedBy uuid.UUID) (*domain.Attachment, error)
	ListForTask(ctx context.Context, taskID uuid.UUID) ([]domain.Attachment, error)
	ListForComment(ctx context.Context, commentID uuid.UUID) ([]domain.Attachment, error)
}

type attachmentRepository struct {
	q *db.Queries
}

func NewAttachmentRepository(q *db.Queries) AttachmentRepository {
	return &attachmentRepository{q: q}
}

func mapDBAttachment(a db.Attachment) domain.Attachment {
	var taskID *string
	if a.TaskID.Valid {
		id := a.TaskID.UUID.String()
		taskID = &id
	}
	var commentID *string
	if a.CommentID.Valid {
		id := a.CommentID.UUID.String()
		commentID = &id
	}
	return domain.Attachment{
		ID:         a.ID.String(),
		TaskID:     taskID,
		CommentID:  commentID,
		FileName:   a.FileName,
		FilePath:   a.FilePath,
		UploadedBy: a.UploadedBy.String(),
		UploadedAt: a.UploadedAt,
	}
}

func (r *attachmentRepository) CreateForTask(ctx context.Context, taskID uuid.UUID, fileName, filePath string, uploadedBy uuid.UUID) (*domain.Attachment, error) {
	row, err := r.q.CreateTaskAttachment(ctx, db.CreateTaskAttachmentParams{
		TaskID:     uuid.NullUUID{UUID: taskID, Valid: true},
		FileName:   fileName,
		FilePath:   filePath,
		UploadedBy: uploadedBy,
	})
	if err != nil {
		return nil, err
	}
	d := mapDBAttachment(row)
	return &d, nil
}

func (r *attachmentRepository) CreateForComment(ctx context.Context, commentID uuid.UUID, fileName, filePath string, uploadedBy uuid.UUID) (*domain.Attachment, error) {
	row, err := r.q.CreateCommentAttachment(ctx, db.CreateCommentAttachmentParams{
		CommentID:  uuid.NullUUID{UUID: commentID, Valid: true},
		FileName:   fileName,
		FilePath:   filePath,
		UploadedBy: uploadedBy,
	})
	if err != nil {
		return nil, err
	}
	d := mapDBAttachment(row)
	return &d, nil
}

func (r *attachmentRepository) ListForTask(ctx context.Context, taskID uuid.UUID) ([]domain.Attachment, error) {
	rows, err := r.q.ListTaskAttachments(ctx, uuid.NullUUID{UUID: taskID, Valid: true})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Attachment, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapDBAttachment(row))
	}
	return result, nil
}

func (r *attachmentRepository) ListForComment(ctx context.Context, commentID uuid.UUID) ([]domain.Attachment, error) {
	rows, err := r.q.ListCommentAttachments(ctx, uuid.NullUUID{UUID: commentID, Valid: true})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Attachment, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapDBAttachment(row))
	}
	return result, nil
}

