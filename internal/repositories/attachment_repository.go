package repositories

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type AttachmentRepository interface {
	List(ctx context.Context, taskID uuid.UUID) ([]domain.Attachment, error)
	Create(ctx context.Context, taskID, uploadedBy uuid.UUID, fileName, filePath, contentType string, fileSize int64) (*domain.Attachment, error)
	GetByID(ctx context.Context, attachmentID uuid.UUID) (*domain.Attachment, error)
	Delete(ctx context.Context, attachmentID uuid.UUID) error
}

type attachmentRepository struct {
	q *db.Queries
}

func NewAttachmentRepository(q *db.Queries) AttachmentRepository {
	return &attachmentRepository{q: q}
}

func (r *attachmentRepository) List(ctx context.Context, taskID uuid.UUID) ([]domain.Attachment, error) {
	rows, err := r.q.ListTaskAttachments(ctx, uuid.NullUUID{UUID: taskID, Valid: true})
	if err != nil {
		return nil, errctx.Wrap(err, "ListTaskAttachments", "taskID", taskID)
	}
	result := make([]domain.Attachment, 0, len(rows))
	for _, row := range rows {
		a := domain.Attachment{
			ID:          row.ID,
			FileName:    row.FileName,
			FilePath:    row.FilePath,
			FileSize:    row.FileSize,
			ContentType: row.ContentType,
			UploadedBy:  row.UploadedBy,
			UploadedAt:  row.UploadedAt,
		}
		if row.TaskID.Valid {
			id := row.TaskID.UUID
			a.TaskID = &id
		}
		if row.CommentID.Valid {
			id := row.CommentID.UUID
			a.CommentID = &id
		}
		result = append(result, a)
	}
	return result, nil
}

func (r *attachmentRepository) Create(ctx context.Context, taskID, uploadedBy uuid.UUID, fileName, filePath, contentType string, fileSize int64) (*domain.Attachment, error) {
	row, err := r.q.CreateAttachment(ctx, db.CreateAttachmentParams{
		TaskID:      uuid.NullUUID{UUID: taskID, Valid: true},
		CommentID:   uuid.NullUUID{},
		FileName:    fileName,
		FilePath:    filePath,
		FileSize:    fileSize,
		ContentType: contentType,
		UploadedBy:  uploadedBy,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateAttachment", "taskID", taskID, "fileName", fileName)
	}
	a := &domain.Attachment{
		ID:          row.ID,
		FileName:    row.FileName,
		FilePath:    row.FilePath,
		FileSize:    row.FileSize,
		ContentType: row.ContentType,
		UploadedBy:  row.UploadedBy,
		UploadedAt:  row.UploadedAt,
	}
	if row.TaskID.Valid {
		id := row.TaskID.UUID
		a.TaskID = &id
	}
	return a, nil
}

func (r *attachmentRepository) GetByID(ctx context.Context, attachmentID uuid.UUID) (*domain.Attachment, error) {
	row, err := r.q.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetAttachmentByID", "attachmentID", attachmentID)
	}
	a := &domain.Attachment{
		ID:          row.ID,
		FileName:    row.FileName,
		FilePath:    row.FilePath,
		FileSize:    row.FileSize,
		ContentType: row.ContentType,
		UploadedBy:  row.UploadedBy,
		UploadedAt:  row.UploadedAt,
	}
	if row.TaskID.Valid {
		id := row.TaskID.UUID
		a.TaskID = &id
	}
	if row.CommentID.Valid {
		id := row.CommentID.UUID
		a.CommentID = &id
	}
	return a, nil
}

func (r *attachmentRepository) Delete(ctx context.Context, attachmentID uuid.UUID) error {
	return errctx.Wrap(r.q.DeleteAttachment(ctx, attachmentID), "DeleteAttachment", "attachmentID", attachmentID)
}
