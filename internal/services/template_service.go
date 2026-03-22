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

	if isDefault {
		_ = s.repo.UnsetDefaultBoard(ctx, templateID)
	}

	count, _ := s.repo.CountBoardsByTemplateID(ctx, templateID)
	sgb := ""
	if swimlaneGroupBy != nil {
		sgb = *swimlaneGroupBy
	}

	board, err := s.repo.CreateBoard(ctx, db.CreateTemplateBoardParams{
		TemplateID:      templateID,
		Name:            name,
		Description:     description,
		IsDefault:       isDefault,
		Order:           count + 1,
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

	// Create default priority values
	pvs, err := s.createDefaultPriorityValues(ctx, board.ID, priorityType)
	if err != nil {
		return domain.TemplateBoard{}, err
	}

	// Create swimlanes if grouped
	var swimlanes []domain.TemplateBoardSwimlane
	if sgb != "" {
		swimlanes, err = s.createSwimlanesForGroup(ctx, board.ID, sgb, pvs)
		if err != nil {
			return domain.TemplateBoard{}, err
		}
	}

	return s.buildBoardDomain(board, columns, swimlanes, pvs, nil), nil
}

func (s *TemplateService) UpdateBoard(ctx context.Context, templateID, boardID uuid.UUID, name, description *string, isDefault *bool, order *int32, priorityType, estimationUnit *string, swimlaneGroupBy *string, clearSwimlaneGroup bool) (domain.TemplateBoard, error) {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil {
		return domain.TemplateBoard{}, domain.ErrNotFound
	}
	if board.TemplateID != templateID {
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
		finalDesc = *description
	}
	finalIsDefault := board.IsDefault
	if isDefault != nil {
		finalIsDefault = *isDefault
	}
	finalOrder := board.Order
	if order != nil {
		finalOrder = *order
	}
	finalPriorityType := board.PriorityType
	priorityChanged := false
	if priorityType != nil && *priorityType != board.PriorityType {
		finalPriorityType = *priorityType
		priorityChanged = true
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
		Order:           finalOrder,
		PriorityType:    finalPriorityType,
		EstimationUnit:  finalEstimationUnit,
		SwimlaneGroupBy: finalSwimlaneGroupBy,
	})
	if err != nil {
		return domain.TemplateBoard{}, err
	}

	// Replace priority values on type change
	if priorityChanged {
		_ = s.repo.DeletePriorityValuesByBoardID(ctx, boardID)
		_, _ = s.createDefaultPriorityValues(ctx, boardID, finalPriorityType)
	}

	// Recreate swimlanes on group change
	if swimlaneGroupChanged {
		_ = s.repo.DeleteSwimlanesByBoardID(ctx, boardID)
		if finalSwimlaneGroupBy != "" {
			pvs, _ := s.repo.ListPriorityValues(ctx, boardID)
			var domPVs []domain.TemplateBoardPriorityValue
			for _, pv := range pvs {
				domPVs = append(domPVs, domain.TemplateBoardPriorityValue{
					ID: pv.ID, BoardID: pv.BoardID, Value: pv.Value, Order: pv.Order,
				})
			}
			_, _ = s.createSwimlanesForGroup(ctx, boardID, finalSwimlaneGroupBy, domPVs)
		}
	}

	return s.loadFullBoard(ctx, updated)
}

func (s *TemplateService) DeleteBoard(ctx context.Context, templateID, boardID uuid.UUID) error {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil {
		return domain.ErrNotFound
	}
	if board.TemplateID != templateID {
		return domain.ErrNotFound
	}

	count, _ := s.repo.CountBoardsByTemplateID(ctx, templateID)
	if count <= 1 {
		return fmt.Errorf("LAST_BOARD: %w", domain.ErrLastBoard)
	}

	return s.repo.DeleteBoard(ctx, boardID)
}

func (s *TemplateService) ReorderBoards(ctx context.Context, templateID uuid.UUID, orders map[uuid.UUID]int32) error {
	for id, order := range orders {
		if err := s.repo.UpdateBoardOrder(ctx, id, order); err != nil {
			return err
		}
	}
	return nil
}

// --- Columns ---

