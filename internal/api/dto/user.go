package dto

type UserResponse struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	FullName  string  `json:"full_name"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Position  *string `json:"position,omitempty"`
}

type UpdateUserProfileRequest struct {
	FullName string  `json:"full_name" binding:"required"`
	Email    string  `json:"email" binding:"required,email"`
	Position *string `json:"position"`
}
