package handlers

import (
	"errors"
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
}

func NewAdminUserHandler(adminUserSvc *services.AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{adminUserSvc: adminUserSvc}
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
		resp = append(resp, mapAdminUserToResponse(u))
	}

	writeSuccess(c, gin.H{
		"users": resp,
		"total": total,
	})
}

// GetUser GET /admin/users/:id — получение пользователя по ID.
func (h *AdminUserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор пользователя")
		return
	}

	user, err := h.adminUserSvc.GetUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить пользователя")
		return
	}

	writeSuccess(c, mapAdminUserToResponse(*user))
}

// CreateUser POST /admin/users — создание пользователя с начальным паролем.
func (h *AdminUserHandler) CreateUser(c *gin.Context) {
	var req dto.AdminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	user, err := h.adminUserSvc.CreateUser(c.Request.Context(), services.AdminCreateUserRequest{
		Username:                  req.Username,
		Email:                     req.Email,
		FullName:                  req.FullName,
		Position:                  req.Position,
		Password:                  req.Password,
		IsActive:                  req.IsActive,
		SystemRoleIDs:             req.RoleIDs,
		OnVacation:                req.OnVacation,
		IsSick:                    req.IsSick,
		AlternativeContactChannel: req.AlternativeContactChannel,
		AlternativeContactInfo:    req.AlternativeContactInfo,
	})
	if err != nil {
		if errors.Is(err, domain.ErrPasswordPolicy) {
			writeError(c, http.StatusBadRequest, "PASSWORD_POLICY_VIOLATION", "Пароль не соответствует политике безопасности")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать пользователя")
		return
	}
	c.JSON(http.StatusCreated, dto.APIResponse{
		Success: true,
		Data:    mapAdminUserToResponse(*user),
		Error:   nil,
	})
}

// UpdateUser PUT /admin/users/:id — обновление данных пользователя.
func (h *AdminUserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор пользователя")
		return
	}

	var req dto.AdminUpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	user, err := h.adminUserSvc.UpdateUser(c.Request.Context(), userID, services.AdminUpdateUserRequest{
		Username:                  req.Username,
		Email:                     req.Email,
		FullName:                  req.FullName,
		Position:                  req.Position,
		IsActive:                  req.IsActive,
		RoleIDs:                   req.RoleIDs,
		OnVacation:                req.OnVacation,
		IsSick:                    req.IsSick,
		AlternativeContactChannel: req.AlternativeContactChannel,
		AlternativeContactInfo:    req.AlternativeContactInfo,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить пользователя")
		return
	}

	writeSuccess(c, mapAdminUserToResponse(*user))
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
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Нельзя удалить свой аккаунт")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить пользователя")
		return
	}
	c.Status(http.StatusNoContent)
}

func mapAdminUserToResponse(u services.AdminUserWithRoles) dto.AdminUserResponse {
	roles := make([]dto.AdminRoleResponse, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, dto.AdminRoleResponse{
			ID:   r.ID.String(),
			Name: r.Name,
		})
	}
	return dto.AdminUserResponse{
		ID:                        u.User.ID,
		Username:                  u.User.Username,
		Email:                     u.User.Email,
		FullName:                  u.User.FullName,
		AvatarURL:                 u.User.AvatarURL,
		Position:                  u.User.Position,
		OnVacation:                u.User.OnVacation,
		IsSick:                    u.User.IsSick,
		AlternativeContactChannel: u.User.AlternativeContactChannel,
		AlternativeContactInfo:    u.User.AlternativeContactInfo,
		IsActive:                  u.User.IsActive,
		Roles:                     roles,
		CreatedAt:                 "",
	}
}
