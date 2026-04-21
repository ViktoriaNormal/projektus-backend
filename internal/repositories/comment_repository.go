package repositories

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type CommentRepository interface {
	List(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error)
	Create(ctx context.Context, taskID, authorID uuid.UUID, content string, parentCommentID *uuid.UUID) (*domain.Comment, error)
	GetByID(ctx context.Context, commentID uuid.UUID) (*domain.Comment, error)
	Delete(ctx context.Context, commentID uuid.UUID) error
}

type commentRepository struct {
	q *db.Queries
}

func NewCommentRepository(q *db.Queries) CommentRepository {
	return &commentRepository{q: q}
}

func (r *commentRepository) List(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error) {
	rows, err := r.q.ListTaskComments(ctx, taskID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListTaskComments", "taskID", taskID)
	}
	result := make([]domain.Comment, 0, len(rows))
	for _, row := range rows {
		c := domain.Comment{
			ID:        row.ID,
			TaskID:    row.TaskID,
			AuthorID:  row.AuthorID,
			Content:   row.Content,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
		if row.ParentCommentID.Valid {
			id := row.ParentCommentID.UUID
			c.ParentCommentID = &id
		}
		result = append(result, c)
	}
	return result, nil
}

func (r *commentRepository) Create(ctx context.Context, taskID, authorID uuid.UUID, content string, parentCommentID *uuid.UUID) (*domain.Comment, error) {
	var parentID uuid.NullUUID
	if parentCommentID != nil {
		parentID = uuid.NullUUID{UUID: *parentCommentID, Valid: true}
	}
	row, err := r.q.CreateComment(ctx, db.CreateCommentParams{
		TaskID:          taskID,
		AuthorID:        authorID,
		Content:         content,
		ParentCommentID: parentID,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateComment", "taskID", taskID, "authorID", authorID)
	}
	c := &domain.Comment{
		ID:        row.ID,
		TaskID:    row.TaskID,
		AuthorID:  row.AuthorID,
		Content:   row.Content,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.ParentCommentID.Valid {
		id := row.ParentCommentID.UUID
		c.ParentCommentID = &id
	}
	return c, nil
}

func (r *commentRepository) GetByID(ctx context.Context, commentID uuid.UUID) (*domain.Comment, error) {
	row, err := r.q.GetCommentByID(ctx, commentID)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetCommentByID", "commentID", commentID)
	}
	c := &domain.Comment{
		ID:        row.ID,
		TaskID:    row.TaskID,
		AuthorID:  row.AuthorID,
		Content:   row.Content,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.ParentCommentID.Valid {
		id := row.ParentCommentID.UUID
		c.ParentCommentID = &id
	}
	return c, nil
}

func (r *commentRepository) Delete(ctx context.Context, commentID uuid.UUID) error {
	return errctx.Wrap(r.q.DeleteComment(ctx, commentID), "DeleteComment", "commentID", commentID)
}
