package domain

import "github.com/google/uuid"

type SystemStatusType string

const (
	StatusInitial    SystemStatusType = "initial"
	StatusInProgress SystemStatusType = "in_progress"
	StatusPaused     SystemStatusType = "paused"
	StatusCompleted  SystemStatusType = "completed"
	StatusCancelled  SystemStatusType = "cancelled"
)

type Board struct {
	ID              uuid.UUID  `json:"id"`
	ProjectID       *uuid.UUID `json:"project_id,omitempty"`
	TemplateID      *uuid.UUID `json:"template_id,omitempty"`
	Name            string     `json:"name"`
	Description     *string    `json:"description,omitempty"`
	IsDefault       bool       `json:"is_default"`
	Order           int16      `json:"order"`
	PriorityType    string     `json:"priority_type"`
	EstimationUnit  string     `json:"estimation_unit"`
	SwimlaneGroupBy string     `json:"swimlane_group_by"`
	PriorityOptions []string   `json:"priority_options,omitempty"`
}

type BoardCustomField struct {
	ID         uuid.UUID `json:"id"`
	BoardID    uuid.UUID `json:"board_id"`
	Name       string    `json:"name"`
	FieldType  string    `json:"field_type"`
	IsSystem   bool      `json:"is_system"`
	IsRequired bool      `json:"is_required"`
	Options    []string  `json:"options"`
}

type ProjectRole struct {
	ID          uuid.UUID               `json:"id"`
	ProjectID   uuid.UUID               `json:"project_id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	IsAdmin     bool                    `json:"is_admin"`
	Order       int32                   `json:"order"`
	Permissions []ProjectRolePermission `json:"permissions"`
}

type ProjectRolePermission struct {
	Area   string `json:"area"`
	Access string `json:"access"`
}

type Tag struct {
	ID      uuid.UUID `json:"id"`
	BoardID uuid.UUID `json:"board_id"`
	Name    string    `json:"name"`
}

type ProjectParam struct {
	ID         uuid.UUID `json:"id"`
	ProjectID  uuid.UUID `json:"project_id"`
	Name       string    `json:"name"`
	FieldType  string    `json:"field_type"`
	IsSystem   bool      `json:"is_system"`
	IsRequired bool      `json:"is_required"`
	Options    []string  `json:"options"`
	Value      *string   `json:"value"`
}

type Column struct {
	ID         uuid.UUID         `json:"id"`
	BoardID    uuid.UUID         `json:"board_id"`
	Name       string            `json:"name"`
	SystemType *SystemStatusType `json:"system_type,omitempty"`
	WipLimit   *int16            `json:"wip_limit,omitempty"`
	Order      int16             `json:"order"`
	IsLocked   bool              `json:"is_locked"`
}

type Swimlane struct {
	ID       uuid.UUID `json:"id"`
	BoardID  uuid.UUID `json:"board_id"`
	Name     string    `json:"name"`
	WipLimit *int16    `json:"wip_limit,omitempty"`
	Order    int16     `json:"order"`
}

type Note struct {
	ID         uuid.UUID  `json:"id"`
	ColumnID   *uuid.UUID `json:"column_id,omitempty"`
	SwimlaneID *uuid.UUID `json:"swimlane_id,omitempty"`
	Content    string     `json:"content"`
}
