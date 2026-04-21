package services

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type BoardService struct {
	repo                 repositories.BoardRepository
	columnRepo           repositories.ColumnRepository
	swimlaneRepo         repositories.SwimlaneRepository
	noteRepo             repositories.NoteRepository
	boardCustomFieldRepo repositories.BoardCustomFieldRepository
	taskRepo             repositories.TaskRepository
	conn                 *sql.DB
}

func NewBoardService(
	repo repositories.BoardRepository,
	columnRepo repositories.ColumnRepository,
	swimlaneRepo repositories.SwimlaneRepository,
	noteRepo repositories.NoteRepository,
	boardCustomFieldRepo repositories.BoardCustomFieldRepository,
	taskRepo repositories.TaskRepository,
	conn *sql.DB,
) *BoardService {
	return &BoardService{
		repo:                 repo,
		columnRepo:           columnRepo,
		swimlaneRepo:         swimlaneRepo,
		noteRepo:             noteRepo,
		boardCustomFieldRepo: boardCustomFieldRepo,
		taskRepo:             taskRepo,
		conn:                 conn,
	}
}

func (s *BoardService) ListBoards(ctx context.Context, projectID uuid.UUID) ([]domain.Board, error) {
	return s.repo.ListProjectBoards(ctx, projectID)
}

func (s *BoardService) CreateBoard(ctx context.Context, projectID uuid.UUID, projectType, name, description string, order int16, priorityType, estimationUnit, swimlaneGroupBy string) (*domain.Board, error) {
	var desc *string
	if description != "" {
		desc = &description
	}

	// Apply type-specific base structure
	if projectType == "kanban" {
		priorityType = "service_class"
		estimationUnit = "time"
		swimlaneGroupBy = domain.SystemBoardFieldIDs["priority"].String()
	} else {
		priorityType = "priority"
		estimationUnit = "story_points"
		swimlaneGroupBy = ""
	}

	// Auto-compute order if not provided
	if order == 0 {
		boards, _ := s.repo.ListProjectBoards(ctx, projectID)
		order = int16(len(boards) + 1)
	}

	pid := projectID
	board := &domain.Board{
		ProjectID:       &pid,
		Name:            name,
		Description:     desc,
		Order:           order,
		PriorityType:    priorityType,
		EstimationUnit:  estimationUnit,
		SwimlaneGroupBy: swimlaneGroupBy,
		PriorityOptions: defaultPriorityOptions(priorityType),
	}
	created, err := s.repo.CreateBoard(ctx, board)
	if err != nil {
		return nil, err
	}

	// Create default columns
	defaults := repositories.DefaultColumns[projectType]
	for i, d := range defaults {
		st := domain.SystemStatusType(d.SystemType)
		_, err := s.columnRepo.Create(ctx, &domain.Column{
			BoardID:    created.ID,
			Name:       d.Name,
			SystemType: &st,
			Order:      int16(i + 1),
			IsLocked:   d.IsLocked,
		})
		if err != nil {
			return nil, err
		}
	}

	// Create default swimlanes when grouped by the priority field
	if swimlaneGroupBy == domain.SystemBoardFieldIDs["priority"].String() {
		for _, pt := range repositories.PriorityTypes {
			if pt.Key == priorityType {
				for i, swName := range pt.DefaultValues {
					_, err := s.swimlaneRepo.Create(ctx, &domain.Swimlane{
						BoardID: created.ID,
						Name:    swName,
						Order:   int16(i + 1),
					})
					if err != nil {
						return nil, err
					}
				}
				break
			}
		}
	}

	return created, nil
}

func (s *BoardService) GetBoard(ctx context.Context, id uuid.UUID) (*domain.Board, error) {
	return s.repo.GetBoardByID(ctx, id)
}

