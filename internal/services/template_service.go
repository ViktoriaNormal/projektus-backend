package services

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

type TemplateService struct {
	repo    repositories.TemplateRepository
	refRepo repositories.ReferenceRepository
}

func NewTemplateService(repo repositories.TemplateRepository, refRepo repositories.ReferenceRepository) *TemplateService {
	return &TemplateService{repo: repo, refRepo: refRepo}
}

// --- Templates ---

// TemplateFullData holds all nested data for a template response
type TemplateFullData struct {
	Boards []domain.TemplateBoard
	Params []domain.TemplateProjectParam
	Roles  []domain.TemplateRole
}

func (s *TemplateService) List(ctx context.Context) ([]domain.ProjectTemplate, []TemplateFullData, error) {
	templates, err := s.repo.List(ctx)
	if err != nil {
		return nil, nil, err
	}
	allData := make([]TemplateFullData, len(templates))
	for i, t := range templates {
		data, err := s.loadFullData(ctx, t.ID)
		if err != nil {
			return nil, nil, err
		}
		allData[i] = data
	}
	return templates, allData, nil
}

func (s *TemplateService) GetByType(ctx context.Context, projectType string) (*domain.ProjectTemplate, TemplateFullData, error) {
	tmpl, err := s.repo.GetByType(ctx, projectType)
	if err != nil {
		return nil, TemplateFullData{}, err
	}
	data, err := s.loadFullData(ctx, tmpl.ID)
	if err != nil {
		return nil, TemplateFullData{}, err
	}
	return tmpl, data, nil
}

func (s *TemplateService) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectTemplate, TemplateFullData, error) {
	tmpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, TemplateFullData{}, err
	}
	data, err := s.loadFullData(ctx, id)
	if err != nil {
		return nil, TemplateFullData{}, err
	}
	return tmpl, data, nil
}

func (s *TemplateService) Create(ctx context.Context, name string, description *string, projectType string) (*domain.ProjectTemplate, TemplateFullData, error) {
	tmpl, err := s.repo.Create(ctx, name, description, projectType)
	if err != nil {
		return nil, TemplateFullData{}, err
	}

	// Create default board
	board, err := s.createDefaultBoard(ctx, tmpl.ID, projectType)
	if err != nil {
		return nil, TemplateFullData{}, err
	}

	return tmpl, TemplateFullData{Boards: []domain.TemplateBoard{board}, Params: []domain.TemplateProjectParam{}, Roles: []domain.TemplateRole{}}, nil
}

func (s *TemplateService) Update(ctx context.Context, id uuid.UUID, name *string, description *string) (*domain.ProjectTemplate, TemplateFullData, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, TemplateFullData{}, err
	}

	finalName := existing.Name
	if name != nil {
		finalName = *name
	}
	finalDesc := existing.Description
	if description != nil {
		finalDesc = description
	}

	tmpl, err := s.repo.Update(ctx, id, finalName, finalDesc)
	if err != nil {
		return nil, TemplateFullData{}, err
	}

	data, err := s.loadFullData(ctx, id)
	if err != nil {
		return nil, TemplateFullData{}, err
	}

	return tmpl, data, nil
}

func (s *TemplateService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	inUse, err := s.repo.IsInUse(ctx, id)
	if err != nil {
		return err
	}
	if inUse {
		return fmt.Errorf("TEMPLATE_IN_USE: %w", domain.ErrTemplateInUse)
	}

	return s.repo.Delete(ctx, id)
}

// --- Boards ---

func (s *TemplateService) CreateBoard(ctx context.Context, templateID uuid.UUID, name, description string, isDefault bool, priorityType, estimationUnit string, swimlaneGroupBy *string) (domain.TemplateBoard, error) {
	tmpl, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return domain.TemplateBoard{}, err
	}

	// Apply type-specific base structure for new boards
	projectType := string(tmpl.Type)
	if projectType == "kanban" {
		priorityType = "service_class"
		estimationUnit = "time"
		sgbDefault := domain.SystemBoardFieldIDs["priority"].String()
		swimlaneGroupBy = &sgbDefault
	} else {
		priorityType = "priority"
		estimationUnit = "story_points"
		swimlaneGroupBy = nil
	}

	if isDefault {
		_ = s.repo.UnsetDefaultBoard(ctx, templateID)
	}

	count, _ := s.repo.CountBoardsByTemplateID(ctx, templateID)
	sgb := ""
	if swimlaneGroupBy != nil {
		sgb = *swimlaneGroupBy
	}

	board, err := s.repo.CreateBoard(ctx, db.CreateTemplateBoardParams{
		TemplateID:      uuid.NullUUID{UUID: templateID, Valid: true},
		Name:            name,
		Description:     sql.NullString{String: description, Valid: description != ""},
		IsDefault:       isDefault,
		SortOrder:       int16(count + 1),
		PriorityType:    priorityType,
		EstimationUnit:  estimationUnit,
		SwimlaneGroupBy: sgb,
	})
	if err != nil {
		return domain.TemplateBoard{}, err
	}

	// Create default columns
	columns, err := s.createDefaultColumns(ctx, board.ID, string(tmpl.Type))
	if err != nil {
		return domain.TemplateBoard{}, err
	}

	// Create swimlanes if grouped
	var swimlanes []domain.TemplateBoardSwimlane
	if sgb != "" {
		swimlanes, err = s.createSwimlanesForGroup(ctx, board.ID, sgb, priorityType)
		if err != nil {
			return domain.TemplateBoard{}, err
		}
	}

	// System fields are generated at runtime from Go constants, not stored in DB.

	return s.buildBoardDomain(board, columns, swimlanes, nil), nil
}

