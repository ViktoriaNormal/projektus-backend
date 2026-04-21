package repositories

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type TemplateRepository interface {
	List(ctx context.Context) ([]domain.ProjectTemplate, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectTemplate, error)
	GetByType(ctx context.Context, projectType string) (*domain.ProjectTemplate, error)
	Create(ctx context.Context, name string, description *string, projectType string) (*domain.ProjectTemplate, error)
	Update(ctx context.Context, id uuid.UUID, name string, description *string) (*domain.ProjectTemplate, error)
	Delete(ctx context.Context, id uuid.UUID) error
	IsInUse(ctx context.Context, id uuid.UUID) (bool, error)

	// Boards
	ListBoardsByTemplateID(ctx context.Context, templateID uuid.UUID) ([]db.ListTemplateBoardsByTemplateIDRow, error)
	GetBoardByID(ctx context.Context, id uuid.UUID) (db.GetTemplateBoardByIDRow, error)
	CreateBoard(ctx context.Context, params db.CreateTemplateBoardParams) (db.ListTemplateBoardsByTemplateIDRow, error)
	UpdateBoard(ctx context.Context, params db.UpdateTemplateBoardParams) (db.ListTemplateBoardsByTemplateIDRow, error)
	DeleteBoard(ctx context.Context, id uuid.UUID) error
	CountBoardsByTemplateID(ctx context.Context, templateID uuid.UUID) (int32, error)
	UnsetDefaultBoard(ctx context.Context, templateID uuid.UUID) error
	UpdateBoardOrder(ctx context.Context, id uuid.UUID, order int16) error

	// Columns
	ListColumns(ctx context.Context, boardID uuid.UUID) ([]db.Column, error)
	GetColumnByID(ctx context.Context, id uuid.UUID) (db.Column, error)
	CreateColumn(ctx context.Context, params db.CreateTemplateBoardColumnParams) (db.Column, error)
	UpdateColumn(ctx context.Context, params db.UpdateTemplateBoardColumnParams) (db.Column, error)
	DeleteColumn(ctx context.Context, id uuid.UUID) error
	DeleteColumnsByBoardID(ctx context.Context, boardID uuid.UUID) error
	UpdateColumnOrder(ctx context.Context, id uuid.UUID, order int16) error

	// Swimlanes
	ListSwimlanes(ctx context.Context, boardID uuid.UUID) ([]db.Swimlane, error)
	GetSwimlaneByID(ctx context.Context, id uuid.UUID) (db.Swimlane, error)
	CreateSwimlane(ctx context.Context, params db.CreateTemplateBoardSwimlaneParams) (db.Swimlane, error)
	UpdateSwimlane(ctx context.Context, id uuid.UUID, wipLimit sql.NullInt16, note string) (db.Swimlane, error)
	DeleteSwimlane(ctx context.Context, id uuid.UUID) error
	DeleteSwimlanesByBoardID(ctx context.Context, boardID uuid.UUID) error
	UpdateSwimlaneOrder(ctx context.Context, id uuid.UUID, order int16) error

	// Custom fields
	ListCustomFields(ctx context.Context, boardID uuid.UUID) ([]db.BoardField, error)
	GetFieldByID(ctx context.Context, id uuid.UUID) (db.BoardField, error)
	CreateField(ctx context.Context, params db.CreateTemplateBoardFieldParams) (db.BoardField, error)
	UpdateField(ctx context.Context, params db.UpdateTemplateBoardFieldParams) (db.BoardField, error)
	DeleteField(ctx context.Context, id uuid.UUID) error

	// Project params
	ListProjectParams(ctx context.Context, templateID uuid.UUID) ([]db.ListTemplateProjectParamsRow, error)
	GetProjectParamByID(ctx context.Context, id uuid.UUID) (db.GetTemplateProjectParamByIDRow, error)
	CreateProjectParam(ctx context.Context, params db.CreateTemplateProjectParamParams) (db.ListTemplateProjectParamsRow, error)
	UpdateProjectParam(ctx context.Context, params db.UpdateTemplateProjectParamParams) (db.ListTemplateProjectParamsRow, error)
	DeleteProjectParam(ctx context.Context, id uuid.UUID) error

	// Roles
	ListRoles(ctx context.Context, templateID uuid.UUID) ([]db.ListTemplateRolesRow, error)
	GetRoleByID(ctx context.Context, id uuid.UUID) (db.GetTemplateRoleByIDRow, error)
	CreateRole(ctx context.Context, params db.CreateTemplateRoleParams) (db.ListTemplateRolesRow, error)
	UpdateRole(ctx context.Context, params db.UpdateTemplateRoleParams) (db.ListTemplateRolesRow, error)
	UpdateRoleOrder(ctx context.Context, id uuid.UUID, order int32) error
	DeleteRole(ctx context.Context, id uuid.UUID) error
	ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]db.RolePermission, error)
	UpsertRolePermission(ctx context.Context, roleID uuid.UUID, area, access string) error
	DeleteRolePermissions(ctx context.Context, roleID uuid.UUID) error
}

