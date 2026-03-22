package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/pkg/errctx"
)

type TemplateRepository interface {
	List(ctx context.Context) ([]domain.ProjectTemplate, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectTemplate, error)
	Create(ctx context.Context, name string, description *string, projectType string) (*domain.ProjectTemplate, error)
	Update(ctx context.Context, id uuid.UUID, name string, description *string) (*domain.ProjectTemplate, error)
	Delete(ctx context.Context, id uuid.UUID) error
	IsInUse(ctx context.Context, id uuid.UUID) (bool, error)

	// Boards
	ListBoardsByTemplateID(ctx context.Context, templateID uuid.UUID) ([]db.TemplateBoard, error)
	GetBoardByID(ctx context.Context, id uuid.UUID) (db.TemplateBoard, error)
	CreateBoard(ctx context.Context, params db.CreateTemplateBoardParams) (db.TemplateBoard, error)
	UpdateBoard(ctx context.Context, params db.UpdateTemplateBoardParams) (db.TemplateBoard, error)
	DeleteBoard(ctx context.Context, id uuid.UUID) error
	CountBoardsByTemplateID(ctx context.Context, templateID uuid.UUID) (int32, error)
	UnsetDefaultBoard(ctx context.Context, templateID uuid.UUID) error
	UpdateBoardOrder(ctx context.Context, id uuid.UUID, order int32) error

	// Columns
	ListColumns(ctx context.Context, boardID uuid.UUID) ([]db.TemplateBoardColumn, error)
	GetColumnByID(ctx context.Context, id uuid.UUID) (db.TemplateBoardColumn, error)
	CreateColumn(ctx context.Context, params db.CreateTemplateBoardColumnParams) (db.TemplateBoardColumn, error)
	UpdateColumn(ctx context.Context, params db.UpdateTemplateBoardColumnParams) (db.TemplateBoardColumn, error)
	DeleteColumn(ctx context.Context, id uuid.UUID) error
	DeleteColumnsByBoardID(ctx context.Context, boardID uuid.UUID) error
	UpdateColumnOrder(ctx context.Context, id uuid.UUID, order int32) error

	// Swimlanes
	ListSwimlanes(ctx context.Context, boardID uuid.UUID) ([]db.TemplateBoardSwimlane, error)
	GetSwimlaneByID(ctx context.Context, id uuid.UUID) (db.TemplateBoardSwimlane, error)
	CreateSwimlane(ctx context.Context, params db.CreateTemplateBoardSwimlaneParams) (db.TemplateBoardSwimlane, error)
	UpdateSwimlane(ctx context.Context, id uuid.UUID, wipLimit sql.NullInt32, note string) (db.TemplateBoardSwimlane, error)
	DeleteSwimlane(ctx context.Context, id uuid.UUID) error
	DeleteSwimlanesByBoardID(ctx context.Context, boardID uuid.UUID) error
	UpdateSwimlaneOrder(ctx context.Context, id uuid.UUID, order int32) error

	// Priority values
	ListPriorityValues(ctx context.Context, boardID uuid.UUID) ([]db.TemplateBoardPriorityValue, error)
	CreatePriorityValue(ctx context.Context, params db.CreateTemplateBoardPriorityValueParams) (db.TemplateBoardPriorityValue, error)
	DeletePriorityValuesByBoardID(ctx context.Context, boardID uuid.UUID) error

	// Custom fields
	ListCustomFields(ctx context.Context, boardID uuid.UUID) ([]db.TemplateBoardField, error)
	GetFieldByID(ctx context.Context, id uuid.UUID) (db.TemplateBoardField, error)
	CreateField(ctx context.Context, params db.CreateTemplateBoardFieldParams) (db.TemplateBoardField, error)
	UpdateField(ctx context.Context, params db.UpdateTemplateBoardFieldParams) (db.TemplateBoardField, error)
	DeleteField(ctx context.Context, id uuid.UUID) error
	UpdateFieldOrder(ctx context.Context, id uuid.UUID, order int32) error

	// Project params
	ListProjectParams(ctx context.Context, templateID uuid.UUID) ([]db.TemplateProjectParam, error)
	GetProjectParamByID(ctx context.Context, id uuid.UUID) (db.TemplateProjectParam, error)
	CreateProjectParam(ctx context.Context, params db.CreateTemplateProjectParamParams) (db.TemplateProjectParam, error)
	UpdateProjectParam(ctx context.Context, params db.UpdateTemplateProjectParamParams) (db.TemplateProjectParam, error)
	DeleteProjectParam(ctx context.Context, id uuid.UUID) error
	UpdateProjectParamOrder(ctx context.Context, id uuid.UUID, order int32) error

	// Roles
	ListRoles(ctx context.Context, templateID uuid.UUID) ([]db.TemplateRole, error)
	GetRoleByID(ctx context.Context, id uuid.UUID) (db.TemplateRole, error)
	CreateRole(ctx context.Context, params db.CreateTemplateRoleParams) (db.TemplateRole, error)
	UpdateRole(ctx context.Context, params db.UpdateTemplateRoleParams) (db.TemplateRole, error)
	DeleteRole(ctx context.Context, id uuid.UUID) error
	UpdateRoleOrder(ctx context.Context, id uuid.UUID, order int32) error
	CountRoles(ctx context.Context, templateID uuid.UUID) (int32, error)
	ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]db.TemplateRolePermission, error)
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, errctx.Wrap(err, "GetByID", "id", id)
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, errctx.Wrap(err, "Update", "id", id)
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

