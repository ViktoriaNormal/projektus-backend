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
	SwimlaneGroupBy string    `json:"swimlane_group_by"`

	Columns        []TemplateBoardColumn        `json:"columns"`
	Swimlanes      []TemplateBoardSwimlane       `json:"swimlanes"`
	PriorityValues []TemplateBoardPriorityValue  `json:"priority_values"`
	CustomFields   []TemplateBoardCustomField    `json:"custom_fields"`
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

type TemplateBoardPriorityValue struct {
	ID      uuid.UUID `json:"id"`
	BoardID uuid.UUID `json:"board_id"`
	Value   string    `json:"value"`
	Order   int32     `json:"order"`
}

type TemplateBoardCustomField struct {
	ID         uuid.UUID `json:"id"`
	BoardID    uuid.UUID `json:"board_id"`
	Name       string    `json:"name"`
	FieldType  string    `json:"field_type"`
	IsSystem   bool      `json:"is_system"`
	IsRequired bool      `json:"is_required"`
	Order      int32     `json:"order"`
	Options    []string  `json:"options"`
}

type TemplateProjectParam struct {
	ID         uuid.UUID `json:"id"`
	TemplateID uuid.UUID `json:"template_id"`
	Name       string    `json:"name"`
	FieldType  string    `json:"field_type"`
	IsRequired bool      `json:"is_required"`
	Order      int32     `json:"order"`
	Options    []string  `json:"options"`
}

type TemplateRole struct {
	ID          uuid.UUID              `json:"id"`
	TemplateID  uuid.UUID              `json:"template_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	IsDefault   bool                   `json:"is_default"`
	Order       int32                  `json:"order"`
	Permissions []TemplateRolePermission `json:"permissions"`
}

type TemplateRolePermission struct {
	Area   string `json:"area"`
	Access string `json:"access"`
}

// References — all lookup data loaded from DB
type References struct {
	ColumnSystemTypes    []RefColumnSystemType
	TaskStatusTypes      []RefTaskStatusType
	FieldTypes           []RefKeyName
	EstimationUnits      []RefAvailable
	SwimlaneGroupOptions []RefAvailable
	PriorityTypeOptions  []RefPriorityType
	SystemTaskFields     []RefSystemField
	SystemProjectParams  []RefSystemProjectParam
	PermissionAreas      []RefPermissionArea
	AccessLevels         []RefKeyName
}

type RefSystemProjectParam struct {
	Key        string
	Name       string
	FieldType  string
	IsRequired bool
	Options    []string
}

type RefColumnSystemType struct {
	Key         string
	Name        string
	Description string
	Order       int
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

type RefSystemField struct {
	Key          string
	Name         string
	FieldType    string
	AvailableFor []string
	Description  string
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
	ErrColumnLocked       = ErrInvalidInput
	ErrInvalidColumnOrder = ErrInvalidInput
	ErrDefaultRole        = ErrInvalidInput
)