type templateRepository struct {
	q *db.Queries
}

func NewTemplateRepository(q *db.Queries) TemplateRepository {
	return &templateRepository{q: q}
}

func (r *templateRepository) List(ctx context.Context) ([]domain.ProjectTemplate, error) {
	rows, err := r.q.ListProjectTemplates(ctx)
	if err != nil {
		return nil, errctx.Wrap(err, "List")
	}
	templates := make([]domain.ProjectTemplate, 0, len(rows))
	for _, row := range rows {
		templates = append(templates, mapListRowToDomain(row))
	}
	return templates, nil
}

func (r *templateRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectTemplate, error) {
	row, err := r.q.GetProjectTemplateByID(ctx, id)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetByID", "id", id)
	}
	t := mapDBTemplateToDomainFull(row, 0)
	return &t, nil
}

func (r *templateRepository) GetByType(ctx context.Context, projectType string) (*domain.ProjectTemplate, error) {
	row, err := r.q.GetProjectTemplateByType(ctx, projectType)
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "GetByType", "projectType", projectType)
	}
	t := mapDBTemplateToDomainFull(row, 0)
	return &t, nil
}

func (r *templateRepository) Create(ctx context.Context, name string, description *string, projectType string) (*domain.ProjectTemplate, error) {
	desc := sql.NullString{}
	if description != nil {
		desc = sql.NullString{String: *description, Valid: true}
	}
	row, err := r.q.CreateProjectTemplate(ctx, db.CreateProjectTemplateParams{
		Name:        name,
		Description: desc,
		ProjectType: projectType,
	})
	if err != nil {
		return nil, errctx.Wrap(err, "Create")
	}
	t := mapDBTemplateToDomainFull(row, 0)
	return &t, nil
}

func (r *templateRepository) Update(ctx context.Context, id uuid.UUID, name string, description *string) (*domain.ProjectTemplate, error) {
	desc := sql.NullString{}
	if description != nil {
		desc = sql.NullString{String: *description, Valid: true}
	}
	row, err := r.q.UpdateProjectTemplate(ctx, db.UpdateProjectTemplateParams{
		ID:          id,
		Name:        name,
		Description: desc,
	})
	if err != nil {
		return nil, errctx.Wrap(mapSQLErr(err, domain.ErrNotFound), "Update", "id", id)
	}
	t := mapDBTemplateToDomainFull(row, 0)
	return &t, nil
}

func (r *templateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteProjectTemplate(ctx, id)
}

func (r *templateRepository) IsInUse(ctx context.Context, id uuid.UUID) (bool, error) {
	inUse, err := r.q.IsTemplateInUse(ctx, id)
	if err != nil {
		return false, errctx.Wrap(err, "IsInUse", "id", id)
	}
	return inUse, nil
}

// --- Boards ---

func (r *templateRepository) ListBoardsByTemplateID(ctx context.Context, templateID uuid.UUID) ([]db.ListTemplateBoardsByTemplateIDRow, error) {
	return r.q.ListTemplateBoardsByTemplateID(ctx, uuid.NullUUID{UUID: templateID, Valid: true})
}

func (r *templateRepository) GetBoardByID(ctx context.Context, id uuid.UUID) (db.GetTemplateBoardByIDRow, error) {
	return r.q.GetTemplateBoardByID(ctx, id)
}

