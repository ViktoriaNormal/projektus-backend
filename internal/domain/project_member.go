package domain

import "github.com/google/uuid"

// ProjectMemberRoleRef — компактное представление проектной роли участника:
// ID для ссылок, Name для отображения. Используется в ответах API, чтобы фронту
// не приходилось делать отдельный запрос списка ролей ради маппинга id → name.
type ProjectMemberRoleRef struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ProjectMember struct {
	ID        uuid.UUID              `json:"id"`
	ProjectID uuid.UUID              `json:"project_id"`
	UserID    uuid.UUID              `json:"user_id"`
	Roles     []ProjectMemberRoleRef `json:"roles,omitempty"`
}
