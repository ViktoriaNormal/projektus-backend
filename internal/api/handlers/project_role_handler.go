package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type ProjectRoleHandler struct {
	service *services.ProjectRoleService
}

func NewProjectRoleHandler(service *services.ProjectRoleService) *ProjectRoleHandler {
	return &ProjectRoleHandler{service: service}
}

func (h *ProjectRoleHandler) ListRoles(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}
	roles, err := h.service.ListRoles(c.Request.Context(), projectID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить список ролей")
		return
	}
	resp := make([]dto.ProjectRoleDefinitionResponse, 0, len(roles))
	for _, r := range roles {
		resp = append(resp, mapProjectRoleToDTO(r))
	}
	writeSuccess(c, resp)
}

func (h *ProjectRoleHandler) CreateRole(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}
	var req dto.CreateProjectRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	perms := make([]domain.ProjectRolePermission, len(req.Permissions))
	for i, p := range req.Permissions {
		perms[i] = domain.ProjectRolePermission{Area: p.Area, Access: p.Access}
	}
	role, err := h.service.CreateRole(c.Request.Context(), projectID, req.Name, req.Description, perms)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать роль")
		return
	}
	writeSuccess(c, mapProjectRoleToDTO(*role))
}

func (h *ProjectRoleHandler) UpdateRole(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}
	roleID, err := uuid.Parse(c.Param("roleId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор роли")
		return
	}
	var req dto.UpdateProjectRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	var perms []domain.ProjectRolePermission
	if req.Permissions != nil {
		perms = make([]domain.ProjectRolePermission, len(req.Permissions))
		for i, p := range req.Permissions {
			perms[i] = domain.ProjectRolePermission{Area: p.Area, Access: p.Access}
		}
	}
	role, err := h.service.UpdateRole(c.Request.Context(), projectID, roleID, req.Name, req.Description, perms)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Роль не найдена")
			return
		}
		if err == domain.ErrProjectAdminRole {
			writeError(c, http.StatusBadRequest, "PROJECT_ADMIN_ROLE", "Нельзя изменять права доступа роли администратора проекта")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить роль")
		return
	}
	writeSuccess(c, mapProjectRoleToDTO(*role))
}

func (h *ProjectRoleHandler) DeleteRole(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}
	roleID, err := uuid.Parse(c.Param("roleId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор роли")
		return
	}
	err = h.service.DeleteRole(c.Request.Context(), projectID, roleID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Роль не найдена")
			return
		}
		if err == domain.ErrProjectAdminRole {
			writeError(c, http.StatusBadRequest, "PROJECT_ADMIN_ROLE", "Нельзя удалить роль «Администратор проекта»")
			return
		}
if err == domain.ErrRoleHasMembers {
			writeError(c, http.StatusBadRequest, "ROLE_HAS_MEMBERS", "Нельзя удалить роль, к которой привязаны участники")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить роль")
		return
	}
	c.Status(http.StatusNoContent)
}

func mapProjectRoleToDTO(r domain.ProjectRole) dto.ProjectRoleDefinitionResponse {
	perms := make([]dto.ProjectRoleDefPermissionResponse, len(r.Permissions))
	for i, p := range r.Permissions {
		perms[i] = dto.ProjectRoleDefPermissionResponse{Area: p.Area, Access: p.Access}
	}
	return dto.ProjectRoleDefinitionResponse{
		ID:             uuid.MustParse(r.ID),
		Name:           r.Name,
		Description:    r.Description,
		IsAdmin:        r.IsAdmin,
		Permissions:    perms,
	}
}
