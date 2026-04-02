package dto

import "github.com/google/uuid"

type BoardResponse struct {
	ID              uuid.UUID  `json:"id"`
	ProjectID       *uuid.UUID `json:"project_id,omitempty"`
	Name            string     `json:"name"`
	Description     *string    `json:"description,omitempty"`
	IsDefault       bool       `json:"is_default"`
	Order           int32      `json:"order"`
	PriorityType    string     `json:"priority_type"`
	EstimationUnit  string     `json:"estimation_unit"`
	SwimlaneGroupBy *string    `json:"swimlane_group_by,omitempty"`
}

type CreateBoardRequest struct {
	ProjectID       uuid.UUID `json:"project_id" binding:"required"`
	Name            string    `json:"name" binding:"required"`
	Description     string    `json:"description"`
	Order           int32     `json:"order"`
	PriorityType    string    `json:"priority_type"`
	EstimationUnit  string    `json:"estimation_unit"`
	SwimlaneGroupBy *string   `json:"swimlane_group_by,omitempty"`
}

type UpdateBoardRequest struct {
	Name            *string               `json:"name,omitempty"`
	Description     NullableField[string] `json:"description"`
	IsDefault       *bool                 `json:"is_default,omitempty"`
	Order           *int32                `json:"order,omitempty"`
	PriorityType    *string               `json:"priority_type,omitempty"`
	EstimationUnit  *string               `json:"estimation_unit,omitempty"`
	SwimlaneGroupBy *string               `json:"swimlane_group_by,omitempty"`
}

type ColumnResponse struct {
	ID         uuid.UUID  `json:"id"`
	BoardID    uuid.UUID  `json:"board_id"`
	Name       string     `json:"name"`
	SystemType *string    `json:"system_type,omitempty"`
	WipLimit   *int32     `json:"wip_limit,omitempty"`
	Order      int32      `json:"order"`
	IsLocked   bool       `json:"is_locked"`
}

type CreateColumnRequest struct {
	Name       string  `json:"name" binding:"required"`
	SystemType *string `json:"system_type,omitempty"`
	WipLimit   *int32  `json:"wip_limit,omitempty"`
	Order      int32   `json:"order"`
}

type UpdateColumnRequest struct {
	Name       *string               `json:"name,omitempty"`
	SystemType NullableField[string] `json:"system_type"`
	WipLimit   NullableField[int32]  `json:"wip_limit"`
	Order      *int32                `json:"order,omitempty"`
}

type SwimlaneResponse struct {
	ID       uuid.UUID `json:"id"`
	BoardID  uuid.UUID `json:"board_id"`
	Name     string    `json:"name"`
	WipLimit *int32    `json:"wip_limit,omitempty"`
	Order    int32     `json:"order"`
}

type CreateSwimlaneRequest struct {
	Name     string `json:"name" binding:"required"`
	WipLimit *int32 `json:"wip_limit,omitempty"`
	Order    int32  `json:"order"`
}

type UpdateSwimlaneRequest struct {
	Name     *string              `json:"name,omitempty"`
	WipLimit NullableField[int32] `json:"wip_limit"`
	Order    *int32               `json:"order,omitempty"`
}

type NoteResponse struct {
	ID         uuid.UUID  `json:"id"`
	ColumnID   *uuid.UUID `json:"column_id,omitempty"`
	SwimlaneID *uuid.UUID `json:"swimlane_id,omitempty"`
	Content    string     `json:"content"`
}

type CreateNoteRequest struct {
	Content string `json:"content" binding:"required"`
}

type UpdateNoteRequest struct {
	Content *string `json:"content,omitempty"`
}

// --- Reorder requests ---

type BoardOrderItem struct {
	BoardID uuid.UUID `json:"board_id"`
	Order   int32     `json:"order"`
}

type ReorderBoardsRequest struct {
	ProjectID uuid.UUID        `json:"project_id" binding:"required"`
	Orders    []BoardOrderItem `json:"orders" binding:"required"`
}

type ColumnReorderItem struct {
	ColumnID uuid.UUID `json:"column_id"`
	Order    int32     `json:"order"`
}

type BoardReorderColumnsRequest struct {
	Orders []ColumnReorderItem `json:"orders" binding:"required"`
}

type SwimlaneReorderItem struct {
	SwimlaneID uuid.UUID `json:"swimlane_id"`
	Order      int32     `json:"order"`
}

type BoardReorderSwimlanesRequest struct {
	Orders []SwimlaneReorderItem `json:"orders" binding:"required"`
}

// --- Board custom fields ---

type BoardCustomFieldResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	FieldType  string    `json:"field_type"`
	IsSystem   bool      `json:"is_system"`
	IsRequired bool      `json:"is_required"`
	Options    []string  `json:"options"`
}

type CreateBoardCustomFieldRequest struct {
	Name       string   `json:"name" binding:"required"`
	FieldType  string   `json:"field_type" binding:"required,oneof=text number datetime select multiselect checkbox user user_list sprint sprint_list"`
	IsRequired bool     `json:"is_required"`
	Options    []string `json:"options"`
}

type UpdateBoardCustomFieldRequest struct {
	Name       *string  `json:"name,omitempty"`
	IsRequired *bool    `json:"is_required,omitempty"`
	Options    []string `json:"options,omitempty"`
}

