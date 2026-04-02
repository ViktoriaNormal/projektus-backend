package dto

import (
	"github.com/google/uuid"
)

// --- Responses ---

type ProjectTemplateListResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	ProjectType string    `json:"project_type"`
	BoardCount  int       `json:"board_count"`
}

type ProjectTemplateResponse struct {
	ID          uuid.UUID                       `json:"id"`
	Name        string                          `json:"name"`
	Description string                          `json:"description,omitempty"`
	ProjectType string                          `json:"project_type"`
	Boards      []TemplateBoardResponse          `json:"boards"`
	Params      []TemplateProjectParamResponse   `json:"params"`
	Roles       []TemplateRoleResponse           `json:"roles"`
}

type TemplateProjectParamResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	FieldType  string    `json:"field_type"`
	IsSystem   bool      `json:"is_system"`
	IsRequired bool      `json:"is_required"`
	Options    []string  `json:"options"`
}

type TemplateRoleResponse struct {
	ID          uuid.UUID                       `json:"id"`
	Name        string                          `json:"name"`
	Description string                          `json:"description"`
	IsAdmin     bool                            `json:"is_admin"`
	Permissions []TemplateRolePermissionResponse `json:"permissions"`
}

type TemplateRolePermissionResponse struct {
	Area   string `json:"area"`
	Access string `json:"access"`
}

type TemplateBoardResponse struct {
	ID              uuid.UUID                        `json:"id"`
	Name            string                           `json:"name"`
	Description     string                           `json:"description,omitempty"`
	IsDefault       bool                             `json:"is_default"`
	Order           int32                            `json:"order"`
	PriorityType    string                           `json:"priority_type"`
	EstimationUnit  string                           `json:"estimation_unit"`
	SwimlaneGroupBy *string                          `json:"swimlane_group_by"`
	Columns      []TemplateBoardColumnResponse       `json:"columns"`
	Swimlanes    []TemplateBoardSwimlaneResponse     `json:"swimlanes"`
	Fields       []TemplateBoardCustomFieldResponse   `json:"fields"`
}

type TemplateBoardColumnResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	SystemType string    `json:"system_type"`
	WipLimit   *int32    `json:"wip_limit"`
	Order      int32     `json:"order"`
	IsLocked   bool      `json:"is_locked"`
	Note       *string   `json:"note"`
}

type TemplateBoardSwimlaneResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	WipLimit *int32    `json:"wip_limit"`
	Order    int32     `json:"order"`
	Note     *string   `json:"note"`
}

type TemplateBoardCustomFieldResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	FieldType  string    `json:"field_type"`
	IsSystem   bool      `json:"is_system"`
	IsRequired bool      `json:"is_required"`
	Options    []string  `json:"options"`
}

// --- Requests ---

type CreateTemplateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ProjectType string `json:"project_type" binding:"required,oneof=scrum kanban"`
}

type UpdateTemplateRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type CreateTemplateBoardRequest struct {
	Name            string  `json:"name" binding:"required"`
	Description     string  `json:"description"`
	IsDefault       bool    `json:"is_default"`
	PriorityType    string  `json:"priority_type" binding:"omitempty,oneof=priority service_class"`
	EstimationUnit  string  `json:"estimation_unit" binding:"omitempty,oneof=story_points time"`
	SwimlaneGroupBy *string `json:"swimlane_group_by"`
}

type UpdateTemplateBoardRequest struct {
	Name            *string               `json:"name"`
	Description     NullableField[string] `json:"description"`
	IsDefault       *bool                 `json:"is_default"`
	Order           *int32                `json:"order"`
	PriorityType    *string               `json:"priority_type"`
	EstimationUnit  *string               `json:"estimation_unit"`
	SwimlaneGroupBy NullableField[string] `json:"swimlane_group_by"`
}

type ReorderRequest struct {
	Orders []OrderItem `json:"orders" binding:"required"`
}

type OrderItem struct {
	ID    uuid.UUID `json:"board_id,omitempty"`
	Order int32     `json:"order"`
}

type ColumnOrderItem struct {
	ColumnID uuid.UUID `json:"column_id"`
	Order    int32     `json:"order"`
}