func (s *TemplateService) UpdateBoard(ctx context.Context, templateID, boardID uuid.UUID, name, description *string, isDefault *bool, order *int32, priorityType, estimationUnit *string, swimlaneGroupBy *string, clearSwimlaneGroup bool) (domain.TemplateBoard, error) {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil {
		return domain.TemplateBoard{}, domain.ErrNotFound
	}
	if !board.TemplateID.Valid || board.TemplateID.UUID != templateID {
		return domain.TemplateBoard{}, domain.ErrNotFound
	}

	if isDefault != nil && *isDefault {
		_ = s.repo.UnsetDefaultBoard(ctx, templateID)
	}

	finalName := board.Name
	if name != nil {
		finalName = *name
	}
	finalDesc := board.Description
	if description != nil {
		finalDesc = sql.NullString{String: *description, Valid: *description != ""}
	}
	finalIsDefault := board.IsDefault
	if isDefault != nil {
		finalIsDefault = *isDefault
	}
	finalOrder := board.SortOrder
	if order != nil {
		finalOrder = int16(*order)
	}
	finalPriorityType := board.PriorityType
	if priorityType != nil && *priorityType != board.PriorityType {
		finalPriorityType = *priorityType
	}
	finalEstimationUnit := board.EstimationUnit
	if estimationUnit != nil {
		finalEstimationUnit = *estimationUnit
	}
	finalSwimlaneGroupBy := board.SwimlaneGroupBy
	swimlaneGroupChanged := false
	if clearSwimlaneGroup {
		if finalSwimlaneGroupBy != "" {
			finalSwimlaneGroupBy = ""
			swimlaneGroupChanged = true
		}
	} else if swimlaneGroupBy != nil && *swimlaneGroupBy != board.SwimlaneGroupBy {
		finalSwimlaneGroupBy = *swimlaneGroupBy
		swimlaneGroupChanged = true
	}

	updated, err := s.repo.UpdateBoard(ctx, db.UpdateTemplateBoardParams{
		ID:              boardID,
		Name:            finalName,
		Description:     finalDesc,
		IsDefault:       finalIsDefault,
		SortOrder:       finalOrder,
		PriorityType:    finalPriorityType,
		EstimationUnit:  finalEstimationUnit,
		SwimlaneGroupBy: finalSwimlaneGroupBy,
	})
	if err != nil {
		return domain.TemplateBoard{}, err
	}

	// Recreate swimlanes on group change
	if swimlaneGroupChanged {
		_ = s.repo.DeleteSwimlanesByBoardID(ctx, boardID)
		if finalSwimlaneGroupBy != "" {
			_, _ = s.createSwimlanesForGroup(ctx, boardID, finalSwimlaneGroupBy, finalPriorityType)
		}
	}

	return s.loadFullBoard(ctx, updated)
}

func (s *TemplateService) DeleteBoard(ctx context.Context, templateID, boardID uuid.UUID) error {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil {
		return domain.ErrNotFound
	}
	if !board.TemplateID.Valid || board.TemplateID.UUID != templateID {
		return domain.ErrNotFound
	}

	if board.IsDefault {
		return fmt.Errorf("DEFAULT_BOARD_DELETE: %w", domain.ErrDefaultBoard)
	}

	count, _ := s.repo.CountBoardsByTemplateID(ctx, templateID)
	if count <= 1 {
		return fmt.Errorf("LAST_BOARD_DELETE: %w", domain.ErrLastBoard)
	}

	if err := s.repo.DeleteBoard(ctx, boardID); err != nil {
		return err
	}

	// Recompact board orders for the template.
	boards, _ := s.repo.ListBoardsByTemplateID(ctx, templateID)
	sort.Slice(boards, func(i, j int) bool { return boards[i].SortOrder < boards[j].SortOrder })
	for i, b := range boards {
		newOrder := int16(i + 1)
		if b.SortOrder != newOrder {
			_ = s.repo.UpdateBoardOrder(ctx, b.ID, newOrder)
		}
	}
	return nil
}

func (s *TemplateService) ReorderBoards(ctx context.Context, templateID uuid.UUID, orders map[uuid.UUID]int32) error {
	for id, order := range orders {
		if err := s.repo.UpdateBoardOrder(ctx, id, int16(order)); err != nil {
			return err
		}
	}
	return nil
}

// --- Columns ---

func (s *TemplateService) CreateColumn(ctx context.Context, templateID, boardID uuid.UUID, name, systemType string, wipLimit *int32, order int32, note string) (domain.TemplateBoardColumn, error) {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || !board.TemplateID.Valid || board.TemplateID.UUID != templateID {
		return domain.TemplateBoardColumn{}, domain.ErrNotFound
	}

	// Completed columns cannot have WIP limits.
	if systemType == "completed" && wipLimit != nil {
		return domain.TemplateBoardColumn{}, domain.ErrCompletedColumnWip
	}

	wl := sql.NullInt16{}
	if wipLimit != nil {
		wl = sql.NullInt16{Int16: int16(*wipLimit), Valid: true}
	}

	// Сдвигаем существующие колонки с order >= переданного на +1 (splice-вставка)
	columns, err := s.repo.ListColumns(ctx, boardID)
	if err != nil {
		return domain.TemplateBoardColumn{}, err
	}
	for _, c := range columns {
		if c.SortOrder >= int16(order) {
			if err := s.repo.UpdateColumnOrder(ctx, c.ID, c.SortOrder+1); err != nil {
				return domain.TemplateBoardColumn{}, err
			}
		}
	}

	col, err := s.repo.CreateColumn(ctx, db.CreateTemplateBoardColumnParams{
		BoardID:    boardID,
		Name:       name,
		SystemType: sql.NullString{String: systemType, Valid: systemType != ""},
		WipLimit:   wl,
		SortOrder:  int16(order),
		IsLocked:   false,
		Note:       note,
	})
	if err != nil {
		return domain.TemplateBoardColumn{}, err
	}

	// Recompact to ensure sequential 1,2,3,...
	_ = s.recompactTemplateColumnOrders(ctx, boardID)

	return mapDBColumnToDomain(col), nil
}

