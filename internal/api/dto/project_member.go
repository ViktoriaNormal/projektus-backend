package dto

import "github.com/google/uuid"

// ProjectMemberRoleRef — роль участника в ответе API: id + отображаемое имя,
// чтобы фронт мог сразу рендерить без отдельного запроса за списком ролей.
type ProjectMemberRoleRef struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ProjectMemberResponse struct {
	ID        uuid.UUID              `json:"id"`
	ProjectID uuid.UUID              `json:"project_id"`
	UserID    uuid.UUID              `json:"user_id"`
	Roles     []ProjectMemberRoleRef `json:"roles,omitempty"`
}

type AddMemberRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
	Roles  []string  `json:"roles"`
}

type UpdateMemberRolesRequest struct {
	Roles []string `json:"roles" binding:"required"`
}
