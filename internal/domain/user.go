package domain

import "time"

type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	FullName     string     `json:"full_name"`
	AvatarURL    *string    `json:"avatar_url,omitempty"`
	Position     *string    `json:"position,omitempty"`
	IsActive     bool       `json:"-"`
	CreatedAt    time.Time  `json:"-"`
	UpdatedAt    time.Time  `json:"-"`
}
