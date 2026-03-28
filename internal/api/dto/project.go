package dto

import "github.com/google/uuid"

type CreateProjectRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	ProjectType string  `json:"project_type" binding:"required"` // scrum | kanban
	OwnerID     *string `json:"owner_id"`                        // uuid, опционально — по умолчанию текущий пользователь
}

type UpdateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	OwnerID     *string `json:"owner_id"`
}

type ProjectOwnerResponse struct {
	ID        string  `json:"id"`
	FullName  string  `json:"full_name"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Email     string  `json:"email"`
}

type ProjectReferencesResponse struct {
	ColumnSystemTypes    []ReferenceColumnType     `json:"column_system_types"`
	FieldTypes           []ReferenceFieldType      `json:"field_types"`
	EstimationUnits      []ReferenceAvailable      `json:"estimation_units"`
	PriorityTypeOptions  []ReferencePriorityType   `json:"priority_type_options"`
	PermissionAreas      []ReferencePermissionArea `json:"permission_areas"`
	AccessLevels         []ReferenceKeyName        `json:"access_levels"`
}

type ProjectResponse struct {
	ID          uuid.UUID             `json:"id"`
	Key         string                `json:"key"`
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	ProjectType string                `json:"project_type"`
	OwnerID     uuid.UUID             `json:"owner_id"`
	Status      string                `json:"status"`
	CreatedAt   string                `json:"created_at"`
	Owner       *ProjectOwnerResponse `json:"owner,omitempty"`
}
