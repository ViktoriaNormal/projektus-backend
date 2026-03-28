package dto

import "github.com/google/uuid"

type ProjectParamResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	FieldType   string    `json:"field_type"`
	IsSystem    bool      `json:"is_system"`
	IsRequired  bool      `json:"is_required"`
	Order       int32     `json:"order"`
	Options     []string  `json:"options"`
	Value       *string   `json:"value,omitempty"`
}

type CreateProjectParamRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	FieldType   string   `json:"field_type" binding:"required,oneof=text number datetime select multiselect checkbox user user_list"`
	IsRequired  bool     `json:"is_required"`
	Options     []string `json:"options"`
	Value       *string  `json:"value,omitempty"`
}

type UpdateProjectParamRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	IsRequired  *bool    `json:"is_required,omitempty"`
	Options     []string `json:"options,omitempty"`
	Value       *string  `json:"value,omitempty"`
}

type ProjectParamOrderItem struct {
	ParamID uuid.UUID `json:"param_id"`
	Order   int32     `json:"order"`
}

type ReorderProjectParamsRequest struct {
	Orders []ProjectParamOrderItem `json:"orders" binding:"required"`
}
