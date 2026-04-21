package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type ProjectRoleHandler struct {
	service       *services.ProjectRoleService
	permissionSvc *services.PermissionService
}

func NewProjectRoleHandler(service *services.ProjectRoleService, permissionSvc *services.PermissionService) *ProjectRoleHandler {
	return &ProjectRoleHandler{service: service, permissionSvc: permissionSvc}
}

func (h *ProjectRoleHandler) ListRoles(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}
	roles, err := h.service.ListRoles(c.Request.Context(), projectID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить список ролей")
		return
	}
	resp := make([]dto.ProjectRoleDefinitionResponse, 0, len(roles))
	for _, r := range roles {
		resp = append(resp, mapProjectRoleToDTO(r))
	}
	writeSuccess(c, resp)
}

func (h *ProjectRoleHandler) CreateRole(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.CreateProjectRoleRequest](c)
	if !ok {
		return
	}
	perms := make([]domain.ProjectRolePermission, len(req.Permissions))
	for i, p := range req.Permissions {
		perms[i] = domain.ProjectRolePermission{Area: p.Area, Access: p.Access}
	}
	role, err := h.service.CreateRole(c.Request.Context(), projectID, req.Name, req.Description, perms)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать роль")
		return
	}
	writeSuccess(c, mapProjectRoleToDTO(*role))
}

func (h *ProjectRoleHandler) UpdateRole(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}
	roleID, ok := paramUUID(c, "roleId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.UpdateProjectRoleRequest](c)
	if !ok {
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
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить роль")
		return
	}
	writeSuccess(c, mapProjectRoleToDTO(*role))
}

func (h *ProjectRoleHandler) DeleteRole(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}
	roleID, ok := paramUUID(c, "roleId")
	if !ok {
		return
	}
	err := h.service.DeleteRole(c.Request.Context(), projectID, roleID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить роль")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ProjectRoleHandler) GetMyPermissions(c *gin.Context) {
	userID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	perms, err := h.permissionSvc.GetMyPermissions(c.Request.Context(), userID, projectID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить права доступа")
		return
	}

	resp := make([]dto.ProjectRoleDefPermissionResponse, len(perms))
	for i, p := range perms {
		resp[i] = dto.ProjectRoleDefPermissionResponse{Area: p.Area, Access: p.Access}
	}
	writeSuccess(c, resp)
}

func mapProjectRoleToDTO(r domain.ProjectRole) dto.ProjectRoleDefinitionResponse {
	perms := make([]dto.ProjectRoleDefPermissionResponse, len(r.Permissions))
	for i, p := range r.Permissions {
		perms[i] = dto.ProjectRoleDefPermissionResponse{Area: p.Area, Access: p.Access}
	}
	return dto.ProjectRoleDefinitionResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		IsAdmin:     r.IsAdmin,
		Order:       r.Order,
		Permissions: perms,
	}
}