func (r *templateRepository) CreateBoard(ctx context.Context, params db.CreateTemplateBoardParams) (db.ListTemplateBoardsByTemplateIDRow, error) {
	row, err := r.q.CreateTemplateBoard(ctx, params)
	if err != nil {
		return db.ListTemplateBoardsByTemplateIDRow{}, err
	}
	return db.ListTemplateBoardsByTemplateIDRow{
		ID:              row.ID,
		TemplateID:      row.TemplateID,
		Name:            row.Name,
		Description:     row.Description,
		SortOrder:       row.SortOrder,
		PriorityType:    row.PriorityType,
		EstimationUnit:  row.EstimationUnit,
		SwimlaneGroupBy: row.SwimlaneGroupBy,
		PriorityOptions: row.PriorityOptions,
	}, nil
}

func (r *templateRepository) UpdateBoard(ctx context.Context, params db.UpdateTemplateBoardParams) (db.ListTemplateBoardsByTemplateIDRow, error) {
	row, err := r.q.UpdateTemplateBoard(ctx, params)
	if err != nil {
		return db.ListTemplateBoardsByTemplateIDRow{}, err
	}
	return db.ListTemplateBoardsByTemplateIDRow{
		ID:              row.ID,
		TemplateID:      row.TemplateID,
		Name:            row.Name,
		Description:     row.Description,
		SortOrder:       row.SortOrder,
		PriorityType:    row.PriorityType,
		EstimationUnit:  row.EstimationUnit,
		SwimlaneGroupBy: row.SwimlaneGroupBy,
		PriorityOptions: row.PriorityOptions,
	}, nil
}

func (r *templateRepository) DeleteBoard(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTemplateBoardByID(ctx, id)
}

func (r *templateRepository) CountBoardsByTemplateID(ctx context.Context, templateID uuid.UUID) (int32, error) {
	return r.q.CountTemplateBoardsByTemplateID(ctx, uuid.NullUUID{UUID: templateID, Valid: true})
}

func (r *templateRepository) UnsetDefaultBoard(ctx context.Context, templateID uuid.UUID) error {
	return r.q.UnsetDefaultBoardByTemplateID(ctx, uuid.NullUUID{UUID: templateID, Valid: true})
}

func (r *templateRepository) UpdateBoardOrder(ctx context.Context, id uuid.UUID, order int16) error {
	return r.q.UpdateTemplateBoardOrder(ctx, db.UpdateTemplateBoardOrderParams{ID: id, SortOrder: order})
}

// --- Columns ---

func (r *templateRepository) ListColumns(ctx context.Context, boardID uuid.UUID) ([]db.Column, error) {
	return r.q.ListTemplateBoardColumns(ctx, boardID)
}

func (r *templateRepository) GetColumnByID(ctx context.Context, id uuid.UUID) (db.Column, error) {
	return r.q.GetTemplateBoardColumnByID(ctx, id)
}

func (r *templateRepository) CreateColumn(ctx context.Context, params db.CreateTemplateBoardColumnParams) (db.Column, error) {
	return r.q.CreateTemplateBoardColumn(ctx, params)
}

func (r *templateRepository) UpdateColumn(ctx context.Context, params db.UpdateTemplateBoardColumnParams) (db.Column, error) {
	return r.q.UpdateTemplateBoardColumn(ctx, params)
}

func (r *templateRepository) DeleteColumn(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTemplateBoardColumnByID(ctx, id)
}

func (r *templateRepository) DeleteColumnsByBoardID(ctx context.Context, boardID uuid.UUID) error {
	return r.q.DeleteTemplateBoardColumnsByBoardID(ctx, boardID)
}

func (r *templateRepository) UpdateColumnOrder(ctx context.Context, id uuid.UUID, order int16) error {
	return r.q.UpdateTemplateBoardColumnOrder(ctx, db.UpdateTemplateBoardColumnOrderParams{ID: id, SortOrder: order})
}

// --- Swimlanes ---

func (r *templateRepository) ListSwimlanes(ctx context.Context, boardID uuid.UUID) ([]db.Swimlane, error) {
	return r.q.ListTemplateBoardSwimlanes(ctx, boardID)
}

func (r *templateRepository) GetSwimlaneByID(ctx context.Context, id uuid.UUID) (db.Swimlane, error) {
	return r.q.GetTemplateBoardSwimlaneByID(ctx, id)
}

func (r *templateRepository) CreateSwimlane(ctx context.Context, params db.CreateTemplateBoardSwimlaneParams) (db.Swimlane, error) {
	return r.q.CreateTemplateBoardSwimlane(ctx, params)
}

