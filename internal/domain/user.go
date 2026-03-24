package domain

import "time"

type User struct {
	ID                      string    `json:"id"`
	Username                string    `json:"username"`
	Email                   string    `json:"email"`
	PasswordHash            string    `json:"-"`
	FullName                string    `json:"full_name"`
	AvatarURL               *string   `json:"avatar_url,omitempty"`
	Position                *string   `json:"position,omitempty"`
	OnVacation              bool      `json:"on_vacation"`
	IsSick                  bool      `json:"is_sick"`
	AlternativeContactChannel *string `json:"alternative_contact_channel"`
	AlternativeContactInfo    *string `json:"alternative_contact_info"`
	IsActive                bool      `json:"-"`
	CreatedAt               time.Time `json:"-"`
	UpdatedAt               time.Time `json:"-"`
}