func (r *templateRepository) ListBoardsByTemplateID(ctx context.Context, templateID uuid.UUID) ([]db.TemplateBoard, error) {
	return r.q.ListTemplateBoardsByTemplateID(ctx, templateID)
}

func (r *templateRepository) GetBoardByID(ctx context.Context, id uuid.UUID) (db.TemplateBoard, error) {
	return r.q.GetTemplateBoardByID(ctx, id)
}

func (r *templateRepository) CreateBoard(ctx context.Context, params db.CreateTemplateBoardParams) (db.TemplateBoard, error) {
	return r.q.CreateTemplateBoard(ctx, params)
}

func (r *templateRepository) UpdateBoard(ctx context.Context, params db.UpdateTemplateBoardParams) (db.TemplateBoard, error) {
	return r.q.UpdateTemplateBoard(ctx, params)
}

func (r *templateRepository) DeleteBoard(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTemplateBoardByID(ctx, id)
}

func (r *templateRepository) CountBoardsByTemplateID(ctx context.Context, templateID uuid.UUID) (int32, error) {
	return r.q.CountTemplateBoardsByTemplateID(ctx, templateID)
}

func (r *templateRepository) UnsetDefaultBoard(ctx context.Context, templateID uuid.UUID) error {
	return r.q.UnsetDefaultBoardByTemplateID(ctx, templateID)
}

func (r *templateRepository) UpdateBoardOrder(ctx context.Context, id uuid.UUID, order int32) error {
	return r.q.UpdateTemplateBoardOrder(ctx, db.UpdateTemplateBoardOrderParams{ID: id, Order: order})
}

// --- Columns ---

func (r *templateRepository) ListColumns(ctx context.Context, boardID uuid.UUID) ([]db.TemplateBoardColumn, error) {
	return r.q.ListTemplateBoardColumns(ctx, boardID)
}

func (r *templateRepository) GetColumnByID(ctx context.Context, id uuid.UUID) (db.TemplateBoardColumn, error) {
	return r.q.GetTemplateBoardColumnByID(ctx, id)
}

func (r *templateRepository) CreateColumn(ctx context.Context, params db.CreateTemplateBoardColumnParams) (db.TemplateBoardColumn, error) {
	return r.q.CreateTemplateBoardColumn(ctx, params)
}

func (r *templateRepository) UpdateColumn(ctx context.Context, params db.UpdateTemplateBoardColumnParams) (db.TemplateBoardColumn, error) {
	return r.q.UpdateTemplateBoardColumn(ctx, params)
}

