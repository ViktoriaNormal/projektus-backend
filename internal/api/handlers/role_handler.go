package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

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
	permResp := make([]dto.RolePermissionResponse, 0, len(perms))
	for _, p := range perms {
		permResp = append(permResp, dto.RolePermissionResponse{Code: p.Code, Access: p.Access})
	}
	return dto.RoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		IsAdmin:     r.IsAdmin,
		Permissions: permResp,
	}
}

func (h *RoleHandler) ListPermissions(c *gin.Context) {
	perms, err := h.roleService.ListPermissions(c.Request.Context())
	if err != nil {
		respondInternal(c, err, "Не удалось получить список прав доступа")
		return
	}

	resp := make([]dto.PermissionResponse, 0, len(perms))
	for _, p := range perms {
		resp = append(resp, dto.PermissionResponse{
			Code:        p.Code,
			Scope:       p.Scope,
			Name:        p.Name,
			Description: p.Description,
		})
	}

	writeSuccess(c, resp)
}

func (h *RoleHandler) ListSystemRoles(c *gin.Context) {
	roles, err := h.roleService.ListSystemRoles(c.Request.Context())
	if err != nil {
		respondInternal(c, err, "Не удалось получить список ролей")
		return
	}

	resp := make([]dto.RoleResponse, 0, len(roles))
	for _, r := range roles {
		resp = append(resp, h.mapRoleToResponse(c, r))
	}

	writeSuccess(c, resp)
}

func (h *RoleHandler) GetRole(c *gin.Context) {
	id, ok := paramUUID(c, "roleId")
	if !ok {
		return
	}

	role, err := h.roleService.GetSystemRole(c.Request.Context(), id)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить роль")
		return
	}

	writeSuccess(c, h.mapRoleToResponse(c, *role))
}

func (h *RoleHandler) CreateSystemRole(c *gin.Context) {
	req, ok := bindJSON[dto.CreateRoleRequest](c)
	if !ok {
		return
	}

	perms := make([]domain.Permission, len(req.Permissions))
	for i, p := range req.Permissions {
		perms[i] = domain.Permission{Code: p.Code, Access: p.Access}
	}
	role, err := h.roleService.CreateSystemRole(c.Request.Context(), req.Name, req.Description, perms)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать роль")
		return
	}

	writeSuccess(c, h.mapRoleToResponse(c, *role))
}

func (h *RoleHandler) UpdateSystemRole(c *gin.Context) {
	id, ok := paramUUID(c, "roleId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.UpdateRoleRequest](c)
	if !ok {
		return
	}

	perms := make([]domain.Permission, len(req.Permissions))
	for i, p := range req.Permissions {
		perms[i] = domain.Permission{Code: p.Code, Access: p.Access}
	}
	role, err := h.roleService.UpdateSystemRole(c.Request.Context(), id, req.Name, req.Description, perms)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить роль")
		return
	}

	writeSuccess(c, h.mapRoleToResponse(c, *role))
}

func (h *RoleHandler) DeleteRole(c *gin.Context) {
	id, ok := paramUUID(c, "roleId")
	if !ok {
		return
	}

	if err := h.roleService.DeleteSystemRole(c.Request.Context(), id); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить роль")
		return
	}

	writeSuccess(c, gin.H{"message": "Роль удалена"})
}

func (h *RoleHandler) AssignUserRoles(c *gin.Context) {
	userID, ok := paramUUID(c, "userId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.AssignRolesRequest](c)
	if !ok {
		return
	}

	if err := h.roleService.AssignSystemRolesToUser(c.Request.Context(), userID, req.RoleIDs); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось назначить роли пользователю")
		return
	}

	writeSuccess(c, gin.H{"message": "Роли пользователя обновлены"})
}

// GetUserRoles — admin endpoint (GET /admin/users/:userId/roles)
func (h *RoleHandler) GetUserRoles(c *gin.Context) {
	userID, ok := paramUUID(c, "userId")
	if !ok {
		return
	}

	roles, err := h.roleService.GetUserSystemRoles(c.Request.Context(), userID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить роли пользователя")
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
	currentUserID, ok := requireUserUUID(c)
	if !ok {
		return
	}

	userID, ok := paramUUID(c, "id")
	if !ok {
		return
	}

	if currentUserID != userID {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "Можно запрашивать только свои роли")
		return
	}

	roles, err := h.roleService.GetUserSystemRoles(c.Request.Context(), userID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить роли пользователя")
		return
	}

	resp := make([]dto.RoleResponse, 0, len(roles))
	for _, r := range roles {
		resp = append(resp, h.mapRoleToResponse(c, r))
	}

	writeSuccess(c, resp)
}