func (r *templateRepository) UpdateSwimlane(ctx context.Context, id uuid.UUID, wipLimit sql.NullInt16, note string) (db.Swimlane, error) {
	return r.q.UpdateTemplateBoardSwimlane(ctx, db.UpdateTemplateBoardSwimlaneParams{ID: id, WipLimit: wipLimit, Note: note})
}

func (r *templateRepository) DeleteSwimlane(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTemplateBoardSwimlaneByID(ctx, id)
}

func (r *templateRepository) DeleteSwimlanesByBoardID(ctx context.Context, boardID uuid.UUID) error {
	return r.q.DeleteTemplateBoardSwimlanesByBoardID(ctx, boardID)
}

func (r *templateRepository) UpdateSwimlaneOrder(ctx context.Context, id uuid.UUID, order int16) error {
	return r.q.UpdateTemplateBoardSwimlaneOrder(ctx, db.UpdateTemplateBoardSwimlaneOrderParams{ID: id, SortOrder: order})
}

// --- Custom fields ---

func (r *templateRepository) ListCustomFields(ctx context.Context, boardID uuid.UUID) ([]db.BoardField, error) {
	return r.q.ListTemplateBoardFields(ctx, boardID)
}

func (r *templateRepository) GetFieldByID(ctx context.Context, id uuid.UUID) (db.BoardField, error) {
	return r.q.GetTemplateBoardFieldByID(ctx, id)
}

func (r *templateRepository) CreateField(ctx context.Context, params db.CreateTemplateBoardFieldParams) (db.BoardField, error) {
	row, err := r.q.CreateTemplateBoardField(ctx, params)
	if err != nil {
		return db.BoardField{}, err
	}
	return db.BoardField{
		ID:         row.ID,
		BoardID:    row.BoardID,
		Name:       row.Name,
		FieldType:  row.FieldType,
		IsRequired: row.IsRequired,
		Options:    row.Options,
	}, nil
}

func (r *templateRepository) UpdateField(ctx context.Context, params db.UpdateTemplateBoardFieldParams) (db.BoardField, error) {
	row, err := r.q.UpdateTemplateBoardField(ctx, params)
	if err != nil {
		return db.BoardField{}, err
	}
	return db.BoardField{
		ID:         row.ID,
		BoardID:    row.BoardID,
		Name:       row.Name,
		FieldType:  row.FieldType,
		IsRequired: row.IsRequired,
		Options:    row.Options,
	}, nil
}

func (r *templateRepository) DeleteField(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTemplateBoardFieldByID(ctx, id)
}

// --- Helpers ---

func mapListRowToDomain(row db.ListProjectTemplatesRow) domain.ProjectTemplate {
	var descPtr *string
	if row.Description.Valid {
		d := row.Description.String
		descPtr = &d
	}
	return domain.ProjectTemplate{
		ID:          row.ID,
		Name:        row.Name,
		Description: descPtr,
		Type:        domain.ProjectType(row.ProjectType),
		BoardCount:  int(row.BoardCount),
	}
}

func mapDBTemplateToDomainFull(row db.Template, boardCount int) domain.ProjectTemplate {
	var descPtr *string
	if row.Description.Valid {
		d := row.Description.String
		descPtr = &d
	}
	return domain.ProjectTemplate{
		ID:          row.ID,
		Name:        row.Name,
		Description: descPtr,
		Type:        domain.ProjectType(row.ProjectType),
		BoardCount:  boardCount,
	}
}

func OptionsToJSON(options []string) pqtype.NullRawMessage {
	if len(options) == 0 {
		return pqtype.NullRawMessage{}
	}
	data, _ := json.Marshal(options)
	return pqtype.NullRawMessage{RawMessage: data, Valid: true}
}

func JSONToOptions(raw pqtype.NullRawMessage) []string {
	if !raw.Valid {
		return []string{}
	}
	var options []string
	if err := json.Unmarshal(raw.RawMessage, &options); err != nil {
		return []string{}
	}
	return options
}

// --- Project Params ---

func (r *templateRepository) ListProjectParams(ctx context.Context, templateID uuid.UUID) ([]db.ListTemplateProjectParamsRow, error) {
	return r.q.ListTemplateProjectParams(ctx, uuid.NullUUID{UUID: templateID, Valid: true})
}

