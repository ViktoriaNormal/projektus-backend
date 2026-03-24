package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type ProjectHandler struct {
	service *services.ProjectService
}

func NewProjectHandler(service *services.ProjectService) *ProjectHandler {
	return &ProjectHandler{service: service}
}

func (h *ProjectHandler) ListProjects(c *gin.Context) {
	userIDStr := c.GetString("userID")
	ownerID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}

	q := c.Query("q")
	var queryPtr *string
	if q != "" {
		queryPtr = &q
	}
	status := c.Query("status")
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}
	projectType := c.Query("project_type")
	var typePtr *string
	if projectType != "" {
		typePtr = &projectType
	}

	projects, err := h.service.ListProjects(c.Request.Context(), ownerID, queryPtr, statusPtr, typePtr)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить список проектов")
		return
	}

	resp := make([]dto.ProjectResponse, 0, len(projects))
	for _, p := range projects {
		resp = append(resp, mapProjectToDTO(&p))
	}
	writeSuccess(c, resp)
}

func (h *ProjectHandler) CreateProject(c *gin.Context) {
	userIDStr := c.GetString("userID")
	currentUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}

	var req dto.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	ownerID := currentUserID
	if req.OwnerID != nil && *req.OwnerID != "" {
		parsed, err := uuid.Parse(*req.OwnerID)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный owner_id")
			return
		}
		ownerID = parsed
	}

	p, err := h.service.CreateProject(c.Request.Context(), ownerID, req.Name, req.Description, req.ProjectType)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные параметры проекта")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать проект")
		return
	}

	writeSuccess(c, mapProjectToDTO(p))
}

func (h *ProjectHandler) GetProject(c *gin.Context) {
	idStr := c.Param("projectId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный идентификатор проекта")
		return
	}

	p, err := h.service.GetProject(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Проект не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить проект")
		return
	}

	writeSuccess(c, mapProjectToDTO(p))
}

func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	idStr := c.Param("projectId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный идентификатор проекта")
		return
	}

	var req dto.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	// Получаем текущий проект
	p, err := h.service.GetProject(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Проект не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить проект")
		return
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Description != nil {
		p.Description = req.Description
	}
	if req.Status != nil {
		p.Status = domain.ProjectStatus(*req.Status)
	}

	updated, err := h.service.UpdateProject(c.Request.Context(), p)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Проект не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить проект")
		return
	}

	writeSuccess(c, mapProjectToDTO(updated))
}

func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	idStr := c.Param("projectId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный идентификатор проекта")
		return
	}

	confirm := c.Query("confirm") == "true"
	if err := h.service.DeleteProject(c.Request.Context(), id, confirm); err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "CONFIRM_REQUIRED", "Для удаления проекта требуется confirm=true")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить проект")
		return
	}

	writeSuccess(c, gin.H{"message": "Проект удален"})
}

func mapProjectToDTO(p *domain.Project) dto.ProjectResponse {
	desc := ""
	if p.Description != nil {
		desc = *p.Description
	}
	resp := dto.ProjectResponse{
		ID:          p.ID,
		Key:         p.Key,
		Name:        p.Name,
		Description: desc,
		ProjectType: string(p.Type),
		OwnerID:     p.OwnerID,
		Status:      string(p.Status),
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
	}
	if p.Owner != nil {
		resp.Owner = &dto.ProjectOwnerResponse{
			ID:        p.Owner.ID,
			FullName:  p.Owner.FullName,
			AvatarURL: p.Owner.AvatarURL,
			Email:     p.Owner.Email,
		}
	}
	return resp
}

