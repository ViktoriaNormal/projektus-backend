package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type TemplateHandler struct {
	service *services.TemplateService
}

func NewTemplateHandler(service *services.TemplateService) *TemplateHandler {
	return &TemplateHandler{service: service}
}

func (h *TemplateHandler) ListTemplates(c *gin.Context) {
	templates, err := h.service.List(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить шаблоны проектов")
		return
	}
	resp := make([]dto.ProjectTemplateResponse, 0, len(templates))
	for _, t := range templates {
		desc := ""
		if t.Description != nil {
			desc = *t.Description
		}
		resp = append(resp, dto.ProjectTemplateResponse{
			ID:          t.ID,
			Name:        t.Name,
			Description: desc,
			ProjectType: string(t.Type),
		})
	}
	writeSuccess(c, resp)
}

func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	var req dto.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	t, err := h.service.Create(c.Request.Context(), req.Name, req.Description, req.ProjectType)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные параметры шаблона")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать шаблон проекта")
		return
	}
	desc := ""
	if t.Description != nil {
		desc = *t.Description
	}
	writeSuccess(c, dto.ProjectTemplateResponse{
		ID:          t.ID,
		Name:        t.Name,
		Description: desc,
		ProjectType: string(t.Type),
	})
}