type SwimlaneOrderItem struct {
	SwimlaneID uuid.UUID `json:"swimlane_id"`
	Order      int32     `json:"order"`
}

type ReorderColumnsRequest struct {
	Orders []ColumnOrderItem `json:"orders" binding:"required"`
}

type ReorderSwimlanesRequest struct {
	Orders []SwimlaneOrderItem `json:"orders" binding:"required"`
}

type CreateTemplateBoardColumnRequest struct {
	Name       string  `json:"name" binding:"required"`
	SystemType string  `json:"system_type" binding:"required,oneof=initial in_progress completed"`
	WipLimit   *int32  `json:"wip_limit"`
	Order      int32   `json:"order"`
	Note       *string `json:"note"`
}

type UpdateTemplateBoardColumnRequest struct {
	Name       *string               `json:"name"`
	SystemType *string               `json:"system_type"`
	WipLimit   NullableField[int32]  `json:"wip_limit"`
	Note       NullableField[string] `json:"note"`
}

type CreateTemplateBoardSwimlaneRequest struct {
	Name     string `json:"name" binding:"required"`
	WipLimit *int32 `json:"wip_limit"`
	Order    int32  `json:"order"`
}

type UpdateTemplateBoardSwimlaneRequest struct {
	WipLimit NullableField[int32]  `json:"wip_limit"`
	Note     NullableField[string] `json:"note"`
}

type CreateTemplateBoardCustomFieldRequest struct {
	Name       string   `json:"name" binding:"required"`
	FieldType  string   `json:"field_type" binding:"required,oneof=text number datetime select multiselect checkbox user user_list sprint sprint_list"`
	IsRequired bool     `json:"is_required"`
	Options    []string `json:"options"`
}

type UpdateTemplateBoardCustomFieldRequest struct {
	Name       *string  `json:"name"`
	IsRequired *bool    `json:"is_required"`
	Options    []string `json:"options"`
}

// --- Project Params ---

type CreateTemplateProjectParamRequest struct {
	Name       string   `json:"name" binding:"required"`
	FieldType  string   `json:"field_type" binding:"required,oneof=text number datetime select multiselect checkbox user user_list"`
	IsRequired bool     `json:"is_required"`
	Options    []string `json:"options"`
}

type UpdateTemplateProjectParamRequest struct {
	Name       *string  `json:"name"`
	IsRequired *bool    `json:"is_required"`
	Options    []string `json:"options"`
}

// --- Roles ---

type RolePermissionInput struct {
	Area   string `json:"area" binding:"required"`
	Access string `json:"access" binding:"required,oneof=full view none"`
}

type CreateTemplateRoleRequest struct {
	Name        string                `json:"name" binding:"required"`
	Description string                `json:"description"`
	Permissions []RolePermissionInput `json:"permissions" binding:"required"`
}

type UpdateTemplateRoleRequest struct {
	Name        *string               `json:"name"`
	Description *string               `json:"description"`
	Permissions []RolePermissionInput  `json:"permissions"`
}

// --- References ---

type ReferencesResponse struct {
	ColumnSystemTypes           []ReferenceColumnType                `json:"column_system_types"`
	FieldTypes                  []ReferenceFieldType                 `json:"field_types"`
	EstimationUnits             []ReferenceAvailable                 `json:"estimation_units"`
	PriorityTypeOptions         []ReferencePriorityType              `json:"priority_type_options"`
	ProjectStatuses             []ReferenceKeyName                   `json:"project_statuses"`
	PermissionAreas             []ReferencePermissionArea             `json:"permission_areas"`
	AccessLevels                []ReferenceKeyName                   `json:"access_levels"`
}

type ReferencePermissionArea struct {
	Area         string   `json:"area"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	AvailableFor []string `json:"available_for"`
}

type ReferenceColumnType struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ReferenceFieldType struct {
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	AvailableFor  []string `json:"available_for"`
	AllowedScopes []string `json:"allowed_scopes"`
}

type ReferenceKeyName struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type ReferenceAvailable struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	AvailableFor []string `json:"available_for"`
}

type ReferencePriorityType struct {
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	AvailableFor  []string `json:"available_for"`
	DefaultValues []string `json:"default_values"`
}

