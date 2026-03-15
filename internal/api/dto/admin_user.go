package dto

import "github.com/google/uuid"

// AdminCreateUserRequest — создание пользователя администратором (Swagger: AdminCreateUserRequest).
type AdminCreateUserRequest struct {
	Username        string      `json:"username" binding:"required,min=3,max=50"`
	Email           string      `json:"email" binding:"required,email"`
	FullName        string      `json:"fullName" binding:"required"`
	InitialPassword string      `json:"initialPassword" binding:"required,min=6"`
	SystemRoles     []uuid.UUID `json:"systemRoles"` // список ID системных ролей
}

// AdminUserResponse — пользователь в списке админки (без пароля).
type AdminUserResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FullName  string `json:"fullName"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
}
