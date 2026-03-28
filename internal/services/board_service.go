package services

import (
	"context"

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

	col := &domain.Column{
		BoardID:    boardID.String(),
		Name:       name,
		SystemType: systemType,
		WipLimit:   wipLimit,
		Order:      order,
	}
	return s.repo.CreateColumn(ctx, col)
}

func (s *BoardService) UpdateColumn(ctx context.Context, c *domain.Column) (*domain.Column, error) {
	return s.repo.UpdateColumn(ctx, c)
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
	count, err := s.repo.CountTasksInColumn(ctx, id.String())
	if err != nil {
		return err
	}
	if count > 0 {
		return domain.ErrColumnHasTasks
	}
	return s.repo.DeleteColumn(ctx, id.String())
}

func (s *BoardService) ReorderColumns(ctx context.Context, boardID uuid.UUID, orders map[uuid.UUID]int16) error {
	for id, order := range orders {
		if err := s.repo.UpdateColumnOrder(ctx, id.String(), order); err != nil {
			return err
		}
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


