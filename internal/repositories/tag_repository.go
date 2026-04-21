package repositories

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type TagRepository interface {
	ListByBoard(ctx context.Context, boardID uuid.UUID) ([]domain.Tag, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tag, error)
	GetByBoardAndName(ctx context.Context, boardID uuid.UUID, name string) (*domain.Tag, error)
	Create(ctx context.Context, boardID uuid.UUID, name string) (*domain.Tag, error)
	Delete(ctx context.Context, id uuid.UUID) error

	ListTaskTags(ctx context.Context, taskID uuid.UUID) ([]domain.Tag, error)
	ListTagsByTaskIDs(ctx context.Context, taskIDs []uuid.UUID) (map[uuid.UUID][]domain.Tag, error)
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
		ID:      t.ID,
		BoardID: t.BoardID,
		Name:    t.Name,
	}
}

func (r *tagRepository) ListByBoard(ctx context.Context, boardID uuid.UUID) ([]domain.Tag, error) {
	rows, err := r.q.ListTagsByBoard(ctx, boardID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListTagsByBoard", "boardID", boardID)
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
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetTagByID", "id", id)
	}
	t := mapTag(row)
	return &t, nil
}

func (r *tagRepository) GetByBoardAndName(ctx context.Context, boardID uuid.UUID, name string) (*domain.Tag, error) {
	row, err := r.q.GetTagByBoardAndName(ctx, db.GetTagByBoardAndNameParams{BoardID: boardID, Name: name})
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetTagByBoardAndName", "boardID", boardID, "name", name)
	}
	t := mapTag(row)
	return &t, nil
}

func (r *tagRepository) Create(ctx context.Context, boardID uuid.UUID, name string) (*domain.Tag, error) {
	row, err := r.q.CreateTag(ctx, db.CreateTagParams{BoardID: boardID, Name: name})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateTag", "boardID", boardID, "name", name)
	}
	t := mapTag(row)
	return &t, nil
}

func (r *tagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return errctx.Wrap(r.q.DeleteTag(ctx, id), "DeleteTag", "id", id)
}

func (r *tagRepository) ListTaskTags(ctx context.Context, taskID uuid.UUID) ([]domain.Tag, error) {
	rows, err := r.q.ListTaskTags(ctx, taskID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListTaskTags", "taskID", taskID)
	}
	tags := make([]domain.Tag, len(rows))
	for i, row := range rows {
		tags[i] = domain.Tag{ID: row.ID, BoardID: row.BoardID, Name: row.Name}
	}
	return tags, nil
}

func (r *tagRepository) ListTagsByTaskIDs(ctx context.Context, taskIDs []uuid.UUID) (map[uuid.UUID][]domain.Tag, error) {
	rows, err := r.q.ListTagsByTaskIDs(ctx, taskIDs)
	if err != nil {
		return nil, errctx.Wrap(err, "ListTagsByTaskIDs")
	}
	result := make(map[uuid.UUID][]domain.Tag, len(taskIDs))
	for _, row := range rows {
		result[row.TaskID] = append(result[row.TaskID], domain.Tag{
			ID:      row.ID,
			BoardID: row.BoardID,
			Name:    row.Name,
		})
	}
	return result, nil
}

func (r *tagRepository) AddTagToTask(ctx context.Context, taskID, tagID uuid.UUID) error {
	return errctx.Wrap(r.q.AddTagToTask(ctx, db.AddTagToTaskParams{TaskID: taskID, TagID: tagID}), "AddTagToTask", "taskID", taskID, "tagID", tagID)
}

func (r *tagRepository) RemoveTagFromTask(ctx context.Context, taskID, tagID uuid.UUID) error {
	return errctx.Wrap(r.q.RemoveTagFromTask(ctx, db.RemoveTagFromTaskParams{TaskID: taskID, TagID: tagID}), "RemoveTagFromTask", "taskID", taskID, "tagID", tagID)
}

func (r *tagRepository) RemoveAllTagsFromTask(ctx context.Context, taskID uuid.UUID) error {
	return errctx.Wrap(r.q.RemoveAllTagsFromTask(ctx, taskID), "RemoveAllTagsFromTask", "taskID", taskID)
}

func (r *tagRepository) CountTasksWithTag(ctx context.Context, tagID uuid.UUID) (int32, error) {
	n, err := r.q.CountTasksWithTag(ctx, tagID)
	if err != nil {
		return 0, errctx.Wrap(err, "CountTasksWithTag", "tagID", tagID)
	}
	return n, nil
}
