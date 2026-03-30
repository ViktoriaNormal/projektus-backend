package services

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type BoardService struct {
	repo repositories.BoardRepository
}

func NewBoardService(repo repositories.BoardRepository) *BoardService {
	return &BoardService{repo: repo}
}

func (s *BoardService) ListBoards(ctx context.Context, projectID uuid.UUID) ([]domain.Board, error) {
	return s.repo.ListProjectBoards(ctx, projectID.String())
}

func (s *BoardService) CreateBoard(ctx context.Context, projectID uuid.UUID, name, description string, order int16, priorityType, estimationUnit, swimlaneGroupBy string) (*domain.Board, error) {
	var desc *string
	if description != "" {
		desc = &description
	}
	if priorityType == "" {
		priorityType = "priority"
	}
	if estimationUnit == "" {
		estimationUnit = "story_points"
	}
	pid := projectID.String()
	board := &domain.Board{
		ProjectID:       &pid,
		Name:            name,
		Description:     desc,
		Order:           order,
		PriorityType:    priorityType,
		EstimationUnit:  estimationUnit,
		SwimlaneGroupBy: swimlaneGroupBy,
	}
	return s.repo.CreateBoard(ctx, board)
}

func (s *BoardService) GetBoard(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
	return s.repo.GetBoardByID(ctx, id.String())
}

func (s *BoardService) UpdateBoard(ctx context.Context, b *domain.Board) (*domain.Board, error) {
	return s.repo.UpdateBoard(ctx, b)
}

func (s *BoardService) DeleteBoard(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteBoard(ctx, id.String())
}

func (s *BoardService) ListColumns(ctx context.Context, boardID uuid.UUID) ([]domain.Column, error) {
	return s.repo.ListColumns(ctx, boardID.String())
}

func (s *BoardService) CreateColumn(ctx context.Context, boardID uuid.UUID, name string, systemType *domain.SystemStatusType, wipLimit *int16, order int16) (*domain.Column, error) {
	if systemType == nil {
		return nil, domain.ErrInvalidInput
	}
	switch *systemType {
	case domain.StatusInitial, domain.StatusInProgress, domain.StatusPaused, domain.StatusCompleted, domain.StatusCancelled:
		// ok
	default:
		return nil, domain.ErrInvalidInput
	}

	// Splice-insert: shift existing columns with order >= target by +1.
	existing, err := s.repo.ListColumns(ctx, boardID.String())
	if err != nil {
		return nil, err
	}
	for _, c := range existing {
		if c.Order >= order {
			if err := s.repo.UpdateColumnOrder(ctx, c.ID, c.Order+1); err != nil {
				return nil, err
			}
		}
	}

	col := &domain.Column{
		BoardID:    boardID.String(),
		Name:       name,
		SystemType: systemType,
		WipLimit:   wipLimit,
		Order:      order,
	}
	created, err := s.repo.CreateColumn(ctx, col)
	if err != nil {
		return nil, err
	}

	// Recompact to ensure sequential 1,2,3,...
	if err := s.recompactColumnOrders(ctx, boardID.String()); err != nil {
		return nil, err
	}

	// Validate column order after creation.
	columns, _ := s.repo.ListColumns(ctx, boardID.String())
	if err := validateColumnOrderDomain(columns); err != nil {
		_ = s.repo.DeleteColumn(ctx, created.ID)
		_ = s.recompactColumnOrders(ctx, boardID.String())
		return nil, err
	}

	// Re-read created column to get recompacted order.
	created, _ = s.repo.GetColumnByID(ctx, created.ID)
	return created, nil
}

func (s *BoardService) UpdateColumn(ctx context.Context, c *domain.Column) (*domain.Column, error) {
	// Save pre-update state for rollback.
	before, err := s.repo.GetColumnByID(ctx, c.ID)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateColumn(ctx, c)
	if err != nil {
		return nil, err
	}

	// Validate column order after update.
	columns, _ := s.repo.ListColumns(ctx, c.BoardID)
	if err := validateColumnOrderDomain(columns); err != nil {
		// Rollback to pre-update state.
		_, _ = s.repo.UpdateColumn(ctx, before)
		return nil, err
	}

	return updated, nil
}

func (s *BoardService) DeleteColumn(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteColumn(ctx, id.String())
}

func (s *BoardService) ListSwimlanes(ctx context.Context, boardID uuid.UUID) ([]domain.Swimlane, error) {
	return s.repo.ListSwimlanes(ctx, boardID.String())
}

func (s *BoardService) CreateSwimlane(ctx context.Context, boardID uuid.UUID, name string, wipLimit *int16, order int16) (*domain.Swimlane, error) {
	sw := &domain.Swimlane{
		BoardID:  boardID.String(),
		Name:     name,
		WipLimit: wipLimit,
		Order:    order,
	}
	return s.repo.CreateSwimlane(ctx, sw)
}

func (s *BoardService) UpdateSwimlane(ctx context.Context, sw *domain.Swimlane) (*domain.Swimlane, error) {
	return s.repo.UpdateSwimlane(ctx, sw)
}

