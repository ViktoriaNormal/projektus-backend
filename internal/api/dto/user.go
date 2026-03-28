package dto

type UserResponse struct {
	ID                        string  `json:"id"`
	Username                  string  `json:"username"`
	Email                     string  `json:"email"`
	FullName                  string  `json:"full_name"`
	AvatarURL                 *string `json:"avatar_url,omitempty"`
	Position                  *string `json:"position,omitempty"`
	OnVacation                bool    `json:"on_vacation"`
	IsSick                    bool    `json:"is_sick"`
	AlternativeContactChannel *string `json:"alt_contact_channel,omitempty"`
	AlternativeContactInfo    *string `json:"alt_contact_info,omitempty"`
}

type UpdateUserProfileRequest struct {
	FullName                  string  `json:"full_name" binding:"required"`
	Email                     string  `json:"email" binding:"required,email"`
	Position                  *string `json:"position"`
	OnVacation                *bool   `json:"on_vacation"`
	IsSick                    *bool   `json:"is_sick"`
	AlternativeContactChannel *string `json:"alt_contact_channel"`
	AlternativeContactInfo    *string `json:"alt_contact_info"`
}
