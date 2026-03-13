package repositories

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type CommentRepository interface {
	CreateComment(ctx context.Context, taskID, authorID uuid.UUID, content string) (*domain.Comment, error)
	ListTaskComments(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error)
	AddMention(ctx context.Context, commentID, projectMemberID uuid.UUID) error
	ListMentions(ctx context.Context, commentID uuid.UUID) ([]domain.CommentMention, error)
}

type commentRepository struct {
	q *db.Queries
}

func NewCommentRepository(q *db.Queries) CommentRepository {
	return &commentRepository{q: q}
}

func (r *commentRepository) CreateComment(ctx context.Context, taskID, authorID uuid.UUID, content string) (*domain.Comment, error) {
	row, err := r.q.CreateComment(ctx, db.CreateCommentParams{
		TaskID:   taskID,
		AuthorID: authorID,
		Content:  content,
	})
	if err != nil {
		return nil, err
	}
	c := domain.Comment{
		ID:        row.ID.String(),
		TaskID:    row.TaskID.String(),
		AuthorID:  row.AuthorID.String(),
		Content:   row.Content,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	return &c, nil
}

func (r *commentRepository) ListTaskComments(ctx context.Context, taskID uuid.UUID) ([]domain.Comment, error) {
	rows, err := r.q.ListTaskComments(ctx, taskID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Comment, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.Comment{
			ID:        row.ID.String(),
			TaskID:    row.TaskID.String(),
			AuthorID:  row.AuthorID.String(),
			Content:   row.Content,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return result, nil
}

func (r *commentRepository) AddMention(ctx context.Context, commentID, projectMemberID uuid.UUID) error {
	return r.q.CreateCommentMention(ctx, db.CreateCommentMentionParams{
		CommentID:       commentID,
		ProjectMemberID: projectMemberID,
	})
}

func (r *commentRepository) ListMentions(ctx context.Context, commentID uuid.UUID) ([]domain.CommentMention, error) {
	rows, err := r.q.ListCommentMentions(ctx, commentID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.CommentMention, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.CommentMention{
			ID:              row.ID.String(),
			CommentID:       row.CommentID.String(),
			ProjectMemberID: row.ProjectMemberID.String(),
		})
	}
	return result, nil
}

