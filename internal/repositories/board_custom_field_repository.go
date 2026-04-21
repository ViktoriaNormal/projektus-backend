package repositories

import (
	"context"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type BoardCustomFieldRepository interface {
	List(ctx context.Context, boardID uuid.UUID) ([]domain.BoardCustomField, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.BoardCustomField, error)
	Create(ctx context.Context, f *domain.BoardCustomField) (*domain.BoardCustomField, error)
	Update(ctx context.Context, f *domain.BoardCustomField) (*domain.BoardCustomField, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type boardCustomFieldRepository struct {
	q *db.Queries
}

func NewBoardCustomFieldRepository(q *db.Queries) BoardCustomFieldRepository {
	return &boardCustomFieldRepository{q: q}
}

func (r *boardCustomFieldRepository) List(ctx context.Context, boardID uuid.UUID) ([]domain.BoardCustomField, error) {
	rows, err := r.q.ListBoardCustomFields(ctx, boardID)
	if err != nil {
		return nil, errctx.Wrap(err, "ListBoardCustomFields", "boardID", boardID)
	}
	result := make([]domain.BoardCustomField, len(rows))
	for i, row := range rows {
		result[i] = domain.BoardCustomField{
			ID: row.ID, BoardID: row.BoardID, Name: row.Name,
			FieldType: row.FieldType, IsSystem: false,
			IsRequired: row.IsRequired, Options: JSONToOptions(row.Options),
		}
	}
	return result, nil
}

func (r *boardCustomFieldRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.BoardCustomField, error) {
	row, err := r.q.GetBoardCustomFieldByID(ctx, id)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetBoardCustomFieldByID", "id", id)
	}
	f := domain.BoardCustomField{
		ID: row.ID, BoardID: row.BoardID, Name: row.Name,
		FieldType: row.FieldType, IsSystem: false,
		IsRequired: row.IsRequired, Options: JSONToOptions(row.Options),
	}
	return &f, nil
}

func (r *boardCustomFieldRepository) Create(ctx context.Context, f *domain.BoardCustomField) (*domain.BoardCustomField, error) {
	row, err := r.q.CreateBoardCustomField(ctx, db.CreateBoardCustomFieldParams{
		BoardID:    f.BoardID,
		Name:       f.Name,
		FieldType:  f.FieldType,
		IsRequired: f.IsRequired,
		Options:    OptionsToJSON(f.Options),
	})
	if err != nil {
		return nil, errctx.Wrap(err, "CreateBoardCustomField", "boardID", f.BoardID, "name", f.Name)
	}
	created := domain.BoardCustomField{
		ID: row.ID, BoardID: row.BoardID, Name: row.Name,
		FieldType: row.FieldType, IsSystem: false,
		IsRequired: row.IsRequired, Options: JSONToOptions(row.Options),
	}
	return &created, nil
}

func (r *boardCustomFieldRepository) Update(ctx context.Context, f *domain.BoardCustomField) (*domain.BoardCustomField, error) {
	row, err := r.q.UpdateBoardCustomField(ctx, db.UpdateBoardCustomFieldParams{
		ID:         f.ID,
		Name:       f.Name,
		IsRequired: f.IsRequired,
		Options:    OptionsToJSON(f.Options),
	})
	if err != nil {
		return nil, errctx.Wrap(err, "UpdateBoardCustomField", "id", f.ID)
	}
	updated := domain.BoardCustomField{
		ID: row.ID, BoardID: row.BoardID, Name: row.Name,
		FieldType: row.FieldType, IsSystem: false,
		IsRequired: row.IsRequired, Options: JSONToOptions(row.Options),
	}
	return &updated, nil
}

func (r *boardCustomFieldRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return errctx.Wrap(r.q.DeleteBoardCustomFieldByID(ctx, id), "DeleteBoardCustomFieldByID", "id", id)
}
