package domain

import "time"

type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	FullName     string
	AvatarURL    *string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

