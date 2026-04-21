package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type ProjectParamHandler struct {
	service        *services.ProjectParamService
	projectService *services.ProjectService
}

func NewProjectParamHandler(service *services.ProjectParamService, projectService *services.ProjectService) *ProjectParamHandler {
	return &ProjectParamHandler{service: service, projectService: projectService}
}

func (h *ProjectParamHandler) ListParams(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	// Generate system params with real values from the project.
	project, err := h.projectService.GetProject(c.Request.Context(), projectID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить проект")
		return
	}
	systemParams := domain.GenerateSystemProjectParams(project)

	// Get custom params from DB.
	customParams, err := h.service.ListParams(c.Request.Context(), projectID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить параметры проекта")
		return
	}

	allParams := append(systemParams, customParams...)
	resp := make([]dto.ProjectParamResponse, 0, len(allParams))
	for _, p := range allParams {
		resp = append(resp, mapProjectParamToDTO(p))
	}
	writeSuccess(c, resp)
}

func (h *ProjectParamHandler) CreateParam(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.CreateProjectParamRequest](c)
	if !ok {
		return
	}
	param, err := h.service.CreateParam(c.Request.Context(), projectID, req.Name, req.FieldType, req.IsRequired, req.Options, req.Value)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать параметр")
		return
	}
	writeSuccess(c, mapProjectParamToDTO(*param))
}

func (h *ProjectParamHandler) UpdateParam(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}
	paramID, ok := paramUUID(c, "paramId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.UpdateProjectParamRequest](c)
	if !ok {
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
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить параметр")
		return
	}
	writeSuccess(c, mapProjectParamToDTO(*param))
}

func (h *ProjectParamHandler) DeleteParam(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}
	paramID, ok := paramUUID(c, "paramId")
	if !ok {
		return
	}
	err := h.service.DeleteParam(c.Request.Context(), projectID, paramID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить параметр")
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
		ID:         p.ID,
		Name:       p.Name,
		FieldType:  p.FieldType,
		IsSystem:   p.IsSystem,
		IsRequired: p.IsRequired,
		Options:    opts,
		Value:      p.Value,
	}
}