func (s *TemplateService) UpdateColumn(ctx context.Context, templateID, boardID, columnID uuid.UUID, name, systemType *string, wipLimit *int32, clearWipLimit bool, note *string, clearNote bool) (domain.TemplateBoardColumn, error) {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || !board.TemplateID.Valid || board.TemplateID.UUID != templateID {
		return domain.TemplateBoardColumn{}, domain.ErrNotFound
	}

	col, err := s.repo.GetColumnByID(ctx, columnID)
	if err != nil || col.BoardID != boardID {
		return domain.TemplateBoardColumn{}, domain.ErrNotFound
	}

	if col.IsLocked {
		return domain.TemplateBoardColumn{}, fmt.Errorf("COLUMN_LOCKED: %w", domain.ErrColumnLocked)
	}

	finalName := col.Name
	if name != nil {
		finalName = *name
	}
	finalSystemType := col.SystemType
	if systemType != nil {
		finalSystemType = sql.NullString{String: *systemType, Valid: *systemType != ""}
	}
	// Completed columns cannot have WIP limits; reject explicit set.
	effectiveSystemType := col.SystemType.String
	if systemType != nil {
		effectiveSystemType = *systemType
	}
	if effectiveSystemType == "completed" && wipLimit != nil {
		return domain.TemplateBoardColumn{}, domain.ErrCompletedColumnWip
	}

	finalWipLimit := col.WipLimit
	// When changing to completed, auto-clear WIP limit.
	if systemType != nil && *systemType == "completed" {
		finalWipLimit = sql.NullInt16{Valid: false}
	} else if clearWipLimit {
		finalWipLimit = sql.NullInt16{Valid: false}
	} else if wipLimit != nil {
		finalWipLimit = sql.NullInt16{Int16: int16(*wipLimit), Valid: true}
	}
	finalNote := col.Note
	if clearNote {
		finalNote = ""
	} else if note != nil {
		finalNote = *note
	}

	updated, err := s.repo.UpdateColumn(ctx, db.UpdateTemplateBoardColumnParams{
		ID:         columnID,
		Name:       finalName,
		SystemType: finalSystemType,
		WipLimit:   finalWipLimit,
		Note:       finalNote,
	})
	if err != nil {
		return domain.TemplateBoardColumn{}, err
	}

	// Validate column order
	columns, _ := s.repo.ListColumns(ctx, boardID)
	if err := validateColumnOrder(columns); err != nil {
		// Rollback
		_, _ = s.repo.UpdateColumn(ctx, db.UpdateTemplateBoardColumnParams{
			ID: columnID, Name: col.Name, SystemType: col.SystemType, WipLimit: col.WipLimit,
		})
		return domain.TemplateBoardColumn{}, err
	}

	return mapDBColumnToDomain(updated), nil
}

func (s *TemplateService) DeleteColumn(ctx context.Context, templateID, boardID, columnID uuid.UUID) error {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || !board.TemplateID.Valid || board.TemplateID.UUID != templateID {
		return domain.ErrNotFound
	}

	col, err := s.repo.GetColumnByID(ctx, columnID)
	if err != nil || col.BoardID != boardID {
		return domain.ErrNotFound
	}

	if col.IsLocked {
		return fmt.Errorf("COLUMN_LOCKED: %w", domain.ErrColumnLocked)
	}

	// Check minimum type requirement before deleting.
	columns, _ := s.repo.ListColumns(ctx, boardID)
	if col.SystemType.Valid {
		if err := validateMinColumnTypesDB(columns, columnID); err != nil {
			return err
		}
	}

	if err := s.repo.DeleteColumn(ctx, columnID); err != nil {
		return err
	}

	// Validate remaining order
	remaining, _ := s.repo.ListColumns(ctx, boardID)
	if err := validateColumnOrder(remaining); err != nil {
		// Re-create the column
		_, _ = s.repo.CreateColumn(ctx, db.CreateTemplateBoardColumnParams{
			BoardID: boardID, Name: col.Name, SystemType: col.SystemType, WipLimit: col.WipLimit, SortOrder: col.SortOrder, IsLocked: col.IsLocked, Note: col.Note,
		})
		return err
	}

	// Recompact orders after deletion.
	_ = s.recompactTemplateColumnOrders(ctx, boardID)

	return nil
}