func (s *TemplateService) CreateColumn(ctx context.Context, templateID, boardID uuid.UUID, name, systemType string, wipLimit *int32, order int32, note string) (domain.TemplateBoardColumn, error) {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || board.TemplateID != templateID {
		return domain.TemplateBoardColumn{}, domain.ErrNotFound
	}

	wl := sql.NullInt32{}
	if wipLimit != nil {
		wl = sql.NullInt32{Int32: *wipLimit, Valid: true}
	}

	col, err := s.repo.CreateColumn(ctx, db.CreateTemplateBoardColumnParams{
		BoardID:    boardID,
		Name:       name,
		SystemType: systemType,
		WipLimit:   wl,
		Order:      order,
		IsLocked:   false,
		Note:       note,
	})
	if err != nil {
		return domain.TemplateBoardColumn{}, err
	}

	// Validate column order
	columns, _ := s.repo.ListColumns(ctx, boardID)
	if err := validateColumnOrder(columns); err != nil {
		_ = s.repo.DeleteColumn(ctx, col.ID)
		return domain.TemplateBoardColumn{}, err
	}

	return mapDBColumnToDomain(col), nil
}

func (s *TemplateService) UpdateColumn(ctx context.Context, templateID, boardID, columnID uuid.UUID, name, systemType *string, wipLimit *int32, note *string) (domain.TemplateBoardColumn, error) {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || board.TemplateID != templateID {
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
		finalSystemType = *systemType
	}
	finalWipLimit := col.WipLimit
	if wipLimit != nil {
		finalWipLimit = sql.NullInt32{Int32: *wipLimit, Valid: true}
	}
	finalNote := col.Note
	if note != nil {
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
	if err != nil || board.TemplateID != templateID {
		return domain.ErrNotFound
	}

	col, err := s.repo.GetColumnByID(ctx, columnID)
	if err != nil || col.BoardID != boardID {
		return domain.ErrNotFound
	}

	if col.IsLocked {
		return fmt.Errorf("COLUMN_LOCKED: %w", domain.ErrColumnLocked)
	}

	if err := s.repo.DeleteColumn(ctx, columnID); err != nil {
		return err
	}

	// Validate remaining order
	columns, _ := s.repo.ListColumns(ctx, boardID)
	if err := validateColumnOrder(columns); err != nil {
		// Re-create the column
		_, _ = s.repo.CreateColumn(ctx, db.CreateTemplateBoardColumnParams{
			BoardID: boardID, Name: col.Name, SystemType: col.SystemType, WipLimit: col.WipLimit, Order: col.Order, IsLocked: col.IsLocked, Note: col.Note,
		})
		return err
	}

	return nil
}

func (s *TemplateService) ReorderColumns(ctx context.Context, templateID, boardID uuid.UUID, orders map[uuid.UUID]int32) error {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || board.TemplateID != templateID {
		return domain.ErrNotFound
	}

	// Check locked columns
	columns, _ := s.repo.ListColumns(ctx, boardID)
	lockedColumns := make(map[uuid.UUID]int32)
	for _, col := range columns {
		if col.IsLocked {
			lockedColumns[col.ID] = col.Order
		}
	}
	for id, newOrder := range orders {
		if origOrder, ok := lockedColumns[id]; ok && origOrder != newOrder {
			return fmt.Errorf("COLUMN_LOCKED: %w", domain.ErrColumnLocked)
		}
	}

	for id, order := range orders {
		if err := s.repo.UpdateColumnOrder(ctx, id, order); err != nil {
			return err
		}
	}

	// Validate after reorder
	updatedColumns, _ := s.repo.ListColumns(ctx, boardID)
	if err := validateColumnOrder(updatedColumns); err != nil {
		// Rollback
		for _, col := range columns {
			_ = s.repo.UpdateColumnOrder(ctx, col.ID, col.Order)
		}
		return err
	}

	return nil
}

// --- Swimlanes ---

func (s *TemplateService) UpdateSwimlane(ctx context.Context, templateID, boardID, swimlaneID uuid.UUID, wipLimit *int32, note *string) (domain.TemplateBoardSwimlane, error) {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || board.TemplateID != templateID {
		return domain.TemplateBoardSwimlane{}, domain.ErrNotFound
	}

	existing, err := s.repo.GetSwimlaneByID(ctx, swimlaneID)
	if err != nil {
		return domain.TemplateBoardSwimlane{}, domain.ErrNotFound
	}

	wl := sql.NullInt32{}
	if wipLimit != nil {
		wl = sql.NullInt32{Int32: *wipLimit, Valid: true}
	} else if existing.WipLimit.Valid {
		wl = existing.WipLimit
	}
	finalNote := existing.Note
	if note != nil {
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
	if err != nil || board.TemplateID != templateID {
		return domain.ErrNotFound
	}
	return s.repo.DeleteSwimlane(ctx, swimlaneID)
}

func (s *TemplateService) ReorderSwimlanes(ctx context.Context, templateID, boardID uuid.UUID, orders map[uuid.UUID]int32) error {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || board.TemplateID != templateID {
		return domain.ErrNotFound
	}
	for id, order := range orders {
		if err := s.repo.UpdateSwimlaneOrder(ctx, id, order); err != nil {
			return err
		}
	}
	return nil
}

// --- Priority Values ---

func (s *TemplateService) ReplacePriorityValues(ctx context.Context, templateID, boardID uuid.UUID, values []struct {
	Value string
	Order int32
}) ([]domain.TemplateBoardPriorityValue, error) {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || board.TemplateID != templateID {
		return nil, domain.ErrNotFound
	}

	_ = s.repo.DeletePriorityValuesByBoardID(ctx, boardID)

	result := make([]domain.TemplateBoardPriorityValue, 0, len(values))
	for _, v := range values {
		pv, err := s.repo.CreatePriorityValue(ctx, db.CreateTemplateBoardPriorityValueParams{
			BoardID: boardID,
			Value:   v.Value,
			Order:   v.Order,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, domain.TemplateBoardPriorityValue{
			ID: pv.ID, BoardID: pv.BoardID, Value: pv.Value, Order: pv.Order,
		})
	}

	return result, nil
}

// --- Custom Fields ---

func (s *TemplateService) CreateCustomField(ctx context.Context, templateID, boardID uuid.UUID, name, fieldType string, isRequired bool, order int32, options []string) (domain.TemplateBoardCustomField, error) {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || board.TemplateID != templateID {
		return domain.TemplateBoardCustomField{}, domain.ErrNotFound
	}

	field, err := s.repo.CreateField(ctx, db.CreateTemplateBoardFieldParams{
		BoardID:    boardID,
		Code:       "",
		Name:       name,
		FieldType:  fieldType,
		IsSystem:   false,
		IsRequired: isRequired,
		IsActive:   true,
		Order:      order,
		Options:    repositories.OptionsToJSON(options),
	})
	if err != nil {
		return domain.TemplateBoardCustomField{}, err
	}

	return mapDBFieldToDomain(field), nil
}

func (s *TemplateService) UpdateCustomField(ctx context.Context, templateID, boardID, fieldID uuid.UUID, name *string, isRequired *bool, options []string) (domain.TemplateBoardCustomField, error) {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || board.TemplateID != templateID {
		return domain.TemplateBoardCustomField{}, domain.ErrNotFound
	}

	existing, err := s.repo.GetFieldByID(ctx, fieldID)
	if err != nil || existing.BoardID != boardID || existing.IsSystem {
		return domain.TemplateBoardCustomField{}, domain.ErrNotFound
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
	if err != nil || board.TemplateID != templateID {
		return domain.ErrNotFound
	}

	existing, err := s.repo.GetFieldByID(ctx, fieldID)
	if err != nil || existing.BoardID != boardID || existing.IsSystem {
		return domain.ErrNotFound
	}

	return s.repo.DeleteField(ctx, fieldID)
}

func (s *TemplateService) ReorderCustomFields(ctx context.Context, templateID, boardID uuid.UUID, orders map[uuid.UUID]int32) error {
	board, err := s.repo.GetBoardByID(ctx, boardID)
	if err != nil || board.TemplateID != templateID {
		return domain.ErrNotFound
	}
	for id, order := range orders {
		if err := s.repo.UpdateFieldOrder(ctx, id, order); err != nil {
			return err
		}
	}
	return nil
}

// --- Project Params ---

func (s *TemplateService) CreateProjectParam(ctx context.Context, templateID uuid.UUID, name, fieldType string, isRequired bool, order int32, options []string) (domain.TemplateProjectParam, error) {
	_, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return domain.TemplateProjectParam{}, err
	}
	p, err := s.repo.CreateProjectParam(ctx, db.CreateTemplateProjectParamParams{
		TemplateID: templateID, Name: name, FieldType: fieldType, IsRequired: isRequired, Order: order, Options: repositories.OptionsToJSON(options),
	})
	if err != nil {
		return domain.TemplateProjectParam{}, err
	}
	return domain.TemplateProjectParam{ID: p.ID, TemplateID: p.TemplateID, Name: p.Name, FieldType: p.FieldType, IsRequired: p.IsRequired, Order: p.Order, Options: repositories.JSONToOptions(p.Options)}, nil
}

func (s *TemplateService) UpdateProjectParam(ctx context.Context, templateID, paramID uuid.UUID, name *string, isRequired *bool, options []string) (domain.TemplateProjectParam, error) {
	existing, err := s.repo.GetProjectParamByID(ctx, paramID)
	if err != nil || existing.TemplateID != templateID {
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
	return domain.TemplateProjectParam{ID: p.ID, TemplateID: p.TemplateID, Name: p.Name, FieldType: p.FieldType, IsRequired: p.IsRequired, Order: p.Order, Options: repositories.JSONToOptions(p.Options)}, nil
}

func (s *TemplateService) DeleteProjectParam(ctx context.Context, templateID, paramID uuid.UUID) error {
	existing, err := s.repo.GetProjectParamByID(ctx, paramID)
	if err != nil || existing.TemplateID != templateID {
		return domain.ErrNotFound
	}
	return s.repo.DeleteProjectParam(ctx, paramID)
}

func (s *TemplateService) ReorderProjectParams(ctx context.Context, templateID uuid.UUID, orders map[uuid.UUID]int32) error {
	for id, order := range orders {
		if err := s.repo.UpdateProjectParamOrder(ctx, id, order); err != nil {
			return err
		}
	}
	return nil
}

// --- Roles ---

func (s *TemplateService) CreateRole(ctx context.Context, templateID uuid.UUID, name, description string, permissions []domain.TemplateRolePermission) (domain.TemplateRole, error) {
	_, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return domain.TemplateRole{}, err
	}
	count, _ := s.repo.CountRoles(ctx, templateID)
	role, err := s.repo.CreateRole(ctx, db.CreateTemplateRoleParams{
		TemplateID: templateID, Name: name, Description: description, IsDefault: false, Order: count + 1,
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
	if err != nil || role.TemplateID != templateID {
		return domain.TemplateRole{}, domain.ErrNotFound
	}
	finalName := role.Name
	if name != nil {
		finalName = *name
	}
	finalDesc := role.Description
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
	if err != nil || role.TemplateID != templateID {
		return domain.ErrNotFound
	}
	return s.repo.DeleteRole(ctx, roleID)
}

func (s *TemplateService) ReorderRoles(ctx context.Context, templateID uuid.UUID, orders map[uuid.UUID]int32) error {
	for id, order := range orders {
		if err := s.repo.UpdateRoleOrder(ctx, id, order); err != nil {
			return err
		}
	}
	return nil
}

func (s *TemplateService) loadRole(ctx context.Context, r db.TemplateRole) (domain.TemplateRole, error) {
	perms, err := s.repo.ListRolePermissions(ctx, r.ID)
	if err != nil {
		return domain.TemplateRole{}, err
	}
	domPerms := make([]domain.TemplateRolePermission, 0, len(perms))
	for _, p := range perms {
		domPerms = append(domPerms, domain.TemplateRolePermission{Area: p.Area, Access: p.Access})
	}
	return domain.TemplateRole{
		ID: r.ID, TemplateID: r.TemplateID, Name: r.Name, Description: r.Description,
		IsDefault: r.IsDefault, Order: r.Order, Permissions: domPerms,
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
			ID: p.ID, TemplateID: p.TemplateID, Name: p.Name, FieldType: p.FieldType,
			IsRequired: p.IsRequired, Order: p.Order, Options: repositories.JSONToOptions(p.Options),
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
			domPerms = append(domPerms, domain.TemplateRolePermission{Area: p.Area, Access: p.Access})
		}
		result = append(result, domain.TemplateRole{
			ID: r.ID, TemplateID: r.TemplateID, Name: r.Name, Description: r.Description,
			IsDefault: r.IsDefault, Order: r.Order, Permissions: domPerms,
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

func (s *TemplateService) loadFullBoard(ctx context.Context, b db.TemplateBoard) (domain.TemplateBoard, error) {
	columns, err := s.repo.ListColumns(ctx, b.ID)
	if err != nil {
		return domain.TemplateBoard{}, err
	}
	swimlanes, err := s.repo.ListSwimlanes(ctx, b.ID)
	if err != nil {
		return domain.TemplateBoard{}, err
	}
	pvs, err := s.repo.ListPriorityValues(ctx, b.ID)
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
	domPVs := make([]domain.TemplateBoardPriorityValue, 0, len(pvs))
	for _, pv := range pvs {
		domPVs = append(domPVs, domain.TemplateBoardPriorityValue{
			ID: pv.ID, BoardID: pv.BoardID, Value: pv.Value, Order: pv.Order,
		})
	}
	domFields := make([]domain.TemplateBoardCustomField, 0, len(fields))
	for _, f := range fields {
		domFields = append(domFields, mapDBFieldToDomain(f))
	}

	return s.buildBoardDomain(b, domCols, domSws, domPVs, domFields), nil
}

func (s *TemplateService) buildBoardDomain(b db.TemplateBoard, columns []domain.TemplateBoardColumn, swimlanes []domain.TemplateBoardSwimlane, pvs []domain.TemplateBoardPriorityValue, fields []domain.TemplateBoardCustomField) domain.TemplateBoard {
	if columns == nil {
		columns = []domain.TemplateBoardColumn{}
	}
	if swimlanes == nil {
		swimlanes = []domain.TemplateBoardSwimlane{}
	}
	if pvs == nil {
		pvs = []domain.TemplateBoardPriorityValue{}
	}
	if fields == nil {
		fields = []domain.TemplateBoardCustomField{}
	}
	return domain.TemplateBoard{
		ID:              b.ID,
		TemplateID:      b.TemplateID,
		Name:            b.Name,
		Description:     b.Description,
		IsDefault:       b.IsDefault,
		Order:           b.Order,
		PriorityType:    b.PriorityType,
		EstimationUnit:  b.EstimationUnit,
		SwimlaneGroupBy: b.SwimlaneGroupBy,
		Columns:         columns,
		Swimlanes:       swimlanes,
		PriorityValues:  pvs,
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
		swimlaneGroupBy = "service_class"
	}

	board, err := s.repo.CreateBoard(ctx, db.CreateTemplateBoardParams{
		TemplateID:      templateID,
		Name:            "Основная доска",
		Description:     description,
		IsDefault:       true,
		Order:           1,
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

	pvs, err := s.createDefaultPriorityValues(ctx, board.ID, priorityType)
	if err != nil {
		return domain.TemplateBoard{}, err
	}

	// Create swimlanes for Kanban
	var swimlanes []domain.TemplateBoardSwimlane
	if swimlaneGroupBy != "" {
		swimlanes, err = s.createSwimlanesForGroup(ctx, board.ID, swimlaneGroupBy, pvs)
		if err != nil {
			return domain.TemplateBoard{}, err
		}
	}

	return s.buildBoardDomain(board, columns, swimlanes, pvs, nil), nil
}

func (s *TemplateService) createDefaultColumns(ctx context.Context, boardID uuid.UUID, projectType string) ([]domain.TemplateBoardColumn, error) {
	type colDef struct {
		name       string
		systemType string
		order      int32
		isLocked   bool
	}

	var defaults []colDef
	if projectType == "kanban" {
		defaults = []colDef{
			{"Надо сделать", "initial", 1, false},
			{"Готово к работе", "initial", 2, false},
			{"В работе", "in_progress", 3, false},
			{"На проверке", "in_progress", 4, false},
			{"Выполнено", "completed", 5, false},
		}
	} else {
		defaults = []colDef{
			{"Бэклог спринта", "initial", 1, true},
			{"В работе", "in_progress", 2, false},
			{"На проверке", "in_progress", 3, false},
			{"Выполнено", "completed", 4, false},
		}
	}

	columns := make([]domain.TemplateBoardColumn, 0, len(defaults))
	for _, d := range defaults {
		col, err := s.repo.CreateColumn(ctx, db.CreateTemplateBoardColumnParams{
			BoardID:    boardID,
			Name:       d.name,
			SystemType: d.systemType,
			WipLimit:   sql.NullInt32{},
			Order:      d.order,
			IsLocked:   d.isLocked,
			Note:       "",
		})
		if err != nil {
			return nil, err
		}
		columns = append(columns, mapDBColumnToDomain(col))
	}
	return columns, nil
}

func (s *TemplateService) createDefaultPriorityValues(ctx context.Context, boardID uuid.UUID, priorityType string) ([]domain.TemplateBoardPriorityValue, error) {
	var defaults []string
	if priorityType == "service_class" {
		defaults = []string{"Ускоренный", "С фиксированной датой", "Стандартный", "Нематериальный"}
	} else {
		defaults = []string{"Низкий", "Средний", "Высокий", "Критичный"}
	}

	result := make([]domain.TemplateBoardPriorityValue, 0, len(defaults))
	for i, val := range defaults {
		pv, err := s.repo.CreatePriorityValue(ctx, db.CreateTemplateBoardPriorityValueParams{
			BoardID: boardID,
			Value:   val,
			Order:   int32(i + 1),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, domain.TemplateBoardPriorityValue{
			ID: pv.ID, BoardID: pv.BoardID, Value: pv.Value, Order: pv.Order,
		})
	}
	return result, nil
}

func (s *TemplateService) createSwimlanesForGroup(ctx context.Context, boardID uuid.UUID, groupBy string, pvs []domain.TemplateBoardPriorityValue) ([]domain.TemplateBoardSwimlane, error) {
	var names []string
	switch groupBy {
	case "priority":
		for _, pv := range pvs {
			names = append(names, pv.Value)
		}
	case "service_class":
		for _, pv := range pvs {
			names = append(names, pv.Value)
		}
	default:
		return nil, nil
	}

	result := make([]domain.TemplateBoardSwimlane, 0, len(names))
	for i, name := range names {
		sw, err := s.repo.CreateSwimlane(ctx, db.CreateTemplateBoardSwimlaneParams{
			BoardID:  boardID,
			Name:     name,
			Value:    name,
			WipLimit: sql.NullInt32{},
			Order:    int32(i + 1),
			Note:     "",
		})
		if err != nil {
			return nil, err
		}
		result = append(result, mapDBSwimlaneToDomain(sw))
	}
	return result, nil
}

// --- Mapping helpers ---

func mapDBColumnToDomain(c db.TemplateBoardColumn) domain.TemplateBoardColumn {
	var wl *int32
	if c.WipLimit.Valid {
		wl = &c.WipLimit.Int32
	}
	var note *string
	if c.Note != "" {
		note = &c.Note
	}
	return domain.TemplateBoardColumn{
		ID: c.ID, BoardID: c.BoardID, Name: c.Name, SystemType: c.SystemType,
		WipLimit: wl, Order: c.Order, IsLocked: c.IsLocked, Note: note,
	}
}

func mapDBSwimlaneToDomain(sw db.TemplateBoardSwimlane) domain.TemplateBoardSwimlane {
	var wl *int32
	if sw.WipLimit.Valid {
		wl = &sw.WipLimit.Int32
	}
	var note *string
	if sw.Note != "" {
		note = &sw.Note
	}
	return domain.TemplateBoardSwimlane{
		ID: sw.ID, BoardID: sw.BoardID, Name: sw.Name, Value: sw.Value,
		WipLimit: wl, Order: sw.Order, Note: note,
	}
}

func mapDBFieldToDomain(f db.TemplateBoardField) domain.TemplateBoardCustomField {
	return domain.TemplateBoardCustomField{
		ID: f.ID, BoardID: f.BoardID, Name: f.Name, FieldType: f.FieldType,
		IsSystem: f.IsSystem, IsRequired: f.IsRequired, Order: f.Order,
		Options: repositories.JSONToOptions(f.Options),
	}
}

// validateColumnOrder ensures all initial < all in_progress < all completed
func validateColumnOrder(columns []db.TemplateBoardColumn) error {
	sorted := make([]db.TemplateBoardColumn, len(columns))
	copy(sorted, columns)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Order < sorted[j].Order })

	typeOrder := map[string]int{"initial": 0, "in_progress": 1, "completed": 2}
	maxSeen := -1
	for _, col := range sorted {
		to, ok := typeOrder[col.SystemType]
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

// GetReferences loads all reference data from the database
func (s *TemplateService) GetReferences(ctx context.Context) (*domain.References, error) {
	columnTypes, err := s.refRepo.ListColumnSystemTypes(ctx)
	if err != nil {
		return nil, err
	}
	statusTypes, err := s.refRepo.ListTaskStatusTypes(ctx)
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
	systemFields, err := s.refRepo.ListSystemTaskFields(ctx)
	if err != nil {
		return nil, err
	}

	refs := &domain.References{}

	for _, ct := range columnTypes {
		refs.ColumnSystemTypes = append(refs.ColumnSystemTypes, domain.RefColumnSystemType{
			Key: ct.Key, Name: ct.Name, Description: ct.Description, Order: int(ct.SortOrder),
		})
	}
	for _, st := range statusTypes {
		refs.TaskStatusTypes = append(refs.TaskStatusTypes, domain.RefTaskStatusType{
			Key: st.Key, Name: st.Name, Description: st.Description, IsColumnType: st.IsColumnType,
		})
	}
	for _, ft := range fieldTypes {
		refs.FieldTypes = append(refs.FieldTypes, domain.RefKeyName{Key: ft.Key, Name: ft.Name})
	}
	for _, eu := range estimationUnits {
		refs.EstimationUnits = append(refs.EstimationUnits, domain.RefAvailable{
			Key: eu.Key, Name: eu.Name, AvailableFor: eu.AvailableFor,
		})
	}
	// swimlaneGroupOptions формируются динамически.
	// Допустимые типы параметров для группировки по дорожкам:
	//   select, multiselect (уникальные комбинации), checkbox (2 дорожки),
	//   user, sprint, tags.
	// Системные опции — из системных полей подходящих типов.
	// Кастомные поля (select/multiselect/checkbox) добавляются фронтендом на уровне доски.
	refs.SwimlaneGroupOptions = []domain.RefAvailable{
		{Key: "priority", Name: "по приоритету", AvailableFor: []string{"scrum", "kanban"}},
		{Key: "service_class", Name: "по классу обслуживания", AvailableFor: []string{"kanban"}},
		{Key: "executor", Name: "по исполнителю", AvailableFor: []string{"scrum", "kanban"}},
		{Key: "owner", Name: "по автору", AvailableFor: []string{"scrum", "kanban"}},
		{Key: "sprint", Name: "по спринту", AvailableFor: []string{"scrum"}},
		{Key: "tags", Name: "по тегам", AvailableFor: []string{"scrum", "kanban"}},
	}
	for _, pt := range priorityTypes {
		refs.PriorityTypeOptions = append(refs.PriorityTypeOptions, domain.RefPriorityType{
			Key: pt.Key, Name: pt.Name, AvailableFor: pt.AvailableFor, DefaultValues: pt.DefaultValues,
		})
	}
	for _, sf := range systemFields {
		refs.SystemTaskFields = append(refs.SystemTaskFields, domain.RefSystemField{
			Key: sf.Key, Name: sf.Name, FieldType: sf.FieldType, AvailableFor: sf.AvailableFor, Description: sf.Description,
		})
	}

	// System project params
	sysParams, err := s.refRepo.ListSystemProjectParams(ctx)
	if err != nil {
		return nil, err
	}
	for _, sp := range sysParams {
		refs.SystemProjectParams = append(refs.SystemProjectParams, domain.RefSystemProjectParam{
			Key: sp.Key, Name: sp.Name, FieldType: sp.FieldType, IsRequired: sp.IsRequired,
			Options: repositories.JSONToOptions(sp.Options),
		})
	}

	// Permission areas — flat array with availableFor
	permAreas, err := s.refRepo.ListPermissionAreas(ctx)
	if err != nil {
		return nil, err
	}
	areaMap := make(map[string]*domain.RefPermissionArea)
	areaOrder := make([]string, 0)
	for _, pa := range permAreas {
		if existing, ok := areaMap[pa.Area]; ok {
			existing.AvailableFor = append(existing.AvailableFor, pa.ProjectType)
		} else {
			areaMap[pa.Area] = &domain.RefPermissionArea{
				Area: pa.Area, Name: pa.Name, Description: pa.Description,
				AvailableFor: []string{pa.ProjectType},
			}
			areaOrder = append(areaOrder, pa.Area)
		}
	}
	for _, area := range areaOrder {
		refs.PermissionAreas = append(refs.PermissionAreas, *areaMap[area])
	}

	// Access levels
	levels, err := s.refRepo.ListAccessLevels(ctx)
	if err != nil {
		return nil, err
	}
	for _, l := range levels {
		refs.AccessLevels = append(refs.AccessLevels, domain.RefKeyName{Key: l.Key, Name: l.Name})
	}

	return refs, nil
}
