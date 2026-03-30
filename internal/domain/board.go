package domain

type SystemStatusType string

const (
	StatusInitial    SystemStatusType = "initial"
	StatusInProgress SystemStatusType = "in_progress"
	StatusPaused     SystemStatusType = "paused"
	StatusCompleted  SystemStatusType = "completed"
	StatusCancelled  SystemStatusType = "cancelled"
)

type Board struct {
	ID              string  `json:"id"`
	ProjectID       *string `json:"project_id,omitempty"`
	TemplateID      *string `json:"template_id,omitempty"`
	Name            string  `json:"name"`
	Description     *string `json:"description,omitempty"`
	IsDefault       bool    `json:"is_default"`
	Order           int16   `json:"order"`
	PriorityType    string  `json:"priority_type"`
	EstimationUnit  string  `json:"estimation_unit"`
	SwimlaneGroupBy string  `json:"swimlane_group_by"`
}

type BoardCustomField struct {
	ID          string   `json:"id"`
	BoardID     string   `json:"board_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	FieldType   string   `json:"field_type"`
	IsSystem    bool     `json:"is_system"`
	IsRequired  bool     `json:"is_required"`
	Options     []string `json:"options"`
}

type ProjectRole struct {
	ID              string                 `json:"id"`
	ProjectID       string                 `json:"project_id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	IsAdmin         bool                   `json:"is_admin"`
	Permissions     []ProjectRolePermission `json:"permissions"`
}

type ProjectRolePermission struct {
	Area   string `json:"area"`
	Access string `json:"access"`
}

type Tag struct {
	ID      string `json:"id"`
	BoardID string `json:"board_id"`
	Name    string `json:"name"`
}

type ProjectParam struct {
	ID          string   `json:"id"`
	ProjectID   string   `json:"project_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	FieldType   string   `json:"field_type"`
	IsSystem    bool     `json:"is_system"`
	IsRequired  bool     `json:"is_required"`
	Options     []string `json:"options"`
	Value       *string  `json:"value"`
}

type Column struct {
	ID         string            `json:"id"`
	BoardID    string            `json:"board_id"`
	Name       string            `json:"name"`
	SystemType *SystemStatusType `json:"system_type,omitempty"`
	WipLimit   *int16            `json:"wip_limit,omitempty"`
	Order      int16             `json:"order"`
	IsLocked   bool              `json:"is_locked"`
}

type Swimlane struct {
	ID       string `json:"id"`
	BoardID  string `json:"board_id"`
	Name     string `json:"name"`
	WipLimit *int16 `json:"wip_limit,omitempty"`
	Order    int16  `json:"order"`
}

type Note struct {
	ID         string  `json:"id"`
	ColumnID   *string `json:"column_id,omitempty"`
	SwimlaneID *string `json:"swimlane_id,omitempty"`
	Content    string  `json:"content"`
}