func (s *TemplateService) ReorderColumns(ctx context.Context, templateID, boardID uuid.UUID, orders map[uuid.UUID]int32) error {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || !board.TemplateID.Valid || board.TemplateID.UUID != templateID {
		return domain.ErrNotFound
	}

	// Check locked columns
	columns, _ := s.repo.ListColumns(ctx, boardID)
	lockedColumns := make(map[uuid.UUID]int32)
	for _, col := range columns {
		if col.IsLocked {
			lockedColumns[col.ID] = int32(col.SortOrder)
		}
	}
	for id, newOrder := range orders {
		if origOrder, ok := lockedColumns[id]; ok && origOrder != newOrder {
			return fmt.Errorf("COLUMN_LOCKED: %w", domain.ErrColumnLocked)
		}
	}

	for id, order := range orders {
		if err := s.repo.UpdateColumnOrder(ctx, id, int16(order)); err != nil {
			return err
		}
	}

	// Validate after reorder
	updatedColumns, _ := s.repo.ListColumns(ctx, boardID)
	if err := validateColumnOrder(updatedColumns); err != nil {
		// Rollback
		for _, col := range columns {
			_ = s.repo.UpdateColumnOrder(ctx, col.ID, col.SortOrder)
		}
		return err
	}

	return nil
}

// --- Swimlanes ---

func (s *TemplateService) CreateSwimlane(ctx context.Context, templateID, boardID uuid.UUID, name string, wipLimit *int32, order int32) (domain.TemplateBoardSwimlane, error) {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || !board.TemplateID.Valid || board.TemplateID.UUID != templateID {
		return domain.TemplateBoardSwimlane{}, domain.ErrNotFound
	}

	// Reject wipLimit for Scrum templates.
	if wipLimit != nil {
		tmpl, err := s.repo.GetByID(ctx, templateID)
		if err == nil && tmpl.Type == domain.ProjectTypeScrum {
			return domain.TemplateBoardSwimlane{}, domain.ErrScrumWipNotAllowed
		}
	}

	wl := sql.NullInt16{}
	if wipLimit != nil {
		wl = sql.NullInt16{Int16: int16(*wipLimit), Valid: true}
	}

	sw, err := s.repo.CreateSwimlane(ctx, db.CreateTemplateBoardSwimlaneParams{
		BoardID:   boardID,
		Name:      name,
		WipLimit:  wl,
		SortOrder: int16(order),
		Note:      "",
	})
	if err != nil {
		return domain.TemplateBoardSwimlane{}, err
	}

	return mapDBSwimlaneToDomain(sw), nil
}

func (s *TemplateService) UpdateSwimlane(ctx context.Context, templateID, boardID, swimlaneID uuid.UUID, wipLimit *int32, clearWipLimit bool, note *string, clearNote bool) (domain.TemplateBoardSwimlane, error) {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || !board.TemplateID.Valid || board.TemplateID.UUID != templateID {
		return domain.TemplateBoardSwimlane{}, domain.ErrNotFound
	}

	// Reject wipLimit for Scrum templates.
	if wipLimit != nil {
		tmpl, err := s.repo.GetByID(ctx, templateID)
		if err == nil && tmpl.Type == domain.ProjectTypeScrum {
			return domain.TemplateBoardSwimlane{}, domain.ErrScrumWipNotAllowed
		}
	}

	existing, err := s.repo.GetSwimlaneByID(ctx, swimlaneID)
	if err != nil {
		return domain.TemplateBoardSwimlane{}, domain.ErrNotFound
	}

	wl := existing.WipLimit
	if clearWipLimit {
		wl = sql.NullInt16{Valid: false}
	} else if wipLimit != nil {
		wl = sql.NullInt16{Int16: int16(*wipLimit), Valid: true}
	}
	finalNote := existing.Note
	if clearNote {
		finalNote = ""
	} else if note != nil {
		finalNote = *note
	}

	sw, err := s.repo.UpdateSwimlane(ctx, swimlaneID, wl, finalNote)
	if err != nil {
		return domain.TemplateBoardSwimlane{}, err
	}

	return mapDBSwimlaneToDomain(sw), nil
}

func (s *TemplateService) DeleteSwimlane(ctx context.Context, templateID, boardID, swimlaneID uuid.UUID) error {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || !board.TemplateID.Valid || board.TemplateID.UUID != templateID {
		return domain.ErrNotFound
	}
	return s.repo.DeleteSwimlane(ctx, swimlaneID)
}

func (s *TemplateService) ReorderSwimlanes(ctx context.Context, templateID, boardID uuid.UUID, orders map[uuid.UUID]int32) error {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || !board.TemplateID.Valid || board.TemplateID.UUID != templateID {
		return domain.ErrNotFound
	}
	for id, order := range orders {
		if err := s.repo.UpdateSwimlaneOrder(ctx, id, int16(order)); err != nil {
			return err
		}
	}
	return nil
}

// --- Custom Fields ---

func (s *TemplateService) CreateCustomField(ctx context.Context, templateID, boardID uuid.UUID, name, fieldType string, isRequired bool, options []string) (domain.TemplateBoardCustomField, error) {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || !board.TemplateID.Valid || board.TemplateID.UUID != templateID {
		return domain.TemplateBoardCustomField{}, domain.ErrNotFound
	}

	// sprint/sprint_list доступны только для задач Scrum-досок
	if fieldType == "sprint" || fieldType == "sprint_list" {
		tmpl, err := s.repo.GetByID(ctx, templateID)
		if err != nil {
			return domain.TemplateBoardCustomField{}, err
		}
		if string(tmpl.Type) != "scrum" {
			return domain.TemplateBoardCustomField{}, domain.ErrInvalidFieldType
		}
	}

	// Проверка уникальности имени среди полей доски
	existingFields, _ := s.repo.ListCustomFields(ctx, boardID)
	for _, f := range existingFields {
		if f.Name == name {
			return domain.TemplateBoardCustomField{}, domain.ErrConflict
		}
	}

	field, err := s.repo.CreateField(ctx, db.CreateTemplateBoardFieldParams{
		BoardID:    boardID,
		Name:       name,
		FieldType:  fieldType,
		IsRequired: isRequired,
		Options:    repositories.OptionsToJSON(options),
	})
	if err != nil {
		return domain.TemplateBoardCustomField{}, err
	}

	return mapDBFieldToDomain(field), nil
}

