package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type AuthHandler struct {
	auth      services.AuthService
	auditLog  *services.AuditLogService
}

func NewAuthHandler(auth services.AuthService, auditLog *services.AuditLogService) *AuthHandler {
	return &AuthHandler{auth: auth, auditLog: auditLog}
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, dto.APIResponse{
		Success: false,
		Data:    nil,
		Error: &dto.APIError{
			Code:    code,
			Message: message,
		},
	})
}

func writeSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    data,
		Error:   nil,
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	user, err := h.auth.Register(c.Request.Context(), req.Username, req.Email, req.Password, req.FullName)
	if err != nil {
		if errors.Is(err, domain.ErrPasswordPolicy) {
			writeError(c, http.StatusBadRequest, "PASSWORD_POLICY_VIOLATION", "Пароль не соответствует политике безопасности")
			return
		}
		log.Printf("[Register] error: %v", err)
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		return
	}

	writeSuccess(c, gin.H{
		"user": mapUserToResponse(user),
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	ip := c.ClientIP()

	access, refresh, user, err := h.auth.Login(c.Request.Context(), req.Username, req.Password, ip)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Неверный логин или пароль")
		case errors.Is(err, domain.ErrUserBlocked):
			writeError(c, http.StatusTooManyRequests, "USER_BLOCKED", "Пользователь временно заблокирован из-за неудачных попыток входа")
		case errors.Is(err, domain.ErrIPBlocked):
			writeError(c, http.StatusTooManyRequests, "IP_BLOCKED", "IP-адрес временно заблокирован из-за неудачных попыток входа")
		default:
			log.Printf("[Login] error: %v", err)
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		}
		return
	}

	if h.auditLog != nil {
		if uid, err := uuid.Parse(user.ID); err == nil {
			_ = h.auditLog.Log(c.Request.Context(), uid, "auth.login", "user", &uid, map[string]string{"ip": ip})
		}
	}
	// Пока возвращаем только токены и пользователя, без явного списка ролей.
	writeSuccess(c, dto.AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         mapUserToResponse(user),
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	access, refresh, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidToken), errors.Is(err, domain.ErrRefreshTokenRevoked):
			writeError(c, http.StatusUnauthorized, "INVALID_TOKEN", "Недействительный или отозванный refresh токен")
		default:
			log.Printf("[Refresh] error: %v", err)
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		}
		return
	}

	writeSuccess(c, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	if err := h.auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		return
	}

	writeSuccess(c, gin.H{
		"message": "Выход выполнен",
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	if err := h.auth.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Неверный текущий пароль")
		case errors.Is(err, domain.ErrPasswordPolicy):
			writeError(c, http.StatusBadRequest, "PASSWORD_POLICY_VIOLATION", "Пароль не соответствует политике безопасности")
		case errors.Is(err, domain.ErrPasswordReuse):
			writeError(c, http.StatusBadRequest, "PASSWORD_REUSE", "Нельзя использовать один из последних паролей")
		default:
			log.Printf("[ChangePassword] error: %v", err)
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		}
		return
	}
	if h.auditLog != nil {
		if uid, err := uuid.Parse(userID); err == nil {
			_ = h.auditLog.Log(c.Request.Context(), uid, "auth.password.change", "user", &uid, nil)
		}
	}
	writeSuccess(c, gin.H{
		"message": "Пароль успешно изменен",
	})
}

