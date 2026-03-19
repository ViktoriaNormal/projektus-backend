package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type RoleHandler struct {
	roleService *services.RoleService
}

func NewRoleHandler(roleService *services.RoleService) *RoleHandler {
	return &RoleHandler{roleService: roleService}
}

func (h *RoleHandler) mapRoleToResponse(c *gin.Context, r domain.Role) dto.RoleResponse {
	perms, _ := h.roleService.GetRolePermissions(c.Request.Context(), r.ID)
	if perms == nil {
		perms = []string{}
	}
	return dto.RoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Permissions: perms,
	}
}

func (h *RoleHandler) ListSystemRoles(c *gin.Context) {
	roles, err := h.roleService.ListSystemRoles(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить список ролей")
		return
	}

	resp := make([]dto.RoleResponse, 0, len(roles))
	for _, r := range roles {
		resp = append(resp, h.mapRoleToResponse(c, r))
	}

	writeSuccess(c, resp)
}

func (h *RoleHandler) GetRole(c *gin.Context) {
	idStr := c.Param("roleId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный идентификатор роли")
		return
	}

	role, err := h.roleService.GetSystemRole(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Роль не найдена")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить роль")
		return
	}

	writeSuccess(c, h.mapRoleToResponse(c, *role))
}

func (h *RoleHandler) CreateSystemRole(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	role, err := h.roleService.CreateSystemRole(c.Request.Context(), req.Name, req.Description)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Имя роли обязательно")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать роль")
		return
	}

	writeSuccess(c, h.mapRoleToResponse(c, *role))
}

func (h *RoleHandler) UpdateSystemRole(c *gin.Context) {
	idStr := c.Param("roleId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный идентификатор роли")
		return
	}

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	role, err := h.roleService.UpdateSystemRole(c.Request.Context(), id, req.Name, req.Description)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Имя роли обязательно")
			return
		}
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Роль не найдена")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить роль")
		return
	}

	writeSuccess(c, h.mapRoleToResponse(c, *role))
}

func (h *RoleHandler) DeleteRole(c *gin.Context) {
	idStr := c.Param("roleId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный идентификатор роли")
		return
	}

	if err := h.roleService.DeleteSystemRole(c.Request.Context(), id); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить роль")
		return
	}

	writeSuccess(c, gin.H{"message": "Роль удалена"})
}

func (h *RoleHandler) AssignUserRoles(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный идентификатор пользователя")
		return
	}

	var req dto.AssignRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	if err := h.roleService.AssignSystemRolesToUser(c.Request.Context(), userID, req.RoleIDs); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось назначить роли пользователю")
		return
	}

	writeSuccess(c, gin.H{"message": "Роли пользователя обновлены"})
}

// GetUserRoles — admin endpoint (GET /admin/users/:userId/roles)
func (h *RoleHandler) GetUserRoles(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный идентификатор пользователя")
		return
	}

	roles, err := h.roleService.GetUserSystemRoles(c.Request.Context(), userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить роли пользователя")
		return
	}

	resp := make([]dto.RoleResponse, 0, len(roles))
	for _, r := range roles {
		resp = append(resp, h.mapRoleToResponse(c, r))
	}

	writeSuccess(c, resp)
}

// GetMySystemRoles — non-admin endpoint (GET /users/:id/roles), only own roles
func (h *RoleHandler) GetMySystemRoles(c *gin.Context) {
	currentUserID := c.GetString("userID")
	if currentUserID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}

	targetUserID := c.Param("id")
	if currentUserID != targetUserID {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "Можно запрашивать только свои роли")
		return
	}

	userID, err := uuid.Parse(targetUserID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный идентификатор пользователя")
		return
	}

	roles, err := h.roleService.GetUserSystemRoles(c.Request.Context(), userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить роли пользователя")
		return
	}

	resp := make([]dto.RoleResponse, 0, len(roles))
	for _, r := range roles {
		resp = append(resp, h.mapRoleToResponse(c, r))
	}

	writeSuccess(c, resp)
}
