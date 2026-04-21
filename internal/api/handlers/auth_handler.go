package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"projektus-backend/config"
	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type AuthHandler struct {
	cfg     *config.Config
	auth    services.AuthService
	roleSvc *services.RoleService
}

func NewAuthHandler(cfg *config.Config, auth services.AuthService, roleSvc *services.RoleService) *AuthHandler {
	return &AuthHandler{cfg: cfg, auth: auth, roleSvc: roleSvc}
}

func (h *AuthHandler) Register(c *gin.Context) {
	if !h.cfg.AllowPublicRegistration {
		writeError(c, http.StatusForbidden, "REGISTRATION_DISABLED",
			"Самостоятельная регистрация в системе отключена. Обратитесь к администратору для получения учётной записи.")
		return
	}

	req, ok := bindJSON[dto.RegisterRequest](c)
	if !ok {
		return
	}

	user, err := h.auth.Register(c.Request.Context(), req.Username, req.Email, req.Password, req.FullName)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		// Auth-специфичные ошибки — отдельно от таблицы.
		if errors.Is(err, domain.ErrPasswordPolicy) {
			writeError(c, http.StatusBadRequest, "PASSWORD_POLICY_VIOLATION", "Пароль не соответствует политике безопасности")
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
		return
	}

	writeSuccess(c, gin.H{
		"user": mapUserToResponse(user),
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	req, ok := bindJSON[dto.LoginRequest](c)
	if !ok {
		return
	}
	ip := c.ClientIP()

	access, refresh, user, err := h.auth.Login(c.Request.Context(), req.Username, req.Password, ip)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		// Auth-специфичные ошибки с особыми HTTP-кодами (429, 401).
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Неверный логин или пароль")
		case errors.Is(err, domain.ErrUserBlocked):
			writeError(c, http.StatusTooManyRequests, "USER_BLOCKED", "Пользователь временно заблокирован из-за неудачных попыток входа")
		case errors.Is(err, domain.ErrIPBlocked):
			writeError(c, http.StatusTooManyRequests, "IP_BLOCKED", "IP-адрес временно заблокирован из-за неудачных попыток входа")
		default:
			respondInternal(c, err, "Внутренняя ошибка сервера")
		}
		return
	}

	uid := user.ID

	// Подтягиваем системные роли с permissions
	roleResponses := make([]dto.RoleResponse, 0)
	if roles, err := h.roleSvc.GetUserSystemRoles(c.Request.Context(), uid); err == nil {
		for _, r := range roles {
			perms, _ := h.roleSvc.GetRolePermissions(c.Request.Context(), r.ID)
			permResp := make([]dto.RolePermissionResponse, 0, len(perms))
			for _, p := range perms {
				permResp = append(permResp, dto.RolePermissionResponse{Code: p.Code, Access: p.Access})
			}
			roleResponses = append(roleResponses, dto.RoleResponse{
				ID:          r.ID,
				Name:        r.Name,
				Description: r.Description,
				IsAdmin:     r.IsAdmin,
				Permissions: permResp,
			})
		}
	}

	writeSuccess(c, dto.AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         mapUserToResponse(user),
		Roles:        roleResponses,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	req, ok := bindJSON[dto.RefreshRequest](c)
	if !ok {
		return
	}

	access, refresh, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		// Auth-специфичные: токен недействителен/отозван.
		switch {
		case errors.Is(err, domain.ErrInvalidToken), errors.Is(err, domain.ErrRefreshTokenRevoked):
			writeError(c, http.StatusUnauthorized, "INVALID_TOKEN", "Недействительный или отозванный refresh токен")
		default:
			respondInternal(c, err, "Внутренняя ошибка сервера")
		}
		return
	}

	writeSuccess(c, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	req, ok := bindJSON[dto.LogoutRequest](c)
	if !ok {
		return
	}

	if err := h.auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
		return
	}

	writeSuccess(c, gin.H{
		"message": "Выход выполнен",
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	userID := userUUID.String()

	req, ok := bindJSON[dto.ChangePasswordRequest](c)
	if !ok {
		return
	}

	if err := h.auth.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		// Auth-специфичные ошибки смены пароля.
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Неверный текущий пароль")
		case errors.Is(err, domain.ErrPasswordPolicy):
			writeError(c, http.StatusBadRequest, "PASSWORD_POLICY_VIOLATION", "Пароль не соответствует политике безопасности")
		case errors.Is(err, domain.ErrPasswordReuse):
			writeError(c, http.StatusBadRequest, "PASSWORD_REUSE", "Нельзя использовать один из последних паролей")
		default:
			respondInternal(c, err, "Внутренняя ошибка сервера")
		}
		return
	}
	writeSuccess(c, gin.H{
		"message": "Пароль успешно изменен",
	})
}
