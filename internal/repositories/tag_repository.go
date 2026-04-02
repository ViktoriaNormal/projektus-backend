package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type TagRepository interface {
	ListByBoard(ctx context.Context, boardID uuid.UUID) ([]domain.Tag, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tag, error)
	GetByBoardAndName(ctx context.Context, boardID uuid.UUID, name string) (*domain.Tag, error)
	Create(ctx context.Context, boardID uuid.UUID, name string) (*domain.Tag, error)
	Delete(ctx context.Context, id uuid.UUID) error

	ListTaskTags(ctx context.Context, taskID uuid.UUID) ([]domain.Tag, error)
	ListTagsByTaskIDs(ctx context.Context, taskIDs []uuid.UUID) (map[string][]domain.Tag, error)
	AddTagToTask(ctx context.Context, taskID, tagID uuid.UUID) error
	RemoveTagFromTask(ctx context.Context, taskID, tagID uuid.UUID) error
	RemoveAllTagsFromTask(ctx context.Context, taskID uuid.UUID) error
	CountTasksWithTag(ctx context.Context, tagID uuid.UUID) (int32, error)
}

type tagRepository struct {
	q *db.Queries
}

func NewTagRepository(q *db.Queries) TagRepository {
	return &tagRepository{q: q}
}

func mapTag(t db.Tag) domain.Tag {
	return domain.Tag{
		ID:      t.ID.String(),
		BoardID: t.BoardID.String(),
		Name:    t.Name,
	}
}

func (r *tagRepository) ListByBoard(ctx context.Context, boardID uuid.UUID) ([]domain.Tag, error) {
	rows, err := r.q.ListTagsByBoard(ctx, boardID)
	if err != nil {
		return nil, err
	}
	tags := make([]domain.Tag, len(rows))
	for i, row := range rows {
		tags[i] = mapTag(row)
	}
	return tags, nil
}

func (r *tagRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tag, error) {
	row, err := r.q.GetTagByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	t := mapTag(row)
	return &t, nil
}

func (r *tagRepository) GetByBoardAndName(ctx context.Context, boardID uuid.UUID, name string) (*domain.Tag, error) {
	row, err := r.q.GetTagByBoardAndName(ctx, db.GetTagByBoardAndNameParams{BoardID: boardID, Name: name})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	t := mapTag(row)
	return &t, nil
}

func (r *tagRepository) Create(ctx context.Context, boardID uuid.UUID, name string) (*domain.Tag, error) {
	row, err := r.q.CreateTag(ctx, db.CreateTagParams{BoardID: boardID, Name: name})
	if err != nil {
		return nil, err
	}
	t := mapTag(row)
	return &t, nil
}

func (r *tagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTag(ctx, id)
}

func (r *tagRepository) ListTaskTags(ctx context.Context, taskID uuid.UUID) ([]domain.Tag, error) {
	rows, err := r.q.ListTaskTags(ctx, taskID)
	if err != nil {
		return nil, err
	}
	tags := make([]domain.Tag, len(rows))
	for i, row := range rows {
		tags[i] = domain.Tag{ID: row.ID.String(), BoardID: row.BoardID.String(), Name: row.Name}
	}
	return tags, nil
}

func (r *tagRepository) ListTagsByTaskIDs(ctx context.Context, taskIDs []uuid.UUID) (map[string][]domain.Tag, error) {
	rows, err := r.q.ListTagsByTaskIDs(ctx, taskIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]domain.Tag, len(taskIDs))
	for _, row := range rows {
		tid := row.TaskID.String()
		result[tid] = append(result[tid], domain.Tag{
			ID:      row.ID.String(),
			BoardID: row.BoardID.String(),
			Name:    row.Name,
		})
	}
	return result, nil
}

func (r *tagRepository) AddTagToTask(ctx context.Context, taskID, tagID uuid.UUID) error {
	return r.q.AddTagToTask(ctx, db.AddTagToTaskParams{TaskID: taskID, TagID: tagID})
}

func (r *tagRepository) RemoveTagFromTask(ctx context.Context, taskID, tagID uuid.UUID) error {
	return r.q.RemoveTagFromTask(ctx, db.RemoveTagFromTaskParams{TaskID: taskID, TagID: tagID})
}

func (r *tagRepository) RemoveAllTagsFromTask(ctx context.Context, taskID uuid.UUID) error {
	return r.q.RemoveAllTagsFromTask(ctx, taskID)
}

func (r *tagRepository) CountTasksWithTag(ctx context.Context, tagID uuid.UUID) (int32, error) {
	return r.q.CountTasksWithTag(ctx, tagID)
}
