package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
	"projektus-backend/internal/services"
)

type AdminUserHandler struct {
	adminUserSvc *services.AdminUserService
}

func NewAdminUserHandler(adminUserSvc *services.AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{adminUserSvc: adminUserSvc}
}

// ListUsers GET /admin/users — список всех пользователей с пагинацией,
// серверными фильтрами и карточками статистики.
//
// Query-параметры:
//   - limit, offset — пагинация. limit=0 → только счётчики без users[].
//   - includeDeleted=true — показать soft-deleted.
//   - q — ILIKE по username/email/full_name/position.
//   - is_active=true|false — только активные / только заблокированные.
//   - role_id=<uuid> — с назначенной системной ролью.
//
// В data возвращаются:
//   - users[] — страница под limit/offset+фильтры;
//   - total — число подходящих под фильтры записей (без учёта limit/offset);
//   - active_count / inactive_count — глобальные счётчики по всему множеству
//     (не зависят от q/is_active/role_id — для карточек статистики на UI).
func (h *AdminUserHandler) ListUsers(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	// limit=0 (явно) валиден — отдаём только статистику. Отрицательные/невалидные → 20.
	limit := 20
	if n, err := strconv.Atoi(limitStr); err == nil {
		switch {
		case n == 0:
			limit = 0
		case n < 0:
			limit = 20
		default:
			limit = n
		}
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	filter := repositories.AdminUserListFilter{
		IncludeDeleted: c.Query("includeDeleted") == "true",
	}
	if q := c.Query("q"); q != "" {
		filter.Query = &q
	}
	if v := c.Query("is_active"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			filter.IsActive = &parsed
		}
	}
	if v := c.Query("role_id"); v != "" {
		if parsed, err := uuid.Parse(v); err == nil {
			filter.RoleID = &parsed
		}
	}

	result, err := h.adminUserSvc.ListUsers(c.Request.Context(), int32(limit), int32(offset), filter)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить список пользователей")
		return
	}

	resp := make([]dto.AdminUserResponse, 0, len(result.Users))
	for _, u := range result.Users {
		resp = append(resp, mapAdminUserToResponse(u))
	}

	writeSuccess(c, gin.H{
		"users":          resp,
		"total":          result.Total,
		"active_count":   result.ActiveCount,
		"inactive_count": result.InactiveCount,
	})
}

// GetUser GET /admin/users/:id — получение пользователя по ID.
func (h *AdminUserHandler) GetUser(c *gin.Context) {
	userID, ok := paramUUID(c, "id")
	if !ok {
		return
	}

	user, err := h.adminUserSvc.GetUser(c.Request.Context(), userID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить пользователя")
		return
	}

	writeSuccess(c, mapAdminUserToResponse(*user))
}

// CreateUser POST /admin/users — создание пользователя с начальным паролем.
func (h *AdminUserHandler) CreateUser(c *gin.Context) {
	req, ok := bindJSON[dto.AdminCreateUserRequest](c)
	if !ok {
		return
	}

	position := req.Position
	user, err := h.adminUserSvc.CreateUser(c.Request.Context(), services.AdminCreateUserRequest{
		Username:                  req.Username,
		Email:                     req.Email,
		FullName:                  req.FullName,
		Position:                  &position,
		Password:                  req.Password,
		IsActive:                  req.IsActive,
		SystemRoleIDs:             req.RoleIDs,
		OnVacation:                req.OnVacation,
		IsSick:                    req.IsSick,
		AlternativeContactChannel: req.AlternativeContactChannel,
		AlternativeContactInfo:    req.AlternativeContactInfo,
	})
	if err != nil {
		// ErrPasswordPolicy — auth-специфичная ошибка, нет в таблице.
		if errors.Is(err, domain.ErrPasswordPolicy) {
			writeError(c, http.StatusBadRequest, "PASSWORD_POLICY_VIOLATION", "Пароль не соответствует политике безопасности")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать пользователя")
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
	userID, ok := paramUUID(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.AdminUpdateUserRequest](c)
	if !ok {
		return
	}

	if req.Position.Set && (req.Position.Null || req.Position.Value == "") {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR",
			"Поле «Должность» не может быть пустым")
		return
	}
	if req.RoleIDs != nil && len(*req.RoleIDs) == 0 {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR",
			"У пользователя должна быть назначена хотя бы одна системная роль")
		return
	}

	// Convert NullableField to *string for the service layer.
	// Empty string signals "set to NULL" (service converts "" to sql.NullString{Valid: false}).
	var position *string
	if req.Position.Set {
		if req.Position.Null {
			empty := ""
			position = &empty
		} else {
			position = &req.Position.Value
		}
	}
	var altContactChannel *string
	if req.AlternativeContactChannel.Set {
		if req.AlternativeContactChannel.Null {
			empty := ""
			altContactChannel = &empty
		} else {
			altContactChannel = &req.AlternativeContactChannel.Value
		}
	}
	var altContactInfo *string
	if req.AlternativeContactInfo.Set {
		if req.AlternativeContactInfo.Null {
			empty := ""
			altContactInfo = &empty
		} else {
			altContactInfo = &req.AlternativeContactInfo.Value
		}
	}

	user, err := h.adminUserSvc.UpdateUser(c.Request.Context(), userID, services.AdminUpdateUserRequest{
		Username:                  req.Username,
		Email:                     req.Email,
		FullName:                  req.FullName,
		Position:                  position,
		IsActive:                  req.IsActive,
		RoleIDs:                   req.RoleIDs,
		OnVacation:                req.OnVacation,
		IsSick:                    req.IsSick,
		AlternativeContactChannel: altContactChannel,
		AlternativeContactInfo:    altContactInfo,
	})
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить пользователя")
		return
	}

	writeSuccess(c, mapAdminUserToResponse(*user))
}

// DeleteUser DELETE /admin/users/:id — мягкое удаление пользователя.
func (h *AdminUserHandler) DeleteUser(c *gin.Context) {
	targetID, ok := paramUUID(c, "id")
	if !ok {
		return
	}

	currentID, ok := requireUserUUID(c)
	if !ok {
		return
	}

	if err := h.adminUserSvc.DeleteUser(c.Request.Context(), targetID, currentID); err != nil {
		// Сохраняем специфичный маппинг ErrInvalidInput → «Нельзя удалить свой аккаунт».
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Нельзя удалить свой аккаунт")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить пользователя")
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
		ID:                        u.User.ID.String(),
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