func (s *TemplateService) UpdateCustomField(ctx context.Context, templateID, boardID, fieldID uuid.UUID, name *string, isRequired *bool, options []string) (domain.TemplateBoardCustomField, error) {
	if name != nil && strings.TrimSpace(*name) == "" {
		return domain.TemplateBoardCustomField{}, domain.ErrInvalidInput
	}
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || !board.TemplateID.Valid || board.TemplateID.UUID != templateID {
		return domain.TemplateBoardCustomField{}, domain.ErrNotFound
	}

	existing, err := s.repo.GetFieldByID(ctx, fieldID)
	if err != nil || existing.BoardID != boardID {
		return domain.TemplateBoardCustomField{}, domain.ErrNotFound
	}

	// All DB fields are custom (non-system), so no system field check needed.

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
		finalOptions = repositories.OptionsToJSON(options)
	}

	updated, err := s.repo.UpdateField(ctx, db.UpdateTemplateBoardFieldParams{
		ID:         fieldID,
		Name:       finalName,
		IsRequired: finalRequired,
		Options:    finalOptions,
	})
	if err != nil {
		return domain.TemplateBoardCustomField{}, err
	}

	return mapDBFieldToDomain(updated), nil
}

func (s *TemplateService) DeleteCustomField(ctx context.Context, templateID, boardID, fieldID uuid.UUID) error {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || !board.TemplateID.Valid || board.TemplateID.UUID != templateID {
		return domain.ErrNotFound
	}

	existing, err := s.repo.GetFieldByID(ctx, fieldID)
	if err != nil || existing.BoardID != boardID {
		return domain.ErrNotFound
	}

	return s.repo.DeleteField(ctx, fieldID)
}

// --- Project Params ---

func (s *TemplateService) CreateProjectParam(ctx context.Context, templateID uuid.UUID, name, fieldType string, isRequired bool, options []string) (domain.TemplateProjectParam, error) {
	_, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return domain.TemplateProjectParam{}, err
	}

	// sprint/sprint_list недопустимы для параметров проекта
	if fieldType == "sprint" || fieldType == "sprint_list" {
		return domain.TemplateProjectParam{}, domain.ErrInvalidFieldType
	}

	// Проверка уникальности имени среди параметров проекта
	existingParams, _ := s.repo.ListProjectParams(ctx, templateID)
	for _, p := range existingParams {
		if p.Name == name {
			return domain.TemplateProjectParam{}, domain.ErrConflict
		}
	}

	p, err := s.repo.CreateProjectParam(ctx, db.CreateTemplateProjectParamParams{
		TemplateID: uuid.NullUUID{UUID: templateID, Valid: true}, Name: name, FieldType: fieldType, IsRequired: isRequired, Options: repositories.OptionsToJSON(options),
	})
	if err != nil {
		return domain.TemplateProjectParam{}, err
	}
	return domain.TemplateProjectParam{ID: p.ID, TemplateID: p.TemplateID.UUID, Name: p.Name, FieldType: p.FieldType, IsRequired: p.IsRequired, Options: repositories.JSONToOptions(p.Options)}, nil
}

func (s *TemplateService) UpdateProjectParam(ctx context.Context, templateID, paramID uuid.UUID, name *string, isRequired *bool, options []string) (domain.TemplateProjectParam, error) {
	if name != nil && strings.TrimSpace(*name) == "" {
		return domain.TemplateProjectParam{}, domain.ErrInvalidInput
	}
	existing, err := s.repo.GetProjectParamByID(ctx, paramID)
	if err != nil || !existing.TemplateID.Valid || existing.TemplateID.UUID != templateID {
		return domain.TemplateProjectParam{}, domain.ErrNotFound
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
		finalOptions = repositories.OptionsToJSON(options)
	}
	p, err := s.repo.UpdateProjectParam(ctx, db.UpdateTemplateProjectParamParams{ID: paramID, Name: finalName, IsRequired: finalRequired, Options: finalOptions})
	if err != nil {
		return domain.TemplateProjectParam{}, err
	}
	return domain.TemplateProjectParam{ID: p.ID, TemplateID: p.TemplateID.UUID, Name: p.Name, FieldType: p.FieldType, IsRequired: p.IsRequired, Options: repositories.JSONToOptions(p.Options)}, nil
}

func (s *TemplateService) DeleteProjectParam(ctx context.Context, templateID, paramID uuid.UUID) error {
	existing, err := s.repo.GetProjectParamByID(ctx, paramID)
	if err != nil || !existing.TemplateID.Valid || existing.TemplateID.UUID != templateID {
		return domain.ErrNotFound
	}
	return s.repo.DeleteProjectParam(ctx, paramID)
}

// --- Roles ---

