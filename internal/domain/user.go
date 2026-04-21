package domain

import "github.com/google/uuid"

type User struct {
	ID                        uuid.UUID `json:"id"`
	Username                  string    `json:"username"`
	Email                     string    `json:"email"`
	PasswordHash              string    `json:"-"`
	FullName                  string    `json:"full_name"`
	AvatarURL                 *string   `json:"avatar_url,omitempty"`
	Position                  *string   `json:"position,omitempty"`
	OnVacation                bool      `json:"on_vacation"`
	IsSick                    bool      `json:"is_sick"`
	AlternativeContactChannel *string   `json:"alternative_contact_channel"`
	AlternativeContactInfo    *string   `json:"alternative_contact_info"`
	IsActive                  bool      `json:"-"`
}
