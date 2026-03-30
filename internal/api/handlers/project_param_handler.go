package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type ProjectParamHandler struct {
	service *services.ProjectParamService
}

func NewProjectParamHandler(service *services.ProjectParamService) *ProjectParamHandler {
	return &ProjectParamHandler{service: service}
}

func (h *ProjectParamHandler) ListParams(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}
	params, err := h.service.ListParams(c.Request.Context(), projectID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить параметры проекта")
		return
	}
	resp := make([]dto.ProjectParamResponse, 0, len(params))
	for _, p := range params {
		resp = append(resp, mapProjectParamToDTO(p))
	}
	writeSuccess(c, resp)
}

func (h *ProjectParamHandler) CreateParam(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}
	var req dto.CreateProjectParamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	param, err := h.service.CreateParam(c.Request.Context(), projectID, req.Name, req.FieldType, req.IsRequired, req.Options, req.Value)
	if err != nil {
		var pve *domain.ParamValidationError
		if errors.As(err, &pve) {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", pve.Message)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать параметр")
		return
	}
	writeSuccess(c, mapProjectParamToDTO(*param))
}

func (h *ProjectParamHandler) UpdateParam(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}
	paramID, err := uuid.Parse(c.Param("paramId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор параметра")
		return
	}
	var req dto.UpdateProjectParamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	var value *string
	clearValue := false
	if req.Value.Set {
		if req.Value.Null {
			clearValue = true
		} else {
			value = &req.Value.Value
		}
	}
	param, err := h.service.UpdateParam(c.Request.Context(), projectID, paramID, req.Name, req.IsRequired, req.Options, value, clearValue)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Параметр не найден")
			return
		}
		var pve *domain.ParamValidationError
		if errors.As(err, &pve) {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", pve.Message)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить параметр")
		return
	}
	writeSuccess(c, mapProjectParamToDTO(*param))
}

func (h *ProjectParamHandler) DeleteParam(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}
	paramID, err := uuid.Parse(c.Param("paramId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор параметра")
		return
	}
	err = h.service.DeleteParam(c.Request.Context(), projectID, paramID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Параметр не найден")
			return
		}
		if err == domain.ErrSystemParam {
			writeError(c, http.StatusBadRequest, "SYSTEM_PARAM", "Нельзя удалить системный обязательный параметр")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить параметр")
		return
	}
	c.Status(http.StatusNoContent)
}

func mapProjectParamToDTO(p domain.ProjectParam) dto.ProjectParamResponse {
	opts := p.Options
	if opts == nil {
		opts = []string{}
	}
	return dto.ProjectParamResponse{
		ID:          uuid.MustParse(p.ID),
		Name:        p.Name,
		Description: p.Description,
		FieldType:   p.FieldType,
		IsSystem:    p.IsSystem,
		IsRequired:  p.IsRequired,
		Options:     opts,
		Value:       p.Value,
	}
}
