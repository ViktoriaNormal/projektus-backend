package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
)

type BoardRepository interface {
	CreateBoard(ctx context.Context, b *domain.Board) (*domain.Board, error)
	GetBoardByID(ctx context.Context, id string) (*domain.Board, error)
	ListProjectBoards(ctx context.Context, projectID string) ([]domain.Board, error)
	UpdateBoard(ctx context.Context, b *domain.Board) (*domain.Board, error)
	DeleteBoard(ctx context.Context, id string) error
	UpdateBoardOrder(ctx context.Context, id string, order int16) error
	UnsetDefaultBoardByProjectID(ctx context.Context, projectID string) error

	ListColumns(ctx context.Context, boardID string) ([]domain.Column, error)
	GetColumnByID(ctx context.Context, id string) (*domain.Column, error)
	CreateColumn(ctx context.Context, c *domain.Column) (*domain.Column, error)
	UpdateColumn(ctx context.Context, c *domain.Column) (*domain.Column, error)
	DeleteColumn(ctx context.Context, id string) error
	UpdateColumnOrder(ctx context.Context, id string, order int16) error
	CountTasksInColumn(ctx context.Context, id string) (int, error)

	ListSwimlanes(ctx context.Context, boardID string) ([]domain.Swimlane, error)
	GetSwimlaneByID(ctx context.Context, id string) (*domain.Swimlane, error)
	CreateSwimlane(ctx context.Context, s *domain.Swimlane) (*domain.Swimlane, error)
	UpdateSwimlane(ctx context.Context, s *domain.Swimlane) (*domain.Swimlane, error)
	DeleteSwimlane(ctx context.Context, id string) error
	UpdateSwimlaneOrder(ctx context.Context, id string, order int16) error
	CountTasksInSwimlane(ctx context.Context, id string) (int, error)

	ListNotes(ctx context.Context, boardID string) ([]domain.Note, error)
	GetNoteByID(ctx context.Context, id string) (*domain.Note, error)
	CreateNoteForColumn(ctx context.Context, n *domain.Note) (*domain.Note, error)
	CreateNoteForSwimlane(ctx context.Context, n *domain.Note) (*domain.Note, error)
	UpdateNote(ctx context.Context, n *domain.Note) (*domain.Note, error)
	DeleteNote(ctx context.Context, id string) error

	// Custom fields
	ListCustomFields(ctx context.Context, boardID string) ([]domain.BoardCustomField, error)
	GetCustomFieldByID(ctx context.Context, id string) (*domain.BoardCustomField, error)
	CreateCustomField(ctx context.Context, f *domain.BoardCustomField) (*domain.BoardCustomField, error)
	UpdateCustomField(ctx context.Context, f *domain.BoardCustomField) (*domain.BoardCustomField, error)
	DeleteCustomField(ctx context.Context, id string) error

}

type boardRepository struct {
	q *db.Queries
}

func NewBoardRepository(q *db.Queries) BoardRepository {
	return &boardRepository{q: q}
}

func (r *boardRepository) CreateBoard(ctx context.Context, b *domain.Board) (*domain.Board, error) {
	var projectID uuid.NullUUID
	if b.ProjectID != nil {
		if id, err := uuid.Parse(*b.ProjectID); err == nil {
			projectID = uuid.NullUUID{UUID: id, Valid: true}
		}
	}
	var templateID uuid.NullUUID
	if b.TemplateID != nil {
		if id, err := uuid.Parse(*b.TemplateID); err == nil {
			templateID = uuid.NullUUID{UUID: id, Valid: true}
		}
	}
	desc := sql.NullString{}
	if b.Description != nil {
		desc = sql.NullString{String: *b.Description, Valid: true}
	}
	row, err := r.q.CreateBoard(ctx, db.CreateBoardParams{
		ProjectID:       projectID,
		TemplateID:      templateID,
		Name:            b.Name,
		Description:     desc,
		IsDefault:       b.IsDefault,
		SortOrder:       b.Order,
		PriorityType:    b.PriorityType,
		EstimationUnit:  b.EstimationUnit,
		SwimlaneGroupBy: b.SwimlaneGroupBy,
		PriorityOptions: OptionsToJSON(b.PriorityOptions),
	})
	if err != nil {
		return nil, err
	}
	dbBoard := mapBoardRowToDomain(row.ID, row.ProjectID, row.TemplateID, row.Name, row.Description, row.IsDefault, row.SortOrder, row.PriorityType, row.EstimationUnit, row.SwimlaneGroupBy, row.PriorityOptions)
	return &dbBoard, nil
}

