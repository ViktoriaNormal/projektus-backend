package handlers

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
	"projektus-backend/internal/services"
)

type UserHandler struct {
	users    services.UserService
	members  repositories.ProjectMemberRepository
	roleRepo repositories.RoleRepository
}

func NewUserHandler(users services.UserService, members repositories.ProjectMemberRepository, roleRepo repositories.RoleRepository) *UserHandler {
	return &UserHandler{users: users, members: members, roleRepo: roleRepo}
}

func mapUserToResponse(u *domain.User) dto.UserResponse {
	return dto.UserResponse{
		ID:                        u.ID.String(),
		Username:                  u.Username,
		Email:                     u.Email,
		FullName:                  u.FullName,
		AvatarURL:                 u.AvatarURL,
		Position:                  u.Position,
		OnVacation:                u.OnVacation,
		IsSick:                    u.IsSick,
		AlternativeContactChannel: u.AlternativeContactChannel,
		AlternativeContactInfo:    u.AlternativeContactInfo,
	}
}

// GET /api/v1/users?q=&limit=&offset=
//
// Формат ответа:
//
//	{ "users": [...], "total": <int> }
//
// `total` — полное число записей под фильтр `q` (без учёта limit/offset), чтобы
// фронт мог показывать «Всего сотрудников: N» / «Найдено: M» без полного обхода
// страниц. `limit=0` возвращает только `total` с пустым массивом — полезно для
// дешёвого запроса счётчика.
func (h *UserHandler) SearchUsers(c *gin.Context) {
	q := c.Query("q")
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	// limit = 0 — это отдельный валидный случай «хочу только total». Различаем
	// его от отсутствия параметра / неразборной строки: тогда ставим дефолт 20.
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

	users, total, err := h.users.SearchUsers(c.Request.Context(), q, int32(limit), int32(offset))
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
		return
	}

	resp := make([]dto.UserResponse, len(users))
	for i := range users {
		resp[i] = mapUserToResponse(&users[i])
	}

	writeSuccess(c, gin.H{
		"users": resp,
		"total": total,
	})
}

// GET /api/v1/users/:id
func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	u, err := h.users.GetProfile(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrUserNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
		return
	}
	writeSuccess(c, mapUserToResponse(u))
}

// PATCH /api/v1/users/:id
func (h *UserHandler) UpdateUser(c *gin.Context) {
	currentUserUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	currentUserID := currentUserUUID.String()
	targetUserID := c.Param("id")

	req, ok := bindJSON[dto.UpdateUserProfileRequest](c)
	if !ok {
		return
	}

	// Пока нет ролей — считаем, что админа нет
	isAdmin := false

	// Для новых полей: если не переданы, сохраняем текущие значения
	existing, err := h.users.GetProfile(c.Request.Context(), targetUserID)
	if err != nil {
		if err == domain.ErrUserNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
		return
	}

	onVacation := existing.OnVacation
	if req.OnVacation != nil {
		onVacation = *req.OnVacation
	}
	isSick := existing.IsSick
	if req.IsSick != nil {
		isSick = *req.IsSick
	}
	altContactChannel := existing.AlternativeContactChannel
	if req.AlternativeContactChannel.Set {
		altContactChannel = req.AlternativeContactChannel.Ptr()
	}
	altContactInfo := existing.AlternativeContactInfo
	if req.AlternativeContactInfo.Set {
		altContactInfo = req.AlternativeContactInfo.Ptr()
	}

	position := existing.Position
	if req.Position.Set {
		position = req.Position.Ptr()
	}

	u, err := h.users.UpdateProfile(c.Request.Context(), currentUserID, targetUserID, req.FullName, req.Email, position, onVacation, isSick, altContactChannel, altContactInfo, isAdmin)
	if err != nil {
		if err == domain.ErrUserNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
		return
	}

	writeSuccess(c, mapUserToResponse(u))
}

// PUT /api/v1/users/:id/avatar
func (h *UserHandler) UpdateAvatar(c *gin.Context) {
	currentUserUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	currentUserID := currentUserUUID.String()
	targetUserID := c.Param("id")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Файл не предоставлен")
		return
	}
	defer file.Close()

	// ограничение размера ~5MB
	const maxSize = 5 * 1024 * 1024
	data, err := io.ReadAll(http.MaxBytesReader(c.Writer, file, maxSize))
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_FILE", "Файл слишком большой или поврежден")
		return
	}

	// Пока минимальная проверка: по имени файла / расширению. Можно позже добавить MIME-проверку.
	isAdmin := false

	u, err := h.users.UpdateAvatar(c.Request.Context(), currentUserID, targetUserID, header.Filename, data, isAdmin)
	if err != nil {
		if err == domain.ErrUserNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден")
			return
		}
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Допустимые форматы: jpg, jpeg, png, webp")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Внутренняя ошибка сервера")
		return
	}

	writeSuccess(c, mapUserToResponse(u))
}

// GET /api/v1/users/:id/project-roles
func (h *UserHandler) GetMyProjectRoles(c *gin.Context) {
	currentUserUUID, ok := requireUserUUID(c)
	if !ok {
		return
	}

	uid, ok := paramUUID(c, "id")
	if !ok {
		return
	}

	if currentUserUUID != uid {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "Можно запрашивать только свои проектные роли")
		return
	}

	memberships, err := h.members.ListByUser(c.Request.Context(), uid)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить проектные роли")
		return
	}

	resp := make([]dto.ProjectRoleResponse, 0, len(memberships))
	for _, m := range memberships {
		permSet := make(map[string]struct{})
		for _, roleID := range m.RoleIDs {
			perms, err := h.roleRepo.ListRolePermissions(c.Request.Context(), roleID)
			if err != nil {
				continue
			}
			for _, p := range perms {
				permSet[p.Code] = struct{}{}
			}
		}
		permissions := make([]string, 0, len(permSet))
		for code := range permSet {
			permissions = append(permissions, code)
		}

		resp = append(resp, dto.ProjectRoleResponse{
			ProjectID:   m.ProjectID,
			ProjectName: m.ProjectName,
			Roles:       toMemberRoleRefsDTO(m.Roles),
			Permissions: permissions,
		})
	}

	writeSuccess(c, resp)
}