func (r *templateRepository) DeleteColumn(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTemplateBoardColumnByID(ctx, id)
}

func (r *templateRepository) DeleteColumnsByBoardID(ctx context.Context, boardID uuid.UUID) error {
	return r.q.DeleteTemplateBoardColumnsByBoardID(ctx, boardID)
}

func (r *templateRepository) UpdateColumnOrder(ctx context.Context, id uuid.UUID, order int32) error {
	return r.q.UpdateTemplateBoardColumnOrder(ctx, db.UpdateTemplateBoardColumnOrderParams{ID: id, Order: order})
}

// --- Swimlanes ---

func (r *templateRepository) ListSwimlanes(ctx context.Context, boardID uuid.UUID) ([]db.TemplateBoardSwimlane, error) {
	return r.q.ListTemplateBoardSwimlanes(ctx, boardID)
}

func (r *templateRepository) GetSwimlaneByID(ctx context.Context, id uuid.UUID) (db.TemplateBoardSwimlane, error) {
	return r.q.GetTemplateBoardSwimlaneByID(ctx, id)
}

func (r *templateRepository) CreateSwimlane(ctx context.Context, params db.CreateTemplateBoardSwimlaneParams) (db.TemplateBoardSwimlane, error) {
	return r.q.CreateTemplateBoardSwimlane(ctx, params)
}

func (r *templateRepository) UpdateSwimlane(ctx context.Context, id uuid.UUID, wipLimit sql.NullInt32, note string) (db.TemplateBoardSwimlane, error) {
	return r.q.UpdateTemplateBoardSwimlane(ctx, db.UpdateTemplateBoardSwimlaneParams{ID: id, WipLimit: wipLimit, Note: note})
}

func (r *templateRepository) DeleteSwimlane(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTemplateBoardSwimlaneByID(ctx, id)
}

func (r *templateRepository) DeleteSwimlanesByBoardID(ctx context.Context, boardID uuid.UUID) error {
	return r.q.DeleteTemplateBoardSwimlanesByBoardID(ctx, boardID)
}

func (r *templateRepository) UpdateSwimlaneOrder(ctx context.Context, id uuid.UUID, order int32) error {
	return r.q.UpdateTemplateBoardSwimlaneOrder(ctx, db.UpdateTemplateBoardSwimlaneOrderParams{ID: id, Order: order})
}

// --- Priority values ---

func (r *templateRepository) ListPriorityValues(ctx context.Context, boardID uuid.UUID) ([]db.TemplateBoardPriorityValue, error) {
	return r.q.ListTemplateBoardPriorityValues(ctx, boardID)
}

func (r *templateRepository) CreatePriorityValue(ctx context.Context, params db.CreateTemplateBoardPriorityValueParams) (db.TemplateBoardPriorityValue, error) {
	return r.q.CreateTemplateBoardPriorityValue(ctx, params)
}

func (r *templateRepository) DeletePriorityValuesByBoardID(ctx context.Context, boardID uuid.UUID) error {
	return r.q.DeleteTemplateBoardPriorityValuesByBoardID(ctx, boardID)
}

// --- Custom fields ---

func (r *templateRepository) ListCustomFields(ctx context.Context, boardID uuid.UUID) ([]db.TemplateBoardField, error) {
	return r.q.ListTemplateBoardCustomFields(ctx, boardID)
}

func (r *templateRepository) GetFieldByID(ctx context.Context, id uuid.UUID) (db.TemplateBoardField, error) {
	return r.q.GetTemplateBoardFieldByID(ctx, id)
}

func (r *templateRepository) CreateField(ctx context.Context, params db.CreateTemplateBoardFieldParams) (db.TemplateBoardField, error) {
	return r.q.CreateTemplateBoardField(ctx, params)
}

func (r *templateRepository) UpdateField(ctx context.Context, params db.UpdateTemplateBoardFieldParams) (db.TemplateBoardField, error) {
	return r.q.UpdateTemplateBoardField(ctx, params)
}

