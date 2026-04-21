package domain

import (
	"time"

	"github.com/google/uuid"
)

type ProjectTemplate struct {
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`
	Description *string     `json:"description,omitempty"`
	Type        ProjectType `json:"project_type"`
	BoardCount  int         `json:"board_count"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type TemplateBoard struct {
	ID              uuid.UUID `json:"id"`
	TemplateID      uuid.UUID `json:"template_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	IsDefault       bool      `json:"is_default"`
	Order           int32     `json:"order"`
	PriorityType    string    `json:"priority_type"`
	EstimationUnit  string    `json:"estimation_unit"`
	SwimlaneGroupBy string   `json:"swimlane_group_by"`
	PriorityOptions []string `json:"priority_options,omitempty"`

	Columns      []TemplateBoardColumn      `json:"columns"`
	Swimlanes    []TemplateBoardSwimlane    `json:"swimlanes"`
	CustomFields []TemplateBoardCustomField `json:"custom_fields"`
}

type TemplateBoardColumn struct {
	ID         uuid.UUID `json:"id"`
	BoardID    uuid.UUID `json:"board_id"`
	Name       string    `json:"name"`
	SystemType string    `json:"system_type"`
	WipLimit   *int32    `json:"wip_limit"`
	Order      int32     `json:"order"`
	IsLocked   bool      `json:"is_locked"`
	Note       *string   `json:"note"`
}

type TemplateBoardSwimlane struct {
	ID       uuid.UUID `json:"id"`
	BoardID  uuid.UUID `json:"board_id"`
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	WipLimit *int32    `json:"wip_limit"`
	Order    int32     `json:"order"`
	Note     *string   `json:"note"`
}

type TemplateBoardCustomField struct {
	ID         uuid.UUID `json:"id"`
	BoardID    uuid.UUID `json:"board_id"`
	Name       string    `json:"name"`
	FieldType  string    `json:"field_type"`
	IsSystem   bool      `json:"is_system"`
	IsRequired bool      `json:"is_required"`
	Options    []string  `json:"options"`
}

type TemplateProjectParam struct {
	ID         uuid.UUID `json:"id"`
	TemplateID uuid.UUID `json:"template_id"`
	Name       string    `json:"name"`
	FieldType  string    `json:"field_type"`
	IsSystem   bool      `json:"is_system"`
	IsRequired bool      `json:"is_required"`
	Options    []string  `json:"options"`
}

type TemplateRole struct {
	ID          uuid.UUID                `json:"id"`
	TemplateID  uuid.UUID                `json:"template_id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	IsAdmin     bool                     `json:"is_admin"`
	Order       int32                    `json:"order"`
	Permissions []TemplateRolePermission `json:"permissions"`
}

type TemplateRolePermission struct {
	Area   string `json:"area"`
	Access string `json:"access"`
}

// References — all lookup data loaded from DB
type References struct {
	ColumnSystemTypes    []RefColumnSystemType
	FieldTypes           []FieldTypeDefinition
	EstimationUnits      []RefAvailable
	PriorityTypeOptions  []RefPriorityType
	ProjectStatuses      []RefKeyName
	PermissionAreas      []RefPermissionArea
	AccessLevels         []RefKeyName
}

type RefColumnSystemType struct {
	Key         string
	Name        string
	Description string
}

type RefTaskStatusType struct {
	Key          string
	Name         string
	Description  string
	IsColumnType bool
}

type RefKeyName struct {
	Key  string
	Name string
}

type RefAvailable struct {
	Key          string
	Name         string
	AvailableFor []string
}

type RefPriorityType struct {
	Key           string
	Name          string
	AvailableFor  []string
	DefaultValues []string
}

// DefaultColumnDef — определение колонки по умолчанию для доски.
type DefaultColumnDef struct {
	Name       string
	SystemType string
	IsLocked   bool
}

// DefaultBoardFieldDef — определение системного поля доски (единый источник правды).
type DefaultBoardFieldDef struct {
	Key          string   // "title", "priority", "estimation", "deadline", ...
	Name         string
	FieldType    string
	IsRequired   bool
	Options      []string // статические options (для priority/estimation заменяются динамически)
	AvailableFor []string
}

type RefPermissionArea struct {
	Area         string
	Name         string
	Description  string
	AvailableFor []string
}

// Domain errors for templates
var (
	ErrTemplateNotFound   = ErrNotFound
	ErrTemplateInUse      = ErrConflict
	ErrLastBoard          = ErrInvalidInput
	ErrDefaultBoard       = ErrInvalidInput
	ErrColumnLocked       = ErrInvalidInput
	ErrInvalidColumnOrder = ErrInvalidInput
	ErrDefaultRole        = ErrInvalidInput
)