func (s *BoardService) DeleteSwimlane(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteSwimlane(ctx, id.String())
}

func (s *BoardService) ListNotes(ctx context.Context, boardID uuid.UUID) ([]domain.Note, error) {
	return s.repo.ListNotes(ctx, boardID.String())
}

func (s *BoardService) CreateNoteForColumn(ctx context.Context, columnID uuid.UUID, content string) (*domain.Note, error) {
	cid := columnID.String()
	n := &domain.Note{
		ColumnID: &cid,
		Content:  content,
	}
	return s.repo.CreateNoteForColumn(ctx, n)
}

func (s *BoardService) CreateNoteForSwimlane(ctx context.Context, swimlaneID uuid.UUID, content string) (*domain.Note, error) {
	sid := swimlaneID.String()
	n := &domain.Note{
		SwimlaneID: &sid,
		Content:    content,
	}
	return s.repo.CreateNoteForSwimlane(ctx, n)
}

func (s *BoardService) UpdateNote(ctx context.Context, n *domain.Note) (*domain.Note, error) {
	return s.repo.UpdateNote(ctx, n)
}

func (s *BoardService) DeleteNote(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteNote(ctx, id.String())
}

func (s *BoardService) GetNoteByID(ctx context.Context, id uuid.UUID) (*domain.Note, error) {
	return s.repo.GetNoteByID(ctx, id.String())
}

// --- Column extended ---

func (s *BoardService) GetColumnByID(ctx context.Context, id uuid.UUID) (*domain.Column, error) {
	return s.repo.GetColumnByID(ctx, id.String())
}

func (s *BoardService) DeleteColumnSafe(ctx context.Context, id uuid.UUID) error {
	col, err := s.repo.GetColumnByID(ctx, id.String())
	if err != nil {
		return err
	}

	count, err := s.repo.CountTasksInColumn(ctx, id.String())
	if err != nil {
		return err
	}
	if count > 0 {
		return domain.ErrColumnHasTasks
	}

	// Validate: board must keep at least one column of each required type.
	columns, err := s.repo.ListColumns(ctx, col.BoardID)
	if err != nil {
		return err
	}
	if col.SystemType != nil {
		if err := validateMinColumnTypes(columns, id.String()); err != nil {
			return err
		}
	}

	if err := s.repo.DeleteColumn(ctx, id.String()); err != nil {
		return err
	}

	// Recompact orders after deletion.
	return s.recompactColumnOrders(ctx, col.BoardID)
}

func (s *BoardService) ReorderColumns(ctx context.Context, boardID uuid.UUID, orders map[uuid.UUID]int16) error {
	// Save pre-reorder state for rollback.
	columns, err := s.repo.ListColumns(ctx, boardID.String())
	if err != nil {
		return err
	}

	// Check locked columns.
	for _, col := range columns {
		if col.IsLocked {
			colID, _ := uuid.Parse(col.ID)
			if newOrder, ok := orders[colID]; ok && newOrder != col.Order {
				return fmt.Errorf("COLUMN_LOCKED: %w", domain.ErrColumnLocked)
			}
		}
	}

	for id, order := range orders {
		if err := s.repo.UpdateColumnOrder(ctx, id.String(), order); err != nil {
			return err
		}
	}

	// Validate after reorder.
	updated, _ := s.repo.ListColumns(ctx, boardID.String())
	if err := validateColumnOrderDomain(updated); err != nil {
		// Rollback.
		for _, col := range columns {
			_ = s.repo.UpdateColumnOrder(ctx, col.ID, col.Order)
		}
		return err
	}

	return nil
}

// --- Swimlane extended ---

func (s *BoardService) GetSwimlaneByID(ctx context.Context, id uuid.UUID) (*domain.Swimlane, error) {
	return s.repo.GetSwimlaneByID(ctx, id.String())
}

func (s *BoardService) DeleteSwimlaneSafe(ctx context.Context, id uuid.UUID) error {
	count, err := s.repo.CountTasksInSwimlane(ctx, id.String())
	if err != nil {
		return err
	}
	if count > 0 {
		return domain.ErrSwimlaneHasTasks
	}
	return s.repo.DeleteSwimlane(ctx, id.String())
}

func (s *BoardService) ReorderSwimlanes(ctx context.Context, boardID uuid.UUID, orders map[uuid.UUID]int16) error {
	for id, order := range orders {
		if err := s.repo.UpdateSwimlaneOrder(ctx, id.String(), order); err != nil {
			return err
		}
	}
	return nil
}

// --- Board reorder ---

func (s *BoardService) ReorderBoards(ctx context.Context, orders map[uuid.UUID]int16) error {
	for id, order := range orders {
		if err := s.repo.UpdateBoardOrder(ctx, id.String(), order); err != nil {
			return err
		}
	}
	return nil
}

// --- Custom Fields ---

func (s *BoardService) ListCustomFields(ctx context.Context, boardID uuid.UUID) ([]domain.BoardCustomField, error) {
	return s.repo.ListCustomFields(ctx, boardID.String())
}