func (r *boardRepository) GetBoardByID(ctx context.Context, id string) (*domain.Board, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetBoardByID(ctx, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	b := mapBoardRowToDomain(row.ID, row.ProjectID, row.TemplateID, row.Name, row.Description, row.IsDefault, row.SortOrder, row.PriorityType, row.EstimationUnit, row.SwimlaneGroupBy, row.PriorityOptions)
	return &b, nil
}

func (r *boardRepository) ListProjectBoards(ctx context.Context, projectID string) ([]domain.Board, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListProjectBoards(ctx, uuid.NullUUID{UUID: pid, Valid: true})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Board, len(rows))
	for i, row := range rows {
		result[i] = mapBoardRowToDomain(row.ID, row.ProjectID, row.TemplateID, row.Name, row.Description, row.IsDefault, row.SortOrder, row.PriorityType, row.EstimationUnit, row.SwimlaneGroupBy, row.PriorityOptions)
	}
	return result, nil
}

func (r *boardRepository) UpdateBoard(ctx context.Context, b *domain.Board) (*domain.Board, error) {
	id, err := uuid.Parse(b.ID)
	if err != nil {
		return nil, err
	}
	params := db.UpdateBoardParams{ID: id}
	if b.Name != "" {
		params.Name = sql.NullString{String: b.Name, Valid: true}
	}
	if b.Description != nil {
		params.Description = sql.NullString{String: *b.Description, Valid: true}
	}
	params.SortOrder = sql.NullInt16{Int16: b.Order, Valid: true}
	if b.PriorityType != "" {
		params.PriorityType = sql.NullString{String: b.PriorityType, Valid: true}
	}
	if b.EstimationUnit != "" {
		params.EstimationUnit = sql.NullString{String: b.EstimationUnit, Valid: true}
	}
	params.SwimlaneGroupBy = sql.NullString{String: b.SwimlaneGroupBy, Valid: true}
	params.IsDefault = sql.NullBool{Bool: b.IsDefault, Valid: true}
	if len(b.PriorityOptions) > 0 {
		params.PriorityOptions = OptionsToJSON(b.PriorityOptions)
	}

	row, err := r.q.UpdateBoard(ctx, params)
	if err != nil {
		return nil, err
	}
	updated := mapBoardRowToDomain(row.ID, row.ProjectID, row.TemplateID, row.Name, row.Description, row.IsDefault, row.SortOrder, row.PriorityType, row.EstimationUnit, row.SwimlaneGroupBy, row.PriorityOptions)
	return &updated, nil
}

func (r *boardRepository) DeleteBoard(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteBoard(ctx, uid)
}

func (r *boardRepository) UnsetDefaultBoardByProjectID(ctx context.Context, projectID string) error {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return err
	}
	return r.q.UnsetDefaultBoardByProjectID(ctx, uuid.NullUUID{UUID: pid, Valid: true})
}

func (r *boardRepository) ListColumns(ctx context.Context, boardID string) ([]domain.Column, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListBoardColumns(ctx, bid)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Column, len(rows))
	for i, row := range rows {
		result[i] = mapDBColumnToDomain(row)
	}
	return result, nil
}

func (r *boardRepository) CreateColumn(ctx context.Context, c *domain.Column) (*domain.Column, error) {
	bid, err := uuid.Parse(c.BoardID)
	if err != nil {
		return nil, err
	}
	var systemType sql.NullString
	if c.SystemType != nil {
		systemType = sql.NullString{String: string(*c.SystemType), Valid: true}
	}
	var wipLimit sql.NullInt16
	if c.WipLimit != nil {
		wipLimit = sql.NullInt16{Int16: *c.WipLimit, Valid: true}
	}
	row, err := r.q.CreateColumn(ctx, db.CreateColumnParams{
		BoardID:    bid,
		Name:       c.Name,
		SystemType: systemType,
		WipLimit:   wipLimit,
		SortOrder:  c.Order,
		IsLocked:   c.IsLocked,
	})
	if err != nil {
		return nil, err
	}
	created := mapDBColumnToDomain(row)
	return &created, nil
}