func (r *templateRepository) GetProjectParamByID(ctx context.Context, id uuid.UUID) (db.GetTemplateProjectParamByIDRow, error) {
	return r.q.GetTemplateProjectParamByID(ctx, id)
}

func (r *templateRepository) CreateProjectParam(ctx context.Context, params db.CreateTemplateProjectParamParams) (db.ListTemplateProjectParamsRow, error) {
	row, err := r.q.CreateTemplateProjectParam(ctx, params)
	if err != nil {
		return db.ListTemplateProjectParamsRow{}, err
	}
	return db.ListTemplateProjectParamsRow{
		ID:         row.ID,
		TemplateID: row.TemplateID,
		Name:       row.Name,
		FieldType:  row.FieldType,
		IsRequired: row.IsRequired,
		Options:    row.Options,
	}, nil
}

func (r *templateRepository) UpdateProjectParam(ctx context.Context, params db.UpdateTemplateProjectParamParams) (db.ListTemplateProjectParamsRow, error) {
	row, err := r.q.UpdateTemplateProjectParam(ctx, params)
	if err != nil {
		return db.ListTemplateProjectParamsRow{}, err
	}
	return db.ListTemplateProjectParamsRow{
		ID:         row.ID,
		TemplateID: row.TemplateID,
		Name:       row.Name,
		FieldType:  row.FieldType,
		IsRequired: row.IsRequired,
		Options:    row.Options,
	}, nil
}

func (r *templateRepository) DeleteProjectParam(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTemplateProjectParamByID(ctx, id)
}

// --- Roles ---

func (r *templateRepository) ListRoles(ctx context.Context, templateID uuid.UUID) ([]db.ListTemplateRolesRow, error) {
	return r.q.ListTemplateRoles(ctx, uuid.NullUUID{UUID: templateID, Valid: true})
}

func (r *templateRepository) GetRoleByID(ctx context.Context, id uuid.UUID) (db.GetTemplateRoleByIDRow, error) {
	return r.q.GetTemplateRoleByID(ctx, id)
}

func (r *templateRepository) CreateRole(ctx context.Context, params db.CreateTemplateRoleParams) (db.ListTemplateRolesRow, error) {
	row, err := r.q.CreateTemplateRole(ctx, params)
	if err != nil {
		return db.ListTemplateRolesRow{}, err
	}
	return db.ListTemplateRolesRow{
		ID:          row.ID,
		TemplateID:  row.TemplateID,
		Name:        row.Name,
		Description: row.Description,
		IsAdmin:     row.IsAdmin,
		SortOrder:   row.SortOrder,
	}, nil
}

func (r *templateRepository) UpdateRole(ctx context.Context, params db.UpdateTemplateRoleParams) (db.ListTemplateRolesRow, error) {
	row, err := r.q.UpdateTemplateRole(ctx, params)
	if err != nil {
		return db.ListTemplateRolesRow{}, err
	}
	return db.ListTemplateRolesRow{
		ID:          row.ID,
		TemplateID:  row.TemplateID,
		Name:        row.Name,
		Description: row.Description,
		IsAdmin:     row.IsAdmin,
		SortOrder:   row.SortOrder,
	}, nil
}

func (r *templateRepository) UpdateRoleOrder(ctx context.Context, id uuid.UUID, order int32) error {
	return r.q.UpdateTemplateRoleOrder(ctx, db.UpdateTemplateRoleOrderParams{ID: id, SortOrder: order})
}

func (r *templateRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	_ = r.q.DeleteTemplateRolePermissionsByRoleID(ctx, id)
	return r.q.DeleteTemplateRoleByID(ctx, id)
}

func (r *templateRepository) ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]db.RolePermission, error) {
	return r.q.ListTemplateRolePermissions(ctx, roleID)
}

func (r *templateRepository) UpsertRolePermission(ctx context.Context, roleID uuid.UUID, area, access string) error {
	return r.q.UpsertTemplateRolePermission(ctx, db.UpsertTemplateRolePermissionParams{RoleID: roleID, PermissionCode: area, Access: sql.NullString{String: access, Valid: access != ""}})
}

func (r *templateRepository) DeleteRolePermissions(ctx context.Context, roleID uuid.UUID) error {
	return r.q.DeleteTemplateRolePermissionsByRoleID(ctx, roleID)
}
