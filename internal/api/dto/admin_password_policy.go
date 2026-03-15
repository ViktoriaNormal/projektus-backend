package dto

// PasswordPolicyResponse — текущая парольная политика (GET /admin/password-policy).
type PasswordPolicyResponse struct {
	MinLength        int     `json:"minLength"`
	RequireDigits    bool    `json:"requireDigits"`
	RequireLowercase bool    `json:"requireLowercase"`
	RequireUppercase bool    `json:"requireUppercase"`
	RequireSpecial   bool    `json:"requireSpecial"`
	Notes            *string `json:"notes,omitempty"`
	UpdatedAt        string  `json:"updatedAt"`
	UpdatedBy        *string `json:"updatedBy,omitempty"`
}

// UpdatePasswordPolicyRequest — обновление политики (PUT /admin/password-policy).
type UpdatePasswordPolicyRequest struct {
	MinLength        *int    `json:"minLength" binding:"omitempty,min=1,max=100"`
	RequireDigits    *bool   `json:"requireDigits"`
	RequireLowercase *bool   `json:"requireLowercase"`
	RequireUppercase *bool   `json:"requireUppercase"`
	RequireSpecial   *bool   `json:"requireSpecial"`
	Notes            *string `json:"notes"`
}