func (r *templateRepository) DeleteField(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTemplateBoardFieldByID(ctx, id)
}

func (r *templateRepository) UpdateFieldOrder(ctx context.Context, id uuid.UUID, order int32) error {
	return r.q.UpdateTemplateBoardFieldOrder(ctx, db.UpdateTemplateBoardFieldOrderParams{ID: id, Order: order})
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
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func mapDBTemplateToDomainFull(row db.ProjectTemplate, boardCount int) domain.ProjectTemplate {
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
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
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

func (r *templateRepository) ListProjectParams(ctx context.Context, templateID uuid.UUID) ([]db.TemplateProjectParam, error) {
	return r.q.ListTemplateProjectParams(ctx, templateID)
}

func (r *templateRepository) GetProjectParamByID(ctx context.Context, id uuid.UUID) (db.TemplateProjectParam, error) {
	return r.q.GetTemplateProjectParamByID(ctx, id)
}

func (r *templateRepository) CreateProjectParam(ctx context.Context, params db.CreateTemplateProjectParamParams) (db.TemplateProjectParam, error) {
	return r.q.CreateTemplateProjectParam(ctx, params)
}

func (r *templateRepository) UpdateProjectParam(ctx context.Context, params db.UpdateTemplateProjectParamParams) (db.TemplateProjectParam, error) {
	return r.q.UpdateTemplateProjectParam(ctx, params)
}

func (r *templateRepository) DeleteProjectParam(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTemplateProjectParamByID(ctx, id)
}

func (r *templateRepository) UpdateProjectParamOrder(ctx context.Context, id uuid.UUID, order int32) error {
	return r.q.UpdateTemplateProjectParamOrder(ctx, db.UpdateTemplateProjectParamOrderParams{ID: id, Order: order})
}

// --- Roles ---

func (r *templateRepository) ListRoles(ctx context.Context, templateID uuid.UUID) ([]db.TemplateRole, error) {
	return r.q.ListTemplateRoles(ctx, templateID)
}

func (r *templateRepository) GetRoleByID(ctx context.Context, id uuid.UUID) (db.TemplateRole, error) {
	return r.q.GetTemplateRoleByID(ctx, id)
}

func (r *templateRepository) CreateRole(ctx context.Context, params db.CreateTemplateRoleParams) (db.TemplateRole, error) {
	return r.q.CreateTemplateRole(ctx, params)
}

func (r *templateRepository) UpdateRole(ctx context.Context, params db.UpdateTemplateRoleParams) (db.TemplateRole, error) {
	return r.q.UpdateTemplateRole(ctx, params)
}

func (r *templateRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	_ = r.q.DeleteTemplateRolePermissionsByRoleID(ctx, id)
	return r.q.DeleteTemplateRoleByID(ctx, id)
}

func (r *templateRepository) UpdateRoleOrder(ctx context.Context, id uuid.UUID, order int32) error {
	return r.q.UpdateTemplateRoleOrder(ctx, db.UpdateTemplateRoleOrderParams{ID: id, Order: order})
}

func (r *templateRepository) CountRoles(ctx context.Context, templateID uuid.UUID) (int32, error) {
	return r.q.CountTemplateRolesByTemplateID(ctx, templateID)
}

func (r *templateRepository) ListRolePermissions(ctx context.Context, roleID uuid.UUID) ([]db.TemplateRolePermission, error) {
	return r.q.ListTemplateRolePermissions(ctx, roleID)
}

func (r *templateRepository) UpsertRolePermission(ctx context.Context, roleID uuid.UUID, area, access string) error {
	return r.q.UpsertTemplateRolePermission(ctx, db.UpsertTemplateRolePermissionParams{RoleID: roleID, Area: area, Access: access})
}

func (r *templateRepository) DeleteRolePermissions(ctx context.Context, roleID uuid.UUID) error {
	return r.q.DeleteTemplateRolePermissionsByRoleID(ctx, roleID)
}