func (r *boardRepository) UpdateColumn(ctx context.Context, c *domain.Column) (*domain.Column, error) {
	id, err := uuid.Parse(c.ID)
	if err != nil {
		return nil, err
	}
	params := db.UpdateColumnParams{ID: id}
	if c.Name != "" {
		params.Name = sql.NullString{String: c.Name, Valid: true}
	}
	if c.SystemType != nil {
		params.SystemType = sql.NullString{String: string(*c.SystemType), Valid: true}
	}
	if c.WipLimit != nil {
		params.WipLimit = sql.NullInt16{Int16: *c.WipLimit, Valid: true}
	}
	params.SortOrder = sql.NullInt16{Int16: c.Order, Valid: true}
	row, err := r.q.UpdateColumn(ctx, params)
	if err != nil {
		return nil, err
	}
	updated := mapDBColumnToDomain(row)
	return &updated, nil
}

func (r *boardRepository) DeleteColumn(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteColumn(ctx, uid)
}

func (r *boardRepository) ListSwimlanes(ctx context.Context, boardID string) ([]domain.Swimlane, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListBoardSwimlanes(ctx, bid)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Swimlane, len(rows))
	for i, row := range rows {
		result[i] = mapDBSwimlaneToDomain(row)
	}
	return result, nil
}

func (r *boardRepository) CreateSwimlane(ctx context.Context, s *domain.Swimlane) (*domain.Swimlane, error) {
	bid, err := uuid.Parse(s.BoardID)
	if err != nil {
		return nil, err
	}
	var wipLimit sql.NullInt16
	if s.WipLimit != nil {
		wipLimit = sql.NullInt16{Int16: *s.WipLimit, Valid: true}
	}
	row, err := r.q.CreateSwimlane(ctx, db.CreateSwimlaneParams{
		BoardID:   bid,
		Name:      s.Name,
		WipLimit:  wipLimit,
		SortOrder: s.Order,
	})
	if err != nil {
		return nil, err
	}
	created := mapDBSwimlaneToDomain(row)
	return &created, nil
}

func (r *boardRepository) UpdateSwimlane(ctx context.Context, s *domain.Swimlane) (*domain.Swimlane, error) {
	id, err := uuid.Parse(s.ID)
	if err != nil {
		return nil, err
	}
	params := db.UpdateSwimlaneParams{ID: id}
	if s.Name != "" {
		params.Name = sql.NullString{String: s.Name, Valid: true}
	}
	if s.WipLimit != nil {
		params.WipLimit = sql.NullInt16{Int16: *s.WipLimit, Valid: true}
	}
	params.SortOrder = sql.NullInt16{Int16: s.Order, Valid: true}
	row, err := r.q.UpdateSwimlane(ctx, params)
	if err != nil {
		return nil, err
	}
	updated := mapDBSwimlaneToDomain(row)
	return &updated, nil
}

func (r *boardRepository) DeleteSwimlane(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteSwimlane(ctx, uid)
}

func (r *boardRepository) ListNotes(ctx context.Context, boardID string) ([]domain.Note, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListBoardNotes(ctx, bid)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Note, len(rows))
	for i, row := range rows {
		result[i] = mapDBNoteToDomain(row)
	}
	return result, nil
}

