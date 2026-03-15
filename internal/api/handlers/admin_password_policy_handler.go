package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type AdminPasswordPolicyHandler struct {
	policySvc *services.PasswordPolicyService
	auditLog  *services.AuditLogService
}

func NewAdminPasswordPolicyHandler(policySvc *services.PasswordPolicyService, auditLog *services.AuditLogService) *AdminPasswordPolicyHandler {
	return &AdminPasswordPolicyHandler{policySvc: policySvc, auditLog: auditLog}
}

// GetPasswordPolicy GET /admin/password-policy — текущая парольная политика.
func (h *AdminPasswordPolicyHandler) GetPasswordPolicy(c *gin.Context) {
	policy, err := h.policySvc.GetCurrentPolicy(c.Request.Context())
	if err != nil {
		if err == services.ErrNoPasswordPolicy {
			c.JSON(http.StatusNotFound, dto.APIResponse{
				Success: false,
				Data:    nil,
				Error:   &dto.APIError{Code: "NOT_FOUND", Message: "Политика паролей не настроена"},
			})
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить политику паролей")
		return
	}

	resp := dto.PasswordPolicyResponse{
		MinLength:        policy.MinLength,
		RequireDigits:    policy.RequireDigits,
		RequireLowercase: policy.RequireLowercase,
		RequireUppercase: policy.RequireUppercase,
		RequireSpecial:   policy.RequireSpecial,
		Notes:            policy.Notes,
		UpdatedAt:        policy.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if policy.UpdatedBy != nil {
		s := policy.UpdatedBy.String()
		resp.UpdatedBy = &s
	}
	writeSuccess(c, resp)
}

// UpdatePasswordPolicy PUT /admin/password-policy — обновление парольной политики.
func (h *AdminPasswordPolicyHandler) UpdatePasswordPolicy(c *gin.Context) {
	userIDStr := c.GetString("userID")
	if userIDStr == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется авторизация")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_USER", "Некорректный идентификатор пользователя")
		return
	}

	var req dto.UpdatePasswordPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	// Сначала получаем текущую политику, чтобы подставить значения для неуказанных полей
	current, err := h.policySvc.GetCurrentPolicy(c.Request.Context())
	if err != nil && err != services.ErrNoPasswordPolicy {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить текущую политику")
		return
	}

	minLength := 8
	requireDigits, requireLowercase, requireUppercase, requireSpecial := true, true, true, true
	if current != nil {
		minLength = current.MinLength
		requireDigits = current.RequireDigits
		requireLowercase = current.RequireLowercase
		requireUppercase = current.RequireUppercase
		requireSpecial = current.RequireSpecial
	}
	if req.MinLength != nil {
		minLength = *req.MinLength
	}
	if req.RequireDigits != nil {
		requireDigits = *req.RequireDigits
	}
	if req.RequireLowercase != nil {
		requireLowercase = *req.RequireLowercase
	}
	if req.RequireUppercase != nil {
		requireUppercase = *req.RequireUppercase
	}
	if req.RequireSpecial != nil {
		requireSpecial = *req.RequireSpecial
	}

	updated, err := h.policySvc.UpdatePolicy(c.Request.Context(), services.UpdatePasswordPolicyRequest{
		MinLength:        minLength,
		RequireDigits:    requireDigits,
		RequireLowercase: requireLowercase,
		RequireUppercase: requireUppercase,
		RequireSpecial:   requireSpecial,
		Notes:            req.Notes,
	}, userID)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "minLength должен быть от 1 до 100")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить политику паролей")
		return
	}
	if h.auditLog != nil {
		_ = h.auditLog.Log(c.Request.Context(), userID, "admin.password_policy.update", "password_policy", nil, nil)
	}
	resp := dto.PasswordPolicyResponse{
		MinLength:        updated.MinLength,
		RequireDigits:    updated.RequireDigits,
		RequireLowercase: updated.RequireLowercase,
		RequireUppercase: updated.RequireUppercase,
		RequireSpecial:   updated.RequireSpecial,
		Notes:            updated.Notes,
		UpdatedAt:        updated.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if updated.UpdatedBy != nil {
		s := updated.UpdatedBy.String()
		resp.UpdatedBy = &s
	}
	writeSuccess(c, resp)
}
