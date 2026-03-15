package domain

import (
	"time"

	"github.com/google/uuid"
)

// PasswordPolicy — настройки парольной политики.
type PasswordPolicy struct {
	ID               uuid.UUID
	MinLength        int
	RequireDigits    bool
	RequireLowercase bool
	RequireUppercase bool
	RequireSpecial   bool
	Notes            *string
	UpdatedAt        time.Time
	UpdatedBy        *uuid.UUID
}