func (r *boardRepository) CreateNoteForColumn(ctx context.Context, n *domain.Note) (*domain.Note, error) {
	if n.ColumnID == nil {
		return nil, domain.ErrInvalidInput
	}
	cid, err := uuid.Parse(*n.ColumnID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateNoteForColumn(ctx, db.CreateNoteForColumnParams{
		ColumnID: uuid.NullUUID{UUID: cid, Valid: true},
		Content:  n.Content,
	})
	if err != nil {
		return nil, err
	}
	created := mapDBNoteToDomain(row)
	return &created, nil
}

func (r *boardRepository) CreateNoteForSwimlane(ctx context.Context, n *domain.Note) (*domain.Note, error) {
	if n.SwimlaneID == nil {
		return nil, domain.ErrInvalidInput
	}
	sid, err := uuid.Parse(*n.SwimlaneID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateNoteForSwimlane(ctx, db.CreateNoteForSwimlaneParams{
		SwimlaneID: uuid.NullUUID{UUID: sid, Valid: true},
		Content:    n.Content,
	})
	if err != nil {
		return nil, err
	}
	created := mapDBNoteToDomain(row)
	return &created, nil
}

func (r *boardRepository) UpdateNote(ctx context.Context, n *domain.Note) (*domain.Note, error) {
	id, err := uuid.Parse(n.ID)
	if err != nil {
		return nil, err
	}
	params := db.UpdateNoteParams{
		ID:      id,
		Content: sql.NullString{String: n.Content, Valid: true},
	}
	row, err := r.q.UpdateNote(ctx, params)
	if err != nil {
		return nil, err
	}
	updated := mapDBNoteToDomain(row)
	return &updated, nil
}

func (r *boardRepository) DeleteNote(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteNote(ctx, uid)
}

func (r *boardRepository) UpdateBoardOrder(ctx context.Context, id string, order int16) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.UpdateBoardOrder(ctx, db.UpdateBoardOrderParams{ID: uid, SortOrder: order})
}

func (r *boardRepository) GetColumnByID(ctx context.Context, id string) (*domain.Column, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetColumnByID(ctx, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	c := mapDBColumnToDomain(row)
	return &c, nil
}

func (r *boardRepository) UpdateColumnOrder(ctx context.Context, id string, order int16) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.UpdateColumnOrder(ctx, db.UpdateColumnOrderParams{ID: uid, SortOrder: order})
}

func (r *boardRepository) CountTasksInColumn(ctx context.Context, id string) (int, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return 0, err
	}
	count, err := r.q.CountTasksInColumn(ctx, uuid.NullUUID{UUID: uid, Valid: true})
	return int(count), err
}

func (r *boardRepository) GetSwimlaneByID(ctx context.Context, id string) (*domain.Swimlane, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetSwimlaneByID(ctx, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	s := mapDBSwimlaneToDomain(row)
	return &s, nil
}

func (r *boardRepository) UpdateSwimlaneOrder(ctx context.Context, id string, order int16) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.UpdateSwimlaneOrder(ctx, db.UpdateSwimlaneOrderParams{ID: uid, SortOrder: order})
}