func (s *BoardService) CreateCustomField(ctx context.Context, boardID uuid.UUID, name, fieldType string, isRequired bool, options []string) (*domain.BoardCustomField, error) {
	existing, _ := s.repo.ListCustomFields(ctx, boardID.String())
	order := int32(len(existing) + 1)
	f := &domain.BoardCustomField{
		BoardID:    boardID.String(),
		Name:       name,
		FieldType:  fieldType,
		IsSystem:   false,
		IsRequired: isRequired,
		Order:      order,
		Options:    options,
	}
	return s.repo.CreateCustomField(ctx, f)
}

func (s *BoardService) UpdateCustomField(ctx context.Context, boardID, fieldID uuid.UUID, name *string, description *string, isRequired *bool, options []string) (*domain.BoardCustomField, error) {
	existing, err := s.repo.GetCustomFieldByID(ctx, fieldID.String())
	if err != nil {
		return nil, err
	}
	if existing.BoardID != boardID.String() {
		return nil, domain.ErrNotFound
	}

	// Системные поля: запрещено менять name, description, isRequired.
	// Исключение: «Приоритизация» и «Оценка трудозатрат» — можно менять options.
	if existing.IsSystem {
		configurable := existing.Name == "Приоритизация" || existing.Name == "Оценка трудозатрат"
		if name != nil || description != nil || isRequired != nil {
			return nil, domain.ErrSystemField
		}
		if !configurable && options != nil {
			return nil, domain.ErrSystemField
		}
	}

	finalName := existing.Name
	if name != nil {
		finalName = *name
	}
	finalDesc := existing.Description
	if description != nil {
		finalDesc = *description
	}
	finalRequired := existing.IsRequired
	if isRequired != nil {
		finalRequired = *isRequired
	}
	finalOptions := existing.Options
	if options != nil {
		finalOptions = options
	}

	return s.repo.UpdateCustomField(ctx, &domain.BoardCustomField{
		ID:          existing.ID,
		Name:        finalName,
		Description: finalDesc,
		IsRequired:  finalRequired,
		Options:     finalOptions,
	})
}

func (s *BoardService) DeleteCustomField(ctx context.Context, boardID, fieldID uuid.UUID) error {
	existing, err := s.repo.GetCustomFieldByID(ctx, fieldID.String())
	if err != nil {
		return err
	}
	if existing.BoardID != boardID.String() {
		return domain.ErrNotFound
	}
	if existing.IsSystem {
		return domain.ErrSystemField
	}
	return s.repo.DeleteCustomField(ctx, fieldID.String())
}

func (s *BoardService) ReorderCustomFields(ctx context.Context, boardID uuid.UUID, orders map[uuid.UUID]int32) error {
	for id, order := range orders {
		if err := s.repo.UpdateCustomFieldOrder(ctx, id.String(), order); err != nil {
			return err
		}
	}
	return nil
}

// validateColumnOrderDomain ensures column ordering: all initial < all in_progress < all completed.
func validateColumnOrderDomain(columns []domain.Column) error {
	sorted := make([]domain.Column, len(columns))
	copy(sorted, columns)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Order < sorted[j].Order })

	typeOrder := map[domain.SystemStatusType]int{
		domain.StatusInitial:    0,
		domain.StatusInProgress: 1,
		domain.StatusCompleted:  2,
	}
	maxSeen := -1
	for _, col := range sorted {
		if col.SystemType == nil {
			continue
		}
		to, ok := typeOrder[*col.SystemType]
		if !ok {
			continue
		}
		if to < maxSeen {
			return fmt.Errorf("INVALID_COLUMN_ORDER: %w", domain.ErrInvalidColumnOrder)
		}
		if to > maxSeen {
			maxSeen = to
		}
	}
	return nil
}

// recompactColumnOrders renumbers all columns for a board sequentially: 1, 2, 3, ...
func (s *BoardService) recompactColumnOrders(ctx context.Context, boardID string) error {
	columns, err := s.repo.ListColumns(ctx, boardID)
	if err != nil {
		return err
	}
	sort.Slice(columns, func(i, j int) bool { return columns[i].Order < columns[j].Order })
	for i, col := range columns {
		newOrder := int16(i + 1)
		if col.Order != newOrder {
			if err := s.repo.UpdateColumnOrder(ctx, col.ID, newOrder); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateMinColumnTypes checks that removing a column won't leave the board without
// at least one initial, one in_progress, and one completed column.
func validateMinColumnTypes(columns []domain.Column, excludeID string) error {
	counts := map[domain.SystemStatusType]int{
		domain.StatusInitial:    0,
		domain.StatusInProgress: 0,
		domain.StatusCompleted:  0,
	}
	for _, col := range columns {
		if col.ID == excludeID || col.SystemType == nil {
			continue
		}
		if _, ok := counts[*col.SystemType]; ok {
			counts[*col.SystemType]++
		}
	}
	for st, cnt := range counts {
		if cnt < 1 {
			return fmt.Errorf("INVALID_COLUMN_ORDER: на доске должна быть хотя бы одна колонка типа %s: %w", st, domain.ErrInvalidColumnOrder)
		}
	}
	return nil
}