func (s *BoardService) UpdateBoard(ctx context.Context, b *domain.Board) (*domain.Board, error) {
	// Зачем транзакция: при смене priority_type или несовместимой подмене
	// priority_options (PATCH /boards/{id}) tasks.priority у задач доски
	// должен обнуляться вместе с апдейтом самой доски — иначе в БД повиснут
	// строковые значения из старого каталога, которые потом «всплывают» при
	// возврате прежнего типа.
	current, err := s.repo.GetBoardByID(ctx, b.ID)
	if err != nil {
		return nil, err
	}
	typeChanged := current.PriorityType != b.PriorityType
	optionsChanged := !stringSliceEqual(current.PriorityOptions, b.PriorityOptions)

	return repositories.InTxT(ctx, s.conn, func(qtx *db.Queries) (*domain.Board, error) {
		txBoardRepo := repositories.NewBoardRepository(qtx)
		txTaskRepo := repositories.NewTaskRepository(qtx)

		if b.IsDefault && b.ProjectID != nil {
			_ = txBoardRepo.UnsetDefaultBoardByProjectID(ctx, *b.ProjectID)
		}
		updated, err := txBoardRepo.UpdateBoard(ctx, b)
		if err != nil {
			return nil, err
		}

		switch {
		case typeChanged:
			// Полная несовместимость каталогов: все приоритеты на задачах доски
			// сбрасываем в NULL.
			if err := txTaskRepo.ClearPriorityByBoard(ctx, b.ID); err != nil {
				return nil, err
			}
		case optionsChanged && len(b.PriorityOptions) == 0:
			// Опции вычищены целиком — любой сохранённый приоритет становится
			// «сиротой», сбрасываем.
			if err := txTaskRepo.ClearPriorityByBoard(ctx, b.ID); err != nil {
				return nil, err
			}
		case optionsChanged:
			// Частичное изменение списка опций: обнуляем только те задачи,
			// чьё значение priority больше не входит в новый набор.
			if err := txTaskRepo.ClearPriorityByBoardNotIn(ctx, b.ID, b.PriorityOptions); err != nil {
				return nil, err
			}
		}
		return updated, nil
	})
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (s *BoardService) DeleteBoard(ctx context.Context, id uuid.UUID) error {
	// Get board before deletion to know the project.
	board, err := s.repo.GetBoardByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteBoard(ctx, id); err != nil {
		return err
	}

	// Recompact board orders for the project.
	if board.ProjectID != nil {
		boards, _ := s.repo.ListProjectBoards(ctx, *board.ProjectID)
		sort.Slice(boards, func(i, j int) bool { return boards[i].Order < boards[j].Order })
		for i, b := range boards {
			newOrder := int16(i + 1)
			if b.Order != newOrder {
				_ = s.repo.UpdateBoardOrder(ctx, b.ID, newOrder)
			}
		}
	}
	return nil
}

func (s *BoardService) ListColumns(ctx context.Context, boardID uuid.UUID) ([]domain.Column, error) {
	return s.columnRepo.List(ctx, boardID)
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

	// Completed columns cannot have WIP limits.
	if *systemType == domain.StatusCompleted && wipLimit != nil {
		return nil, domain.ErrCompletedColumnWip
	}

	// Splice-insert: shift existing columns with order >= target by +1.
	existing, err := s.columnRepo.List(ctx, boardID)
	if err != nil {
		return nil, err
	}
	for _, c := range existing {
		if c.Order >= order {
			if err := s.columnRepo.UpdateOrder(ctx, c.ID, c.Order+1); err != nil {
				return nil, err
			}
		}
	}

	col := &domain.Column{
		BoardID:    boardID,
		Name:       name,
		SystemType: systemType,
		WipLimit:   wipLimit,
		Order:      order,
	}
	created, err := s.columnRepo.Create(ctx, col)
	if err != nil {
		return nil, err
	}

	// Recompact to ensure sequential 1,2,3,...
	if err := s.recompactColumnOrders(ctx, boardID); err != nil {
		return nil, err
	}

	// Validate column order after creation.
	columns, _ := s.columnRepo.List(ctx, boardID)
	if err := validateColumnOrderDomain(columns); err != nil {
		_ = s.columnRepo.Delete(ctx, created.ID)
		_ = s.recompactColumnOrders(ctx, boardID)
		return nil, err
	}

	// Re-read created column to get recompacted order.
	created, _ = s.columnRepo.GetByID(ctx, created.ID)
	return created, nil
}

func (s *BoardService) UpdateColumn(ctx context.Context, c *domain.Column) (*domain.Column, error) {
	// Completed columns cannot have WIP limits.
	if c.SystemType != nil && *c.SystemType == domain.StatusCompleted && c.WipLimit != nil {
		return nil, domain.ErrCompletedColumnWip
	}
	// When changing to completed, auto-clear WIP limit.
	if c.SystemType != nil && *c.SystemType == domain.StatusCompleted {
		c.WipLimit = nil
	}

	// Save pre-update state for rollback.
	before, err := s.columnRepo.GetByID(ctx, c.ID)
	if err != nil {
		return nil, err
	}

	updated, err := s.columnRepo.Update(ctx, c)
	if err != nil {
		return nil, err
	}

	// Validate column order after update.
	columns, _ := s.columnRepo.List(ctx, c.BoardID)
	if err := validateColumnOrderDomain(columns); err != nil {
		// Rollback to pre-update state.
		_, _ = s.columnRepo.Update(ctx, before)
		return nil, err
	}

	return updated, nil
}

func (s *BoardService) DeleteColumn(ctx context.Context, id uuid.UUID) error {
	return s.columnRepo.Delete(ctx, id)
}

func (s *BoardService) ListSwimlanes(ctx context.Context, boardID uuid.UUID) ([]domain.Swimlane, error) {
	return s.swimlaneRepo.List(ctx, boardID)
}

func (s *BoardService) CreateSwimlane(ctx context.Context, boardID uuid.UUID, name string, wipLimit *int16, order int16) (*domain.Swimlane, error) {
	sw := &domain.Swimlane{
		BoardID:  boardID,
		Name:     name,
		WipLimit: wipLimit,
		Order:    order,
	}
	return s.swimlaneRepo.Create(ctx, sw)
}

func (s *BoardService) UpdateSwimlane(ctx context.Context, sw *domain.Swimlane) (*domain.Swimlane, error) {
	return s.swimlaneRepo.Update(ctx, sw)
}

func (s *BoardService) DeleteSwimlane(ctx context.Context, id uuid.UUID) error {
	return s.swimlaneRepo.Delete(ctx, id)
}

func (s *BoardService) ListNotes(ctx context.Context, boardID uuid.UUID) ([]domain.Note, error) {
	return s.noteRepo.List(ctx, boardID)
}

func (s *BoardService) CreateNoteForColumn(ctx context.Context, columnID uuid.UUID, content string) (*domain.Note, error) {
	cid := columnID
	n := &domain.Note{
		ColumnID: &cid,
		Content:  content,
	}
	return s.noteRepo.CreateForColumn(ctx, n)
}

func (s *BoardService) CreateNoteForSwimlane(ctx context.Context, swimlaneID uuid.UUID, content string) (*domain.Note, error) {
	sid := swimlaneID
	n := &domain.Note{
		SwimlaneID: &sid,
		Content:    content,
	}
	return s.noteRepo.CreateForSwimlane(ctx, n)
}

func (s *BoardService) UpdateNote(ctx context.Context, n *domain.Note) (*domain.Note, error) {
	return s.noteRepo.Update(ctx, n)
}

func (s *BoardService) DeleteNote(ctx context.Context, id uuid.UUID) error {
	return s.noteRepo.Delete(ctx, id)
}

func (s *BoardService) GetNoteByID(ctx context.Context, id uuid.UUID) (*domain.Note, error) {
	return s.noteRepo.GetByID(ctx, id)
}

// --- Column extended ---

func (s *BoardService) GetColumnByID(ctx context.Context, id uuid.UUID) (*domain.Column, error) {
	return s.columnRepo.GetByID(ctx, id)
}

func (s *BoardService) DeleteColumnSafe(ctx context.Context, id uuid.UUID) error {
	col, err := s.columnRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	count, err := s.columnRepo.CountTasks(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return domain.ErrColumnHasTasks
	}

	// Validate: board must keep at least one column of each required type.
	columns, err := s.columnRepo.List(ctx, col.BoardID)
	if err != nil {
		return err
	}
	if col.SystemType != nil {
		if err := validateMinColumnTypes(columns, id); err != nil {
			return err
		}
	}

	if err := s.columnRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Recompact orders after deletion.
	return s.recompactColumnOrders(ctx, col.BoardID)
}

func (s *BoardService) ReorderColumns(ctx context.Context, boardID uuid.UUID, orders map[uuid.UUID]int16) error {
	// Save pre-reorder state for rollback.
	columns, err := s.columnRepo.List(ctx, boardID)
	if err != nil {
		return err
	}

	// Check locked columns.
	for _, col := range columns {
		if col.IsLocked {
			if newOrder, ok := orders[col.ID]; ok && newOrder != col.Order {
				return fmt.Errorf("COLUMN_LOCKED: %w", domain.ErrColumnLocked)
			}
		}
	}

	for id, order := range orders {
		if err := s.columnRepo.UpdateOrder(ctx, id, order); err != nil {
			return err
		}
	}

	// Validate after reorder.
	updated, _ := s.columnRepo.List(ctx, boardID)
	if err := validateColumnOrderDomain(updated); err != nil {
		// Rollback.
		for _, col := range columns {
			_ = s.columnRepo.UpdateOrder(ctx, col.ID, col.Order)
		}
		return err
	}

	return nil
}

// --- Swimlane extended ---

func (s *BoardService) GetSwimlaneByID(ctx context.Context, id uuid.UUID) (*domain.Swimlane, error) {
	return s.swimlaneRepo.GetByID(ctx, id)
}

func (s *BoardService) DeleteSwimlaneSafe(ctx context.Context, id uuid.UUID) error {
	if err := s.swimlaneRepo.ClearFromTasks(ctx, id); err != nil {
		return err
	}
	return s.swimlaneRepo.Delete(ctx, id)
}

func (s *BoardService) ReorderSwimlanes(ctx context.Context, boardID uuid.UUID, orders map[uuid.UUID]int16) error {
	for id, order := range orders {
		if err := s.swimlaneRepo.UpdateOrder(ctx, id, order); err != nil {
			return err
		}
	}
	return nil
}

// --- Board reorder ---

func (s *BoardService) ReorderBoards(ctx context.Context, orders map[uuid.UUID]int16) error {
	for id, order := range orders {
		if err := s.repo.UpdateBoardOrder(ctx, id, order); err != nil {
			return err
		}
	}
	return nil
}

// --- Custom Fields ---

func (s *BoardService) ListCustomFields(ctx context.Context, boardID uuid.UUID) ([]domain.BoardCustomField, error) {
	return s.boardCustomFieldRepo.List(ctx, boardID)
}

func (s *BoardService) CreateCustomField(ctx context.Context, boardID uuid.UUID, name, fieldType string, isRequired bool, options []string) (*domain.BoardCustomField, error) {
	// Таблица board_fields хранит только кастомные поля (системные — виртуальные,
	// генерируются из catalog.DefaultBoardFields). Поэтому любой приходящий сюда
	// is_required=true означает попытку сделать кастомное поле обязательным,
	// что запрещено политикой.
	if isRequired {
		return nil, domain.ErrRequiredCustomFieldNotAllowed
	}
	f := &domain.BoardCustomField{
		BoardID:    boardID,
		Name:       name,
		FieldType:  fieldType,
		IsSystem:   false,
		IsRequired: false,
		Options:    options,
	}
	return s.boardCustomFieldRepo.Create(ctx, f)
}

func (s *BoardService) UpdateCustomField(ctx context.Context, boardID, fieldID uuid.UUID, name *string, isRequired *bool, options []string) (*domain.BoardCustomField, error) {
	existing, err := s.boardCustomFieldRepo.GetByID(ctx, fieldID)
	if err != nil {
		return nil, err
	}
	if existing.BoardID != boardID {
		return nil, domain.ErrNotFound
	}

	// Системные поля: запрещено менять name, isRequired.
	// Исключение: «Приоритизация» и «Оценка трудозатрат» — можно менять options.
	if existing.IsSystem {
		configurable := existing.Name == "Приоритизация" || existing.Name == "Оценка трудозатрат"
		if name != nil || isRequired != nil {
			return nil, domain.ErrSystemField
		}
		if !configurable && options != nil {
			return nil, domain.ErrSystemField
		}
	}

	// Кастомное поле не может стать обязательным (политика — обязательными
	// бывают только системные поля).
	if !existing.IsSystem && isRequired != nil && *isRequired {
		return nil, domain.ErrRequiredCustomFieldNotAllowed
	}

	finalName := existing.Name
	if name != nil {
		finalName = *name
	}
	finalRequired := existing.IsRequired
	if isRequired != nil {
		finalRequired = *isRequired
	}
	finalOptions := existing.Options
	if options != nil {
		finalOptions = options
	}

	return s.boardCustomFieldRepo.Update(ctx, &domain.BoardCustomField{
		ID:         existing.ID,
		Name:       finalName,
		IsRequired: finalRequired,
		Options:    finalOptions,
	})
}

func (s *BoardService) DeleteCustomField(ctx context.Context, boardID, fieldID uuid.UUID) error {
	existing, err := s.boardCustomFieldRepo.GetByID(ctx, fieldID)
	if err != nil {
		return err
	}
	if existing.BoardID != boardID {
		return domain.ErrNotFound
	}
	if existing.IsSystem {
		return domain.ErrSystemField
	}
	return s.boardCustomFieldRepo.Delete(ctx, fieldID)
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
func (s *BoardService) recompactColumnOrders(ctx context.Context, boardID uuid.UUID) error {
	columns, err := s.columnRepo.List(ctx, boardID)
	if err != nil {
		return err
	}
	sort.Slice(columns, func(i, j int) bool { return columns[i].Order < columns[j].Order })
	for i, col := range columns {
		newOrder := int16(i + 1)
		if col.Order != newOrder {
			if err := s.columnRepo.UpdateOrder(ctx, col.ID, newOrder); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateMinColumnTypes checks that removing a column won't leave the board without
// at least one initial, one in_progress, and one completed column.
func validateMinColumnTypes(columns []domain.Column, excludeID uuid.UUID) error {
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

func defaultPriorityOptions(priorityType string) []string {
	for _, pt := range repositories.PriorityTypes {
		if pt.Key == priorityType {
			return pt.DefaultValues
		}
	}
	return nil
}
