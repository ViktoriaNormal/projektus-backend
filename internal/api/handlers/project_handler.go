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
	service       *services.ProjectService
	templateSvc   *services.TemplateService
	permissionSvc *services.PermissionService
}

func NewProjectHandler(service *services.ProjectService, templateSvc *services.TemplateService, permissionSvc *services.PermissionService) *ProjectHandler {
	return &ProjectHandler{service: service, templateSvc: templateSvc, permissionSvc: permissionSvc}
}

func (h *ProjectHandler) GetReferences(c *gin.Context) {
	refs, err := h.templateSvc.GetReferences(c.Request.Context())
	if err != nil {
		respondInternal(c, err, "Не удалось загрузить справочники")
		return
	}

	resp := dto.ProjectReferencesResponse{
		ColumnSystemTypes:   make([]dto.ReferenceColumnType, 0, len(refs.ColumnSystemTypes)),
		FieldTypes:          make([]dto.ReferenceFieldType, 0, len(refs.FieldTypes)),
		EstimationUnits:     make([]dto.ReferenceAvailable, 0, len(refs.EstimationUnits)),
		PriorityTypeOptions: make([]dto.ReferencePriorityType, 0, len(refs.PriorityTypeOptions)),
		PermissionAreas:     make([]dto.ReferencePermissionArea, 0, len(refs.PermissionAreas)),
		AccessLevels:        make([]dto.ReferenceKeyName, 0, len(refs.AccessLevels)),
	}

	for _, ct := range refs.ColumnSystemTypes {
		resp.ColumnSystemTypes = append(resp.ColumnSystemTypes, dto.ReferenceColumnType{
			Key: ct.Key, Name: ct.Name, Description: ct.Description,
		})
	}
	for _, ft := range refs.FieldTypes {
		resp.FieldTypes = append(resp.FieldTypes, dto.ReferenceFieldType{
			Key: ft.Key, Name: ft.Name, AvailableFor: ft.AvailableFor, AllowedScopes: ft.AllowedScopes,
		})
	}
	for _, eu := range refs.EstimationUnits {
		resp.EstimationUnits = append(resp.EstimationUnits, dto.ReferenceAvailable{
			Key: eu.Key, Name: eu.Name, AvailableFor: eu.AvailableFor,
		})
	}
	for _, pt := range refs.PriorityTypeOptions {
		resp.PriorityTypeOptions = append(resp.PriorityTypeOptions, dto.ReferencePriorityType{
			Key: pt.Key, Name: pt.Name, AvailableFor: pt.AvailableFor, DefaultValues: pt.DefaultValues,
		})
	}
	for _, a := range refs.PermissionAreas {
		resp.PermissionAreas = append(resp.PermissionAreas, dto.ReferencePermissionArea{
			Area: a.Area, Name: a.Name, Description: a.Description, AvailableFor: a.AvailableFor,
		})
	}
	for _, l := range refs.AccessLevels {
		resp.AccessLevels = append(resp.AccessLevels, dto.ReferenceKeyName{Key: l.Key, Name: l.Name})
	}

	writeSuccess(c, resp)
}

func (h *ProjectHandler) ListProjects(c *gin.Context) {
	userID, ok := requireUserUUID(c)
	if !ok {
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

	// system.projects.manage = full/view → все проекты; none → только свои
	sysAccess := h.permissionSvc.GetProjectManageAccess(c.Request.Context(), userID)

	var projects []domain.Project
	var err error
	if sysAccess == "full" || sysAccess == "view" {
		projects, err = h.service.ListAllProjects(c.Request.Context(), queryPtr, statusPtr, typePtr)
	} else {
		projects, err = h.service.ListProjects(c.Request.Context(), userID, queryPtr, statusPtr, typePtr)
	}
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить список проектов")
		return
	}

	resp := make([]dto.ProjectResponse, 0, len(projects))
	for _, p := range projects {
		resp = append(resp, mapProjectToDTO(&p))
	}
	writeSuccess(c, resp)
}

func (h *ProjectHandler) CreateProject(c *gin.Context) {
	currentUserID, ok := requireUserUUID(c)
	if !ok {
		return
	}

	req, ok := bindJSON[dto.CreateProjectRequest](c)
	if !ok {
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
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать проект")
		return
	}

	writeSuccess(c, mapProjectToDTO(p))
}

func (h *ProjectHandler) GetProject(c *gin.Context) {
	id, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	p, err := h.service.GetProject(c.Request.Context(), id)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить проект")
		return
	}

	writeSuccess(c, mapProjectToDTO(p))
}

func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	id, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.UpdateProjectRequest](c)
	if !ok {
		return
	}

	// Получаем текущий проект
	p, err := h.service.GetProject(c.Request.Context(), id)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить проект")
		return
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Description.Set {
		p.Description = req.Description.Ptr()
	}
	if req.Status != nil {
		p.Status = domain.ProjectStatus(*req.Status)
	}
	var newOwnerID *uuid.UUID
	if req.OwnerID != nil {
		ownerID, err := uuid.Parse(*req.OwnerID)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный owner_id")
			return
		}
		p.OwnerID = ownerID
		newOwnerID = &ownerID
	}
	if req.SprintDurationWeeks != nil {
		p.SprintDurationWeeks = req.SprintDurationWeeks
	}
	if req.IncompleteTasksAction != nil {
		p.IncompleteTasksAction = *req.IncompleteTasksAction
	}

	updated, err := h.service.UpdateProject(c.Request.Context(), p, newOwnerID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить проект")
		return
	}

	writeSuccess(c, mapProjectToDTO(updated))
}

func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	id, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	confirm := c.Query("confirm") == "true"
	if err := h.service.DeleteProject(c.Request.Context(), id, confirm); err != nil {
		// Особый случай: ErrInvalidInput здесь означает отсутствие confirm=true
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "CONFIRM_REQUIRED", "Для удаления проекта требуется confirm=true")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить проект")
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
		ID:                    p.ID,
		Key:                   p.Key,
		Name:                  p.Name,
		Description:           desc,
		ProjectType:           string(p.Type),
		OwnerID:               p.OwnerID,
		Status:                string(p.Status),
		SprintDurationWeeks:   p.SprintDurationWeeks,
		IncompleteTasksAction: p.IncompleteTasksAction,
		CreatedAt:             p.CreatedAt.Format(time.RFC3339),
	}
	if p.Owner != nil {
		resp.Owner = &dto.ProjectOwnerResponse{
			ID:        p.Owner.ID.String(),
			FullName:  p.Owner.FullName,
			AvatarURL: p.Owner.AvatarURL,
			Email:     p.Owner.Email,
		}
	}
	return resp
}
