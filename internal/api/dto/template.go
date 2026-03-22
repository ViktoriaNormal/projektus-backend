package dto

import (
	"time"

	"github.com/google/uuid"
)

// --- Responses ---

type ProjectTemplateListResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	ProjectType string    `json:"projectType"`
	BoardCount  int       `json:"boardCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ProjectTemplateResponse struct {
	ID                  uuid.UUID                       `json:"id"`
	Name                string                          `json:"name"`
	Description         string                          `json:"description,omitempty"`
	ProjectType         string                          `json:"projectType"`
	CreatedAt           time.Time                       `json:"createdAt"`
	UpdatedAt           time.Time                       `json:"updatedAt"`
	Boards              []TemplateBoardResponse          `json:"boards"`
	CustomProjectParams []TemplateProjectParamResponse   `json:"customProjectParams"`
	Roles               []TemplateRoleResponse           `json:"roles"`
}

type TemplateProjectParamResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	FieldType  string    `json:"fieldType"`
	IsSystem   bool      `json:"isSystem"`
	IsRequired bool      `json:"isRequired"`
	Order      int32     `json:"order"`
	Options    []string  `json:"options"`
}

type TemplateRoleResponse struct {
	ID          uuid.UUID                       `json:"id"`
	Name        string                          `json:"name"`
	Description string                          `json:"description"`
	IsDefault   bool                            `json:"isDefault"`
	Order       int32                           `json:"order"`
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
	IsDefault       bool                             `json:"isDefault"`
	Order           int32                            `json:"order"`
	PriorityType    string                           `json:"priorityType"`
	EstimationUnit  string                           `json:"estimationUnit"`
	SwimlaneGroupBy *string                          `json:"swimlaneGroupBy"`
	Columns         []TemplateBoardColumnResponse    `json:"columns"`
	Swimlanes       []TemplateBoardSwimlaneResponse  `json:"swimlanes"`
	PriorityValues  []TemplateBoardPriorityValueResponse `json:"priorityValues"`
	CustomFields    []TemplateBoardCustomFieldResponse    `json:"customFields"`
}

type TemplateBoardColumnResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	SystemType string    `json:"systemType"`
	WipLimit   *int32    `json:"wipLimit"`
	Order      int32     `json:"order"`
	IsLocked   bool      `json:"isLocked"`
	Note       *string   `json:"note"`
}

type TemplateBoardSwimlaneResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	WipLimit *int32    `json:"wipLimit"`
	Order    int32     `json:"order"`
	Note     *string   `json:"note"`
}

type TemplateBoardPriorityValueResponse struct {
	ID    uuid.UUID `json:"id"`
	Value string    `json:"value"`
	Order int32     `json:"order"`
}

type TemplateBoardCustomFieldResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	FieldType  string    `json:"fieldType"`
	IsSystem   bool      `json:"isSystem"`
	IsRequired bool      `json:"isRequired"`
	Order      int32     `json:"order"`
	Options    []string  `json:"options"`
}

// --- Requests ---

type CreateTemplateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ProjectType string `json:"projectType" binding:"required,oneof=scrum kanban"`
}

type UpdateTemplateRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type CreateTemplateBoardRequest struct {
	Name            string  `json:"name" binding:"required"`
	Description     string  `json:"description"`
	IsDefault       bool    `json:"isDefault"`
	PriorityType    string  `json:"priorityType" binding:"required,oneof=priority service_class"`
	EstimationUnit  string  `json:"estimationUnit" binding:"required,oneof=story_points time"`
	SwimlaneGroupBy *string `json:"swimlaneGroupBy"`
}

type UpdateTemplateBoardRequest struct {
	Name            *string `json:"name"`
	Description     *string `json:"description"`
	IsDefault       *bool   `json:"isDefault"`
	Order           *int32  `json:"order"`
	PriorityType    *string `json:"priorityType"`
	EstimationUnit  *string `json:"estimationUnit"`
	SwimlaneGroupBy *string `json:"swimlaneGroupBy"`
}

type ReorderRequest struct {
	Orders []OrderItem `json:"orders" binding:"required"`
}

type OrderItem struct {
	ID    uuid.UUID `json:"boardId,omitempty"`
	Order int32     `json:"order"`
}

type ColumnOrderItem struct {
	ColumnID uuid.UUID `json:"columnId"`
	Order    int32     `json:"order"`
}

type SwimlaneOrderItem struct {
	SwimlaneID uuid.UUID `json:"swimlaneId"`
	Order      int32     `json:"order"`
}

type FieldOrderItem struct {
	FieldID uuid.UUID `json:"fieldId"`
	Order   int32     `json:"order"`
}

type ReorderColumnsRequest struct {
	Orders []ColumnOrderItem `json:"orders" binding:"required"`
}

type ReorderSwimlanesRequest struct {
	Orders []SwimlaneOrderItem `json:"orders" binding:"required"`
}

type ReorderFieldsRequest struct {
	Orders []FieldOrderItem `json:"orders" binding:"required"`
}

type CreateTemplateBoardColumnRequest struct {
	Name       string  `json:"name" binding:"required"`
	SystemType string  `json:"systemType" binding:"required,oneof=initial in_progress completed"`
	WipLimit   *int32  `json:"wipLimit"`
	Order      int32   `json:"order"`
	Note       *string `json:"note"`
}

type UpdateTemplateBoardColumnRequest struct {
	Name       *string `json:"name"`
	SystemType *string `json:"systemType"`
	WipLimit   *int32  `json:"wipLimit"`
	Note       *string `json:"note"`
}

type UpdateTemplateBoardSwimlaneRequest struct {
	WipLimit *int32  `json:"wipLimit"`
	Note     *string `json:"note"`
}

type PriorityValueItem struct {
	Value string `json:"value" binding:"required"`
	Order int32  `json:"order"`
}

type CreateTemplateBoardCustomFieldRequest struct {
	Name       string   `json:"name" binding:"required"`
	FieldType  string   `json:"fieldType" binding:"required,oneof=text number datetime select multiselect checkbox user"`
	IsRequired bool     `json:"isRequired"`
	Order      int32    `json:"order"`
	Options    []string `json:"options"`
}

type UpdateTemplateBoardCustomFieldRequest struct {
	Name       *string  `json:"name"`
	IsRequired *bool    `json:"isRequired"`
	Options    []string `json:"options"`
}

// --- Project Params ---

type CreateTemplateProjectParamRequest struct {
	Name       string   `json:"name" binding:"required"`
	FieldType  string   `json:"fieldType" binding:"required,oneof=text number datetime select multiselect checkbox user"`
	IsRequired bool     `json:"isRequired"`
	Order      int32    `json:"order"`
	Options    []string `json:"options"`
}

type UpdateTemplateProjectParamRequest struct {
	Name       *string  `json:"name"`
	IsRequired *bool    `json:"isRequired"`
	Options    []string `json:"options"`
}

type ParamOrderItem struct {
	ParamID uuid.UUID `json:"paramId"`
	Order   int32     `json:"order"`
}

type ReorderParamsRequest struct {
	Orders []ParamOrderItem `json:"orders" binding:"required"`
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

type RoleOrderItem struct {
	RoleID uuid.UUID `json:"roleId"`
	Order  int32     `json:"order"`
}

type ReorderRolesRequest struct {
	Orders []RoleOrderItem `json:"orders" binding:"required"`
}

// --- References ---

type ReferencesResponse struct {
	ColumnSystemTypes           []ReferenceColumnType                `json:"columnSystemTypes"`
	TaskStatusTypes             []ReferenceTaskStatusType            `json:"taskStatusTypes"`
	FieldTypes                  []ReferenceKeyName                   `json:"fieldTypes"`
	EstimationUnits             []ReferenceAvailable                 `json:"estimationUnits"`
	SwimlaneGroupOptions        []ReferenceAvailable                 `json:"swimlaneGroupOptions"`
	SwimlaneGroupableFieldTypes []string                             `json:"swimlaneGroupableFieldTypes"`
	PriorityTypeOptions         []ReferencePriorityType              `json:"priorityTypeOptions"`
	SystemTaskFields            []ReferenceSystemField               `json:"systemTaskFields"`
	SystemProjectParams         []ReferenceSystemProjectParam        `json:"systemProjectParams"`
	PermissionAreas             []ReferencePermissionArea             `json:"permissionAreas"`
	AccessLevels                []ReferenceKeyName                   `json:"accessLevels"`
}

type ReferenceSystemProjectParam struct {
	Key        string   `json:"key"`
	Name       string   `json:"name"`
	FieldType  string   `json:"fieldType"`
	IsRequired bool     `json:"isRequired"`
	Options    []string `json:"options"`
}

type ReferencePermissionArea struct {
	Area         string   `json:"area"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	AvailableFor []string `json:"availableFor"`
}

type ReferenceColumnType struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Order       int    `json:"order"`
}

type ReferenceTaskStatusType struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	IsColumnType bool   `json:"isColumnType"`
}

type ReferenceKeyName struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type ReferenceAvailable struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	AvailableFor []string `json:"availableFor"`
}

type ReferencePriorityType struct {
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	AvailableFor  []string `json:"availableFor"`
	DefaultValues []string `json:"defaultValues"`
}

type ReferenceSystemField struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	FieldType    string   `json:"fieldType"`
	AvailableFor []string `json:"availableFor"`
	Description  string   `json:"description,omitempty"`
}