func (r *boardRepository) CountTasksInSwimlane(ctx context.Context, id string) (int, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return 0, err
	}
	count, err := r.q.CountTasksInSwimlane(ctx, uuid.NullUUID{UUID: uid, Valid: true})
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r *boardRepository) GetNoteByID(ctx context.Context, id string) (*domain.Note, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetNoteByID(ctx, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	n := mapDBNoteToDomain(row)
	return &n, nil
}

// --- Custom fields ---

func (r *boardRepository) ListCustomFields(ctx context.Context, boardID string) ([]domain.BoardCustomField, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListBoardCustomFields(ctx, bid)
	if err != nil {
		return nil, err
	}
	result := make([]domain.BoardCustomField, len(rows))
	for i, row := range rows {
		result[i] = domain.BoardCustomField{
			ID: row.ID.String(), BoardID: row.BoardID.String(), Name: row.Name,
			FieldType: row.FieldType, IsSystem: false,
			IsRequired: row.IsRequired, Options: JSONToOptions(row.Options),
		}
	}
	return result, nil
}

func (r *boardRepository) GetCustomFieldByID(ctx context.Context, id string) (*domain.BoardCustomField, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetBoardCustomFieldByID(ctx, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	f := domain.BoardCustomField{
		ID: row.ID.String(), BoardID: row.BoardID.String(), Name: row.Name,
		FieldType: row.FieldType, IsSystem: false,
		IsRequired: row.IsRequired, Options: JSONToOptions(row.Options),
	}
	return &f, nil
}

func (r *boardRepository) CreateCustomField(ctx context.Context, f *domain.BoardCustomField) (*domain.BoardCustomField, error) {
	bid, err := uuid.Parse(f.BoardID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateBoardCustomField(ctx, db.CreateBoardCustomFieldParams{
		BoardID:    bid,
		Name:       f.Name,
		FieldType:  f.FieldType,
		IsRequired: f.IsRequired,
		Options:    OptionsToJSON(f.Options),
	})
	if err != nil {
		return nil, err
	}
	created := domain.BoardCustomField{
		ID: row.ID.String(), BoardID: row.BoardID.String(), Name: row.Name,
		FieldType: row.FieldType, IsSystem: false,
		IsRequired: row.IsRequired, Options: JSONToOptions(row.Options),
	}
	return &created, nil
}

func (r *boardRepository) UpdateCustomField(ctx context.Context, f *domain.BoardCustomField) (*domain.BoardCustomField, error) {
	uid, err := uuid.Parse(f.ID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateBoardCustomField(ctx, db.UpdateBoardCustomFieldParams{
		ID:         uid,
		Name:       f.Name,
		IsRequired: f.IsRequired,
		Options:    OptionsToJSON(f.Options),
	})
	if err != nil {
		return nil, err
	}
	updated := domain.BoardCustomField{
		ID: row.ID.String(), BoardID: row.BoardID.String(), Name: row.Name,
		FieldType: row.FieldType, IsSystem: false,
		IsRequired: row.IsRequired, Options: JSONToOptions(row.Options),
	}
	return &updated, nil
}

func (r *boardRepository) DeleteCustomField(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.q.DeleteBoardCustomFieldByID(ctx, uid)
}

func mapBoardRowToDomain(id uuid.UUID, projectID, templateID uuid.NullUUID, name string, desc sql.NullString, isDefault bool, sortOrder int16, priorityType, estimationUnit, swimlaneGroupBy string, priorityOptions pqtype.NullRawMessage) domain.Board {
	var pid *string
	if projectID.Valid {
		s := projectID.UUID.String()
		pid = &s
	}
	var tid *string
	if templateID.Valid {
		s := templateID.UUID.String()
		tid = &s
	}
	var d *string
	if desc.Valid {
		v := desc.String
		d = &v
	}
	return domain.Board{
		ID:              id.String(),
		ProjectID:       pid,
		TemplateID:      tid,
		Name:            name,
		Description:     d,
		IsDefault:       isDefault,
		Order:           sortOrder,
		PriorityType:    priorityType,
		EstimationUnit:  estimationUnit,
		SwimlaneGroupBy: swimlaneGroupBy,
		PriorityOptions: JSONToOptions(priorityOptions),
	}
}

func mapDBColumnToDomain(c db.Column) domain.Column {
	var systemType *domain.SystemStatusType
	if c.SystemType.Valid {
		st := domain.SystemStatusType(c.SystemType.String)
		systemType = &st
	}
	var wip *int16
	if c.WipLimit.Valid {
		v := c.WipLimit.Int16
		wip = &v
	}
	return domain.Column{
		ID:         c.ID.String(),
		BoardID:    c.BoardID.String(),
		Name:       c.Name,
		SystemType: systemType,
		WipLimit:   wip,
		Order:      c.SortOrder,
		IsLocked:   c.IsLocked,
	}
}

func mapDBSwimlaneToDomain(s db.Swimlane) domain.Swimlane {
	var wip *int16
	if s.WipLimit.Valid {
		v := s.WipLimit.Int16
		wip = &v
	}
	return domain.Swimlane{
		ID:       s.ID.String(),
		BoardID:  s.BoardID.String(),
		Name:     s.Name,
		WipLimit: wip,
		Order:    s.SortOrder,
	}
}

func mapDBNoteToDomain(n db.Note) domain.Note {
	var columnID *string
	if n.ColumnID.Valid {
		id := n.ColumnID.UUID.String()
		columnID = &id
	}
	var swimlaneID *string
	if n.SwimlaneID.Valid {
		id := n.SwimlaneID.UUID.String()
		swimlaneID = &id
	}
	return domain.Note{
		ID:         n.ID.String(),
		ColumnID:   columnID,
		SwimlaneID: swimlaneID,
		Content:    n.Content,
	}
}
