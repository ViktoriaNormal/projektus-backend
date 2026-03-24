package dto

import "github.com/google/uuid"

// AdminCreateUserRequest — создание пользователя администратором.
type AdminCreateUserRequest struct {
	Username                  string      `json:"username" binding:"required,min=3,max=50"`
	Email                     string      `json:"email" binding:"required,email"`
	FullName                  string      `json:"full_name" binding:"required"`
	Position                  *string     `json:"position"`
	Password                  string      `json:"password" binding:"required,min=6"`
	IsActive                  *bool       `json:"is_active"`
	RoleIDs                   []uuid.UUID `json:"role_ids"`
	OnVacation                *bool       `json:"on_vacation"`
	IsSick                    *bool       `json:"is_sick"`
	AlternativeContactChannel *string     `json:"alternative_contact_channel"`
	AlternativeContactInfo    *string     `json:"alternative_contact_info"`
}

// AdminUpdateUserRequest — обновление пользователя администратором.
type AdminUpdateUserRequest struct {
	Username                  *string      `json:"username" binding:"omitempty,min=3,max=50"`
	Email                     *string      `json:"email" binding:"omitempty,email"`
	FullName                  *string      `json:"full_name"`
	Position                  *string      `json:"position"`
	IsActive                  *bool        `json:"is_active"`
	RoleIDs                   *[]uuid.UUID `json:"role_ids"`
	OnVacation                *bool        `json:"on_vacation"`
	IsSick                    *bool        `json:"is_sick"`
	AlternativeContactChannel *string      `json:"alternative_contact_channel"`
	AlternativeContactInfo    *string      `json:"alternative_contact_info"`
}

// AdminRoleResponse — роль в ответе AdminUser.
type AdminRoleResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AdminUserResponse — пользователь в ответе админки.
type AdminUserResponse struct {
	ID                        string              `json:"id"`
	Username                  string              `json:"username"`
	Email                     string              `json:"email"`
	FullName                  string              `json:"full_name"`
	AvatarURL                 *string             `json:"avatar_url"`
	Position                  *string             `json:"position"`
	OnVacation                bool                `json:"on_vacation"`
	IsSick                    bool                `json:"is_sick"`
	AlternativeContactChannel *string             `json:"alternative_contact_channel"`
	AlternativeContactInfo    *string             `json:"alternative_contact_info"`
	IsActive                  bool                `json:"is_active"`
	Roles                     []AdminRoleResponse `json:"roles"`
	CreatedAt                 string              `json:"created_at"`
}
