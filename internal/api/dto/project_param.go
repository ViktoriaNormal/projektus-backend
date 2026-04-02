package dto

import "github.com/google/uuid"

type ProjectParamResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	FieldType  string    `json:"field_type"`
	IsSystem   bool      `json:"is_system"`
	IsRequired bool      `json:"is_required"`
	Options    []string  `json:"options"`
	Value      *string   `json:"value,omitempty"`
}

type CreateProjectParamRequest struct {
	Name       string   `json:"name" binding:"required"`
	FieldType  string   `json:"field_type" binding:"required,oneof=text number datetime select multiselect checkbox user user_list"`
	IsRequired bool     `json:"is_required"`
	Options    []string `json:"options"`
	Value      *string  `json:"value,omitempty"`
}

type UpdateProjectParamRequest struct {
	Name       *string               `json:"name,omitempty"`
	IsRequired *bool                 `json:"is_required,omitempty"`
	Options    []string              `json:"options,omitempty"`
	Value      NullableField[string] `json:"value"`
}

