package dto

import "github.com/google/uuid"

// AdminCreateUserRequest — создание пользователя администратором.
type AdminCreateUserRequest struct {
	Username                  string      `json:"username" binding:"required,min=3,max=50"`
	Email                     string      `json:"email" binding:"required,email"`
	FullName                  string      `json:"full_name" binding:"required"`
	Position                  *string     `json:"position,omitempty"`
	Password                  string      `json:"password" binding:"required,min=6"`
	IsActive                  *bool       `json:"is_active,omitempty"`
	RoleIDs                   []uuid.UUID `json:"role_ids"`
	OnVacation                *bool       `json:"on_vacation,omitempty"`
	IsSick                    *bool       `json:"is_sick,omitempty"`
	AlternativeContactChannel *string     `json:"alt_contact_channel,omitempty"`
	AlternativeContactInfo    *string     `json:"alt_contact_info,omitempty"`
}

// AdminUpdateUserRequest — обновление пользователя администратором.
type AdminUpdateUserRequest struct {
	Username                  *string               `json:"username" binding:"omitempty,min=3,max=50"`
	Email                     *string               `json:"email" binding:"omitempty,email"`
	FullName                  *string               `json:"full_name,omitempty"`
	Position                  NullableField[string] `json:"position"`
	IsActive                  *bool                 `json:"is_active,omitempty"`
	RoleIDs                   *[]uuid.UUID          `json:"role_ids,omitempty"`
	OnVacation                *bool                 `json:"on_vacation,omitempty"`
	IsSick                    *bool                 `json:"is_sick,omitempty"`
	AlternativeContactChannel NullableField[string] `json:"alt_contact_channel"`
	AlternativeContactInfo    NullableField[string] `json:"alt_contact_info"`
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
	AvatarURL                 *string             `json:"avatar_url,omitempty"`
	Position                  *string             `json:"position,omitempty"`
	OnVacation                bool                `json:"on_vacation"`
	IsSick                    bool                `json:"is_sick"`
	AlternativeContactChannel *string             `json:"alt_contact_channel,omitempty"`
	AlternativeContactInfo    *string             `json:"alt_contact_info,omitempty"`
	IsActive                  bool                `json:"is_active"`
	Roles                     []AdminRoleResponse `json:"roles"`
	CreatedAt                 string              `json:"created_at"`
}