func (s *TemplateService) CreateRole(ctx context.Context, templateID uuid.UUID, name, description string, permissions []domain.TemplateRolePermission) (domain.TemplateRole, error) {
	if strings.TrimSpace(name) == "" {
		return domain.TemplateRole{}, domain.ErrInvalidInput
	}
	_, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return domain.TemplateRole{}, err
	}
	role, err := s.repo.CreateRole(ctx, db.CreateTemplateRoleParams{
		TemplateID: uuid.NullUUID{UUID: templateID, Valid: true}, Name: name, Description: description,
	})
	if err != nil {
		return domain.TemplateRole{}, err
	}
	for _, p := range permissions {
		_ = s.repo.UpsertRolePermission(ctx, role.ID, p.Area, p.Access)
	}
	return s.loadRole(ctx, role)
}

func (s *TemplateService) UpdateRole(ctx context.Context, templateID, roleID uuid.UUID, name, description *string, permissions []domain.TemplateRolePermission) (domain.TemplateRole, error) {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil || !role.TemplateID.Valid || role.TemplateID.UUID != templateID {
		return domain.TemplateRole{}, domain.ErrNotFound
	}
	if name != nil && strings.TrimSpace(*name) == "" {
		return domain.TemplateRole{}, domain.ErrInvalidInput
	}
	// Роль is_admin в шаблоне: можно менять только name и description, права менять нельзя
	if role.IsAdmin && permissions != nil {
		return domain.TemplateRole{}, domain.ErrTemplateAdminRole
	}
	finalName := role.Name
	if name != nil {
		finalName = *name
	}
	finalDesc := ""
	if description != nil {
		finalDesc = *description
	}
	updated, err := s.repo.UpdateRole(ctx, db.UpdateTemplateRoleParams{ID: roleID, Name: finalName, Description: finalDesc})
	if err != nil {
		return domain.TemplateRole{}, err
	}
	if permissions != nil {
		for _, p := range permissions {
			_ = s.repo.UpsertRolePermission(ctx, roleID, p.Area, p.Access)
		}
	}
	return s.loadRole(ctx, updated)
}

func (s *TemplateService) DeleteRole(ctx context.Context, templateID, roleID uuid.UUID) error {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil || !role.TemplateID.Valid || role.TemplateID.UUID != templateID {
		return domain.ErrNotFound
	}
	if role.IsAdmin {
		return domain.ErrTemplateAdminRole
	}
	return s.repo.DeleteRole(ctx, roleID)
}

func (s *TemplateService) loadRole(ctx context.Context, r db.ListTemplateRolesRow) (domain.TemplateRole, error) {
	perms, err := s.repo.ListRolePermissions(ctx, r.ID)
	if err != nil {
		return domain.TemplateRole{}, err
	}
	domPerms := make([]domain.TemplateRolePermission, 0, len(perms))
	for _, p := range perms {
		domPerms = append(domPerms, domain.TemplateRolePermission{Area: p.PermissionCode, Access: p.Access.String})
	}
	return domain.TemplateRole{
		ID: r.ID, TemplateID: r.TemplateID.UUID, Name: r.Name,
		IsAdmin: r.IsAdmin, Permissions: domPerms,
	}, nil
}

// --- Private helpers ---

func (s *TemplateService) loadFullData(ctx context.Context, templateID uuid.UUID) (TemplateFullData, error) {
	boards, err := s.loadBoards(ctx, templateID)
	if err != nil {
		return TemplateFullData{}, err
	}
	params, err := s.loadProjectParams(ctx, templateID)
	if err != nil {
		return TemplateFullData{}, err
	}
	roles, err := s.loadRoles(ctx, templateID)
	if err != nil {
		return TemplateFullData{}, err
	}
	return TemplateFullData{Boards: boards, Params: params, Roles: roles}, nil
}

func (s *TemplateService) loadProjectParams(ctx context.Context, templateID uuid.UUID) ([]domain.TemplateProjectParam, error) {
	dbParams, err := s.repo.ListProjectParams(ctx, templateID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.TemplateProjectParam, 0, len(dbParams))
	for _, p := range dbParams {
		result = append(result, domain.TemplateProjectParam{
			ID: p.ID, TemplateID: p.TemplateID.UUID, Name: p.Name,
			FieldType: p.FieldType, IsSystem: false, IsRequired: p.IsRequired,
			Options: repositories.JSONToOptions(p.Options),
		})
	}
	return result, nil
}

