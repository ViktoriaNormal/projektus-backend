package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type NoteRepository interface {
	List(ctx context.Context, boardID uuid.UUID) ([]domain.Note, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Note, error)
	CreateForColumn(ctx context.Context, n *domain.Note) (*domain.Note, error)
	CreateForSwimlane(ctx context.Context, n *domain.Note) (*domain.Note, error)
	Update(ctx context.Context, n *domain.Note) (*domain.Note, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type noteRepository struct {
	q *db.Queries
}

func NewNoteRepository(q *db.Queries) NoteRepository {
	return &noteRepository{q: q}
}

func (r *noteRepository) List(ctx context.Context, boardID uuid.UUID) ([]domain.Note, error) {
	rows, err := r.q.ListBoardNotes(ctx, boardID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListBoardNotes", "boardID", boardID)
	}
	result := make([]domain.Note, len(rows))
	for i, row := range rows {
		result[i] = mapDBNoteToDomain(row)
	}
	return result, nil
}

func (r *noteRepository) CreateForColumn(ctx context.Context, n *domain.Note) (*domain.Note, error) {
	if n.ColumnID == nil {
		return nil, domain.ErrInvalidInput
	}
	cid := *n.ColumnID
	row, err := r.q.CreateNoteForColumn(ctx, db.CreateNoteForColumnParams{
		ColumnID: uuid.NullUUID{UUID: cid, Valid: true},
		Content:  n.Content,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateNoteForColumn", "columnID", cid)
	}
	created := mapDBNoteToDomain(row)
	return &created, nil
}

func (r *noteRepository) CreateForSwimlane(ctx context.Context, n *domain.Note) (*domain.Note, error) {
	if n.SwimlaneID == nil {
		return nil, domain.ErrInvalidInput
	}
	sid := *n.SwimlaneID
	row, err := r.q.CreateNoteForSwimlane(ctx, db.CreateNoteForSwimlaneParams{
		SwimlaneID: uuid.NullUUID{UUID: sid, Valid: true},
		Content:    n.Content,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateNoteForSwimlane", "swimlaneID", sid)
	}
	created := mapDBNoteToDomain(row)
	return &created, nil
}

func (r *noteRepository) Update(ctx context.Context, n *domain.Note) (*domain.Note, error) {
	params := db.UpdateNoteParams{
		ID:      n.ID,
		Content: sql.NullString{String: n.Content, Valid: true},
	}
	row, err := r.q.UpdateNote(ctx, params)
	if err != nil {
		return nil, errctx.Wrap(err, "UpdateNote", "id", n.ID)
	}
	updated := mapDBNoteToDomain(row)
	return &updated, nil
}

func (r *noteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return errctx.Wrap(r.q.DeleteNote(ctx, id), "DeleteNote", "id", id)
}

func (r *noteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Note, error) {
	row, err := r.q.GetNoteByID(ctx, id)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetNoteByID", "id", id)
	}
	n := mapDBNoteToDomain(row)
	return &n, nil
}

func mapDBNoteToDomain(n db.Note) domain.Note {
	return domain.Note{
		ID:         n.ID,
		ColumnID:   nullUUIDToPtr(n.ColumnID),
		SwimlaneID: nullUUIDToPtr(n.SwimlaneID),
		Content:    n.Content,
	}
}
