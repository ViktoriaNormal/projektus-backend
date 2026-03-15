package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type AdminUserHandler struct {
	adminUserSvc *services.AdminUserService
	auditLog     *services.AuditLogService
}

func NewAdminUserHandler(adminUserSvc *services.AdminUserService, auditLog *services.AuditLogService) *AdminUserHandler {
	return &AdminUserHandler{adminUserSvc: adminUserSvc, auditLog: auditLog}
}

// ListUsers GET /admin/users — список всех пользователей с пагинацией.
func (h *AdminUserHandler) ListUsers(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	includeDeleted := c.Query("includeDeleted") == "true"

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	users, total, err := h.adminUserSvc.ListUsers(c.Request.Context(), int32(limit), int32(offset), includeDeleted)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить список пользователей")
		return
	}

	resp := make([]dto.AdminUserResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, dto.AdminUserResponse{
			ID:        u.ID,
			Username:  u.Username,
			Email:     u.Email,
			FullName:  u.FullName,
			IsActive:  u.IsActive,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	writeSuccess(c, gin.H{
		"users": resp,
		"total": total,
	})
}

// CreateUser POST /admin/users — создание пользователя с начальным паролем.
func (h *AdminUserHandler) CreateUser(c *gin.Context) {
	var req dto.AdminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	user, err := h.adminUserSvc.CreateUser(c.Request.Context(), services.AdminCreateUserRequest{
		Username:        req.Username,
		Email:           req.Email,
		FullName:        req.FullName,
		InitialPassword: req.InitialPassword,
		SystemRoleIDs:   req.SystemRoles,
	})
	if err != nil {
		if err == domain.ErrPasswordPolicy {
			writeError(c, http.StatusBadRequest, "PASSWORD_POLICY_VIOLATION", "Пароль не соответствует политике безопасности")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать пользователя")
		return
	}
	if h.auditLog != nil {
		if adminIDStr := c.GetString("userID"); adminIDStr != "" {
			if adminID, err := uuid.Parse(adminIDStr); err == nil {
				if createdID, err := uuid.Parse(user.ID); err == nil {
					_ = h.auditLog.Log(c.Request.Context(), adminID, "admin.user.create", "user", &createdID, nil)
				}
			}
		}
	}
	c.JSON(http.StatusCreated, dto.APIResponse{
		Success: true,
		Data: dto.AdminUserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			FullName:  user.FullName,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
		Error: nil,
	})
}

// DeleteUser DELETE /admin/users/:id — мягкое удаление пользователя.
func (h *AdminUserHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	targetID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор пользователя")
		return
	}

	currentIDStr := c.GetString("userID")
	if currentIDStr == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется авторизация")
		return
	}
	currentID, err := uuid.Parse(currentIDStr)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Некорректный контекст пользователя")
		return
	}

	if err := h.adminUserSvc.DeleteUser(c.Request.Context(), targetID, currentID); err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Нельзя удалить свой аккаунт")
			return
		}
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить пользователя")
		return
	}
	if h.auditLog != nil {
		_ = h.auditLog.Log(c.Request.Context(), currentID, "admin.user.delete", "user", &targetID, nil)
	}
	c.Status(http.StatusNoContent)
}
