package dto

// PasswordPolicyResponse — текущая парольная политика (GET /admin/password-policy).
type PasswordPolicyResponse struct {
	MinLength        int     `json:"min_length"`
	RequireDigits    bool    `json:"require_digits"`
	RequireLowercase bool    `json:"require_lowercase"`
	RequireUppercase bool    `json:"require_uppercase"`
	RequireSpecial   bool    `json:"require_special"`
	Notes            *string `json:"notes,omitempty"`
	UpdatedAt        string  `json:"updated_at"`
	UpdatedBy        *string `json:"updated_by,omitempty"`
}

// UpdatePasswordPolicyRequest — обновление политики (PUT /admin/password-policy).
type UpdatePasswordPolicyRequest struct {
	MinLength        *int    `json:"min_length" binding:"omitempty,min=1,max=100"`
	RequireDigits    *bool   `json:"require_digits"`
	RequireLowercase *bool   `json:"require_lowercase"`
	RequireUppercase *bool   `json:"require_uppercase"`
	RequireSpecial   *bool   `json:"require_special"`
	Notes            *string `json:"notes"`
}
