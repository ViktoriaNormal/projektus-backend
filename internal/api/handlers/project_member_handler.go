package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type ProjectMemberHandler struct {
	service *services.ProjectMemberService
}

func NewProjectMemberHandler(service *services.ProjectMemberService) *ProjectMemberHandler {
	return &ProjectMemberHandler{service: service}
}

func (h *ProjectMemberHandler) ListMembers(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный идентификатор проекта")
		return
	}

	members, err := h.service.ListMembers(c.Request.Context(), projectID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить список участников")
		return
	}

	resp := make([]dto.ProjectMemberResponse, 0, len(members))
	for _, m := range members {
		resp = append(resp, dto.ProjectMemberResponse{
			ID:        m.ID,
			ProjectID: m.ProjectID,
			UserID:    m.UserID,
			Roles:     m.Roles,
		})
	}
	writeSuccess(c, resp)
}

func (h *ProjectMemberHandler) AddMember(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный идентификатор проекта")
		return
	}

	var req dto.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	member, err := h.service.AddMember(c.Request.Context(), projectID, req.UserID, req.Roles)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось добавить участника")
		return
	}

	writeSuccess(c, dto.ProjectMemberResponse{
		ID:        member.ID,
		ProjectID: member.ProjectID,
		UserID:    member.UserID,
		Roles:     member.Roles,
	})
}

func (h *ProjectMemberHandler) RemoveMember(c *gin.Context) {
	memberIDStr := c.Param("memberId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный идентификатор участника")
		return
	}

	if err := h.service.RemoveMember(c.Request.Context(), memberID); err != nil {
		if err == domain.ErrLastProjectAdmin {
			writeError(c, http.StatusBadRequest, "LAST_PROJECT_ADMIN", "Нельзя удалить последнего участника с ролью «Администратор проекта»")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить участника")
		return
	}

	writeSuccess(c, gin.H{"message": "Участник удален"})
}

func (h *ProjectMemberHandler) UpdateMemberRoles(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный идентификатор проекта")
		return
	}
	memberIDStr := c.Param("memberId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный идентификатор участника")
		return
	}

	var req dto.UpdateMemberRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	member, err := h.service.UpdateMemberRoles(c.Request.Context(), memberID, projectID, req.Roles)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Участник не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить роли участника")
		return
	}

	writeSuccess(c, dto.ProjectMemberResponse{
		ID:        member.ID,
		ProjectID: member.ProjectID,
		UserID:    member.UserID,
		Roles:     member.Roles,
	})
}