func (s *TemplateService) loadRoles(ctx context.Context, templateID uuid.UUID) ([]domain.TemplateRole, error) {
	dbRoles, err := s.repo.ListRoles(ctx, templateID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.TemplateRole, 0, len(dbRoles))
	for _, r := range dbRoles {
		perms, err := s.repo.ListRolePermissions(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		domPerms := make([]domain.TemplateRolePermission, 0, len(perms))
		for _, p := range perms {
			domPerms = append(domPerms, domain.TemplateRolePermission{Area: p.PermissionCode, Access: p.Access.String})
		}
		result = append(result, domain.TemplateRole{
			ID: r.ID, TemplateID: r.TemplateID.UUID, Name: r.Name,
			IsAdmin: r.IsAdmin, Permissions: domPerms,
		})
	}
	return result, nil
}

func (s *TemplateService) loadBoards(ctx context.Context, templateID uuid.UUID) ([]domain.TemplateBoard, error) {
	dbBoards, err := s.repo.ListBoardsByTemplateID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	boards := make([]domain.TemplateBoard, 0, len(dbBoards))
	for _, b := range dbBoards {
		full, err := s.loadFullBoard(ctx, b)
		if err != nil {
			return nil, err
		}
		boards = append(boards, full)
	}
	return boards, nil
}

func (s *TemplateService) loadFullBoard(ctx context.Context, b db.ListTemplateBoardsByTemplateIDRow) (domain.TemplateBoard, error) {
	columns, err := s.repo.ListColumns(ctx, b.ID)
	if err != nil {
		return domain.TemplateBoard{}, err
	}
	swimlanes, err := s.repo.ListSwimlanes(ctx, b.ID)
	if err != nil {
		return domain.TemplateBoard{}, err
	}
	fields, err := s.repo.ListCustomFields(ctx, b.ID)
	if err != nil {
		return domain.TemplateBoard{}, err
	}

	domCols := make([]domain.TemplateBoardColumn, 0, len(columns))
	for _, c := range columns {
		domCols = append(domCols, mapDBColumnToDomain(c))
	}
	domSws := make([]domain.TemplateBoardSwimlane, 0, len(swimlanes))
	for _, sw := range swimlanes {
		domSws = append(domSws, mapDBSwimlaneToDomain(sw))
	}
	domFields := make([]domain.TemplateBoardCustomField, 0, len(fields))
	for _, f := range fields {
		domFields = append(domFields, mapDBFieldToDomain(f))
	}

	return s.buildBoardDomain(b, domCols, domSws, domFields), nil
}

func (s *TemplateService) buildBoardDomain(b db.ListTemplateBoardsByTemplateIDRow, columns []domain.TemplateBoardColumn, swimlanes []domain.TemplateBoardSwimlane, fields []domain.TemplateBoardCustomField) domain.TemplateBoard {
	if columns == nil {
		columns = []domain.TemplateBoardColumn{}
	}
	if swimlanes == nil {
		swimlanes = []domain.TemplateBoardSwimlane{}
	}
	if fields == nil {
		fields = []domain.TemplateBoardCustomField{}
	}
	return domain.TemplateBoard{
		ID:              b.ID,
		TemplateID:      b.TemplateID.UUID,
		Name:            b.Name,
		Description:     b.Description.String,
		IsDefault:       b.IsDefault,
		Order:           int32(b.SortOrder),
		PriorityType:    b.PriorityType,
		EstimationUnit:  b.EstimationUnit,
		SwimlaneGroupBy: b.SwimlaneGroupBy,
		PriorityOptions: repositories.JSONToOptions(b.PriorityOptions),
		Columns:         columns,
		Swimlanes:       swimlanes,
		CustomFields:    fields,
	}
}

func (s *TemplateService) createDefaultBoard(ctx context.Context, templateID uuid.UUID, projectType string) (domain.TemplateBoard, error) {
	priorityType := "priority"
	estimationUnit := "story_points"
	description := "Доска для основного хода разработки"
	swimlaneGroupBy := ""
	if projectType == "kanban" {
		priorityType = "service_class"
		estimationUnit = "time"
		description = "Kanban-доска с поддержкой WIP лимитов"
		swimlaneGroupBy = domain.SystemBoardFieldIDs["priority"].String()
	}

	board, err := s.repo.CreateBoard(ctx, db.CreateTemplateBoardParams{
		TemplateID:      uuid.NullUUID{UUID: templateID, Valid: true},
		Name:            "Основная доска",
		Description:     sql.NullString{String: description, Valid: description != ""},
		IsDefault:       true,
		SortOrder:       1,
		PriorityType:    priorityType,
		EstimationUnit:  estimationUnit,
		SwimlaneGroupBy: swimlaneGroupBy,
	})
	if err != nil {
		return domain.TemplateBoard{}, err
	}

	columns, err := s.createDefaultColumns(ctx, board.ID, projectType)
	if err != nil {
		return domain.TemplateBoard{}, err
	}

	// Create swimlanes for Kanban
	var swimlanes []domain.TemplateBoardSwimlane
	if swimlaneGroupBy != "" {
		swimlanes, err = s.createSwimlanesForGroup(ctx, board.ID, swimlaneGroupBy, priorityType)
		if err != nil {
			return domain.TemplateBoard{}, err
		}
	}

	return s.buildBoardDomain(board, columns, swimlanes, nil), nil
}

func (s *TemplateService) createDefaultColumns(ctx context.Context, boardID uuid.UUID, projectType string) ([]domain.TemplateBoardColumn, error) {
	defaults := repositories.DefaultColumns[projectType]

	columns := make([]domain.TemplateBoardColumn, 0, len(defaults))
	for i, d := range defaults {
		col, err := s.repo.CreateColumn(ctx, db.CreateTemplateBoardColumnParams{
			BoardID:    boardID,
			Name:       d.Name,
			SystemType: sql.NullString{String: d.SystemType, Valid: d.SystemType != ""},
			WipLimit:   sql.NullInt16{},
			SortOrder:  int16(i + 1),
			IsLocked:   d.IsLocked,
			Note:       "",
		})
		if err != nil {
			return nil, err
		}
		columns = append(columns, mapDBColumnToDomain(col))
	}
	return columns, nil
}

// createSystemFields removed: system fields are now generated at runtime from Go constants.

func (s *TemplateService) createSwimlanesForGroup(ctx context.Context, boardID uuid.UUID, groupBy, priorityType string) ([]domain.TemplateBoardSwimlane, error) {
	// Resolve the lookup key: if swimlane_group_by is the priority system field UUID,
	// use the board's priority_type to look up default swimlane names.
	lookupKey := groupBy
	if groupBy == domain.SystemBoardFieldIDs["priority"].String() {
		lookupKey = priorityType
	}

	var names []string
	for _, pt := range repositories.PriorityTypes {
		if pt.Key == lookupKey {
			names = pt.DefaultValues
			break
		}
	}
	if len(names) == 0 {
		return nil, nil
	}

	result := make([]domain.TemplateBoardSwimlane, 0, len(names))
	for i, name := range names {
		sw, err := s.repo.CreateSwimlane(ctx, db.CreateTemplateBoardSwimlaneParams{
			BoardID:  boardID,
			Name:     name,
			WipLimit: sql.NullInt16{},
			SortOrder: int16(i + 1),
			Note:      "",
		})
		if err != nil {
			return nil, err
		}
		result = append(result, mapDBSwimlaneToDomain(sw))
	}
	return result, nil
}

// --- Mapping helpers ---

func mapDBColumnToDomain(c db.Column) domain.TemplateBoardColumn {
	var wl *int32
	if c.WipLimit.Valid {
		v := int32(c.WipLimit.Int16)
		wl = &v
	}
	var note *string
	if c.Note != "" {
		note = &c.Note
	}
	return domain.TemplateBoardColumn{
		ID: c.ID, BoardID: c.BoardID, Name: c.Name, SystemType: c.SystemType.String,
		WipLimit: wl, Order: int32(c.SortOrder), IsLocked: c.IsLocked, Note: note,
	}
}

func mapDBSwimlaneToDomain(sw db.Swimlane) domain.TemplateBoardSwimlane {
	var wl *int32
	if sw.WipLimit.Valid {
		v := int32(sw.WipLimit.Int16)
		wl = &v
	}
	var note *string
	if sw.Note != "" {
		note = &sw.Note
	}
	return domain.TemplateBoardSwimlane{
		ID: sw.ID, BoardID: sw.BoardID, Name: sw.Name,
		WipLimit: wl, Order: int32(sw.SortOrder), Note: note,
	}
}

func mapDBFieldToDomain(f db.BoardField) domain.TemplateBoardCustomField {
	return domain.TemplateBoardCustomField{
		ID: f.ID, BoardID: f.BoardID, Name: f.Name,
		FieldType: f.FieldType, IsSystem: false, IsRequired: f.IsRequired,
		Options: repositories.JSONToOptions(f.Options),
	}
}

// validateColumnOrder ensures all initial < all in_progress < all completed
func validateColumnOrder(columns []db.Column) error {
	sorted := make([]db.Column, len(columns))
	copy(sorted, columns)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].SortOrder < sorted[j].SortOrder })

	typeOrder := map[string]int{"initial": 0, "in_progress": 1, "completed": 2}
	maxSeen := -1
	for _, col := range sorted {
		to, ok := typeOrder[col.SystemType.String]
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

// recompactTemplateColumnOrders renumbers all columns for a template board sequentially: 1, 2, 3, ...
func (s *TemplateService) recompactTemplateColumnOrders(ctx context.Context, boardID uuid.UUID) error {
	columns, err := s.repo.ListColumns(ctx, boardID)
	if err != nil {
		return err
	}
	sort.Slice(columns, func(i, j int) bool { return columns[i].SortOrder < columns[j].SortOrder })
	for i, col := range columns {
		newOrder := int16(i + 1)
		if col.SortOrder != newOrder {
			if err := s.repo.UpdateColumnOrder(ctx, col.ID, newOrder); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateMinColumnTypesDB checks that removing a column won't leave the board without
// at least one initial, one in_progress, and one completed column. Works with db.Column.
func validateMinColumnTypesDB(columns []db.Column, excludeID uuid.UUID) error {
	counts := map[string]int{"initial": 0, "in_progress": 0, "completed": 0}
	for _, col := range columns {
		if col.ID == excludeID || !col.SystemType.Valid {
			continue
		}
		if _, ok := counts[col.SystemType.String]; ok {
			counts[col.SystemType.String]++
		}
	}
	for st, cnt := range counts {
		if cnt < 1 {
			return fmt.Errorf("INVALID_COLUMN_ORDER: на доске должна быть хотя бы одна колонка типа %s: %w", st, domain.ErrInvalidColumnOrder)
		}
	}
	return nil
}

// GetReferences loads all reference/lookup data
func (s *TemplateService) GetReferences(ctx context.Context) (*domain.References, error) {
	columnTypes, err := s.refRepo.ListColumnSystemTypes(ctx)
	if err != nil {
		return nil, err
	}
	fieldTypes, err := s.refRepo.ListFieldTypes(ctx)
	if err != nil {
		return nil, err
	}
	estimationUnits, err := s.refRepo.ListEstimationUnits(ctx)
	if err != nil {
		return nil, err
	}
	priorityTypes, err := s.refRepo.ListPriorityTypes(ctx)
	if err != nil {
		return nil, err
	}
	permAreas, err := s.refRepo.ListPermissionAreas(ctx)
	if err != nil {
		return nil, err
	}
	levels, err := s.refRepo.ListAccessLevels(ctx)
	if err != nil {
		return nil, err
	}

	refs := &domain.References{
		ColumnSystemTypes:   columnTypes,
		FieldTypes:          fieldTypes,
		EstimationUnits:     estimationUnits,
		PriorityTypeOptions: priorityTypes,
		ProjectStatuses:     repositories.ProjectStatuses,
		PermissionAreas:     permAreas,
	}

	// SwimlaneGroupOptions не заполняем — фронтенд формирует список
	// доступных параметров для группировки дорожек динамически
	// из всех параметров доски (системных + кастомных).

	for _, l := range levels {
		refs.AccessLevels = append(refs.AccessLevels, domain.RefKeyName{Key: l.Key, Name: l.Name})
	}

	return refs, nil
}
