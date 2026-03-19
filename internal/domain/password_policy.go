package domain

import (
	"time"

	"github.com/google/uuid"
)

// PasswordPolicy — настройки парольной политики.
type PasswordPolicy struct {
	ID               uuid.UUID  `json:"id"`
	MinLength        int        `json:"min_length"`
	RequireDigits    bool       `json:"require_digits"`
	RequireLowercase bool       `json:"require_lowercase"`
	RequireUppercase bool       `json:"require_uppercase"`
	RequireSpecial   bool       `json:"require_special"`
	Notes            *string    `json:"notes,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at"`
	UpdatedBy        *uuid.UUID `json:"updated_by,omitempty"`
}
