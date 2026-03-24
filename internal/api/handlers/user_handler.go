package handlers

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
	"projektus-backend/internal/services"
)

type UserHandler struct {
	users      services.UserService
	members    repositories.ProjectMemberRepository
	roleRepo   repositories.RoleRepository
}

func NewUserHandler(users services.UserService, members repositories.ProjectMemberRepository, roleRepo repositories.RoleRepository) *UserHandler {
	return &UserHandler{users: users, members: members, roleRepo: roleRepo}
}

func mapUserToResponse(u *domain.User) dto.UserResponse {
	return dto.UserResponse{
		ID:                        u.ID,
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
func (h *UserHandler) SearchUsers(c *gin.Context) {
	q := c.Query("q")
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	users, err := h.users.SearchUsers(c.Request.Context(), q, int32(limit), int32(offset))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		return
	}

	resp := make([]dto.UserResponse, len(users))
	for i := range users {
		resp[i] = mapUserToResponse(&users[i])
	}

	writeSuccess(c, gin.H{
		"users": resp,
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
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		return
	}
	writeSuccess(c, mapUserToResponse(u))
}

// PATCH /api/v1/users/:id
func (h *UserHandler) UpdateUser(c *gin.Context) {
	currentUserID := c.GetString("userID")
	if currentUserID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}
	targetUserID := c.Param("id")

	var req dto.UpdateUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
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
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
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
	if req.AlternativeContactChannel != nil {
		altContactChannel = req.AlternativeContactChannel
	}
	altContactInfo := existing.AlternativeContactInfo
	if req.AlternativeContactInfo != nil {
		altContactInfo = req.AlternativeContactInfo
	}

	u, err := h.users.UpdateProfile(c.Request.Context(), currentUserID, targetUserID, req.FullName, req.Email, req.Position, onVacation, isSick, altContactChannel, altContactInfo, isAdmin)
	if err != nil {
		switch err {
		case domain.ErrAccessDenied:
			writeError(c, http.StatusForbidden, "ACCESS_DENIED", "Недостаточно прав для изменения профиля")
		case domain.ErrUserNotFound:
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден")
		default:
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		}
		return
	}

	writeSuccess(c, mapUserToResponse(u))
}

// PUT /api/v1/users/:id/avatar
func (h *UserHandler) UpdateAvatar(c *gin.Context) {
	currentUserID := c.GetString("userID")
	if currentUserID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}
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
		switch err {
		case domain.ErrAccessDenied:
			writeError(c, http.StatusForbidden, "ACCESS_DENIED", "Недостаточно прав для изменения аватара")
		case domain.ErrUserNotFound:
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден")
		case domain.ErrInvalidInput:
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Допустимые форматы: jpg, jpeg, png, webp")
		default:
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера")
		}
		return
	}

	writeSuccess(c, mapUserToResponse(u))
}

// GET /api/v1/users/:id/project-roles
func (h *UserHandler) GetMyProjectRoles(c *gin.Context) {
	currentUserID := c.GetString("userID")
	if currentUserID == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}

	targetUserID := c.Param("id")
	if currentUserID != targetUserID {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "Можно запрашивать только свои проектные роли")
		return
	}

	uid, err := uuid.Parse(targetUserID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный идентификатор пользователя")
		return
	}

	memberships, err := h.members.ListByUser(c.Request.Context(), uid)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить проектные роли")
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
			Roles:       m.Roles,
			Permissions: permissions,
		})
	}

	writeSuccess(c, resp)
}

