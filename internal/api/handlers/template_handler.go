package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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

// GET /v1/admin/project-templates/references
func (h *TemplateHandler) GetReferences(c *gin.Context) {
	refs, err := h.service.GetReferences(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось загрузить справочники")
		return
	}

	resp := dto.ReferencesResponse{
		ColumnSystemTypes:    make([]dto.ReferenceColumnType, 0, len(refs.ColumnSystemTypes)),
		FieldTypes:           make([]dto.ReferenceFieldType, 0, len(refs.FieldTypes)),
		EstimationUnits:      make([]dto.ReferenceAvailable, 0, len(refs.EstimationUnits)),
		PriorityTypeOptions:         make([]dto.ReferencePriorityType, 0, len(refs.PriorityTypeOptions)),
		SystemTaskFields:     make([]dto.ReferenceSystemField, 0, len(refs.SystemTaskFields)),
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
	for _, ps := range refs.ProjectStatuses {
		resp.ProjectStatuses = append(resp.ProjectStatuses, dto.ReferenceKeyName{Key: ps.Key, Name: ps.Name})
	}
	for _, sf := range refs.SystemTaskFields {
		resp.SystemTaskFields = append(resp.SystemTaskFields, dto.ReferenceSystemField{
			Key: sf.Key, Name: sf.Name, FieldType: sf.FieldType, AvailableFor: sf.AvailableFor, Description: sf.Description,
		})
	}
	for _, sp := range refs.SystemProjectParams {
		resp.SystemProjectParams = append(resp.SystemProjectParams, dto.ReferenceSystemProjectParam{
			Key: sp.Key, Name: sp.Name, FieldType: sp.FieldType, IsRequired: sp.IsRequired, Options: sp.Options,
		})
	}
	resp.PermissionAreas = make([]dto.ReferencePermissionArea, 0, len(refs.PermissionAreas))
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

// GET /v1/admin/project-templates
func (h *TemplateHandler) ListTemplates(c *gin.Context) {
	templates, allData, err := h.service.List(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить шаблоны проектов")
		return
	}
	resp := make([]dto.ProjectTemplateResponse, 0, len(templates))
	for i, t := range templates {
		resp = append(resp, mapTemplateToResponse(&t, allData[i]))
	}
	writeSuccess(c, resp)
}

// POST /v1/admin/project-templates
func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	var req dto.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}

	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}

	tmpl, data, err := h.service.Create(c.Request.Context(), req.Name, desc, req.ProjectType)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать шаблон")
		return
	}

	writeSuccess(c, mapTemplateToResponse(tmpl, data))
}

// GET /v1/admin/project-templates/:templateId
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор шаблона")
		return
	}

	tmpl, data, err := h.service.GetByID(c.Request.Context(), templateID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Шаблон не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить шаблон")
		return
	}

	writeSuccess(c, mapTemplateToResponse(tmpl, data))
}

// PATCH /v1/admin/project-templates/:templateId
func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор шаблона")
		return
	}

	var req dto.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}

	tmpl, data, err := h.service.Update(c.Request.Context(), templateID, req.Name, req.Description)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Шаблон не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить шаблон")
		return
	}

	writeSuccess(c, mapTemplateToResponse(tmpl, data))
}

// DELETE /v1/admin/project-templates/:templateId
func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор шаблона")
		return
	}

	err = h.service.Delete(c.Request.Context(), templateID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Шаблон не найден")
			return
		}
		if strings.Contains(err.Error(), "TEMPLATE_IN_USE") {
			writeError(c, http.StatusBadRequest, "TEMPLATE_IN_USE", "Шаблон используется проектами")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить шаблон")
		return
	}

	writeSuccess(c, nil)
}

// POST /v1/admin/project-templates/:templateId/boards
func (h *TemplateHandler) CreateBoard(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор шаблона")
		return
	}

	var req dto.CreateTemplateBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}

	board, err := h.service.CreateBoard(c.Request.Context(), templateID, req.Name, req.Description, req.IsDefault, req.PriorityType, req.EstimationUnit, req.SwimlaneGroupBy)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Шаблон не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать доску")
		return
	}

	writeSuccess(c, mapBoardToResponse(board))
}

// PATCH /v1/admin/project-templates/:templateId/boards/:boardId
func (h *TemplateHandler) UpdateBoard(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор шаблона")
		return
	}
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}

	var raw map[string]json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}
	var req dto.UpdateTemplateBoardRequest
	rawBytes, _ := json.Marshal(raw)
	_ = json.Unmarshal(rawBytes, &req)

	// Detect explicit null for swimlaneGroupBy
	clearSwimlaneGroup := false
	var swimlaneGroupBy *string
	if rawVal, exists := raw["swimlane_group_by"]; exists && string(rawVal) == "null" {
		clearSwimlaneGroup = true
	} else if req.SwimlaneGroupBy != nil {
		swimlaneGroupBy = req.SwimlaneGroupBy
	}

	board, err := h.service.UpdateBoard(c.Request.Context(), templateID, boardID, req.Name, req.Description, req.IsDefault, req.Order, req.PriorityType, req.EstimationUnit, swimlaneGroupBy, clearSwimlaneGroup)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить доску")
		return
	}

	writeSuccess(c, mapBoardToResponse(board))
}

// DELETE /v1/admin/project-templates/:templateId/boards/:boardId
func (h *TemplateHandler) DeleteBoard(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор шаблона")
		return
	}
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}

	err = h.service.DeleteBoard(c.Request.Context(), templateID, boardID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
			return
		}
		if strings.Contains(err.Error(), "DEFAULT_BOARD_DELETE") {
			writeError(c, http.StatusBadRequest, "DEFAULT_BOARD_DELETE", "Нельзя удалить доску по умолчанию")
			return
		}
		if strings.Contains(err.Error(), "LAST_BOARD_DELETE") {
			writeError(c, http.StatusBadRequest, "LAST_BOARD_DELETE", "Нельзя удалить единственную доску")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить доску")
		return
	}

	writeSuccess(c, nil)
}

// PATCH /v1/admin/project-templates/:templateId/boards/reorder
func (h *TemplateHandler) ReorderBoards(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор шаблона")
		return
	}

	var req dto.ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}

	orders := make(map[uuid.UUID]int32)
	for _, o := range req.Orders {
		orders[o.ID] = o.Order
	}

	if err := h.service.ReorderBoards(c.Request.Context(), templateID, orders); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось изменить порядок досок")
		return
	}

	writeSuccess(c, nil)
}

// POST /v1/admin/project-templates/:templateId/boards/:boardId/columns
func (h *TemplateHandler) CreateColumn(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор шаблона")
		return
	}
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}

	var req dto.CreateTemplateBoardColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}

	note := ""
	if req.Note != nil {
		note = *req.Note
	}
	col, err := h.service.CreateColumn(c.Request.Context(), templateID, boardID, req.Name, req.SystemType, req.WipLimit, req.Order, note)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать колонку")
		return
	}

	writeSuccess(c, mapColumnToResponse(col))
}

// PATCH /v1/admin/project-templates/:templateId/boards/:boardId/columns/:columnId
func (h *TemplateHandler) UpdateColumn(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	columnID, err := uuid.Parse(c.Param("columnId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}

	var raw map[string]json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}

	var req dto.UpdateTemplateBoardColumnRequest
	rawBytes, _ := json.Marshal(raw)
	_ = json.Unmarshal(rawBytes, &req)

	// Различаем "wip_limit не передан" vs "wip_limit: null"
	clearWipLimit := false
	if rawVal, exists := raw["wip_limit"]; exists && string(rawVal) == "null" {
		clearWipLimit = true
	}
	clearNote := false
	if rawVal, exists := raw["note"]; exists && string(rawVal) == "null" {
		clearNote = true
	}

	col, err := h.service.UpdateColumn(c.Request.Context(), templateID, boardID, columnID, req.Name, req.SystemType, req.WipLimit, clearWipLimit, req.Note, clearNote)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Колонка не найдена")
			return
		}
		if strings.Contains(err.Error(), "COLUMN_LOCKED") {
			writeError(c, http.StatusBadRequest, "COLUMN_LOCKED", "Нельзя редактировать заблокированную колонку")
			return
		}
		if strings.Contains(err.Error(), "INVALID_COLUMN_ORDER") {
			writeError(c, http.StatusBadRequest, "INVALID_COLUMN_ORDER", "Нарушен порядок типов колонок")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить колонку")
		return
	}

	writeSuccess(c, mapColumnToResponse(col))
}

// DELETE /v1/admin/project-templates/:templateId/boards/:boardId/columns/:columnId
func (h *TemplateHandler) DeleteColumn(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	columnID, err := uuid.Parse(c.Param("columnId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}

	err = h.service.DeleteColumn(c.Request.Context(), templateID, boardID, columnID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Колонка не найдена")
			return
		}
		if strings.Contains(err.Error(), "COLUMN_LOCKED") {
			writeError(c, http.StatusBadRequest, "COLUMN_LOCKED", "Нельзя удалить заблокированную колонку")
			return
		}
		if strings.Contains(err.Error(), "INVALID_COLUMN_ORDER") {
			writeError(c, http.StatusBadRequest, "INVALID_COLUMN_ORDER", "После удаления нарушится порядок типов")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить колонку")
		return
	}

	writeSuccess(c, nil)
}

// PATCH /v1/admin/project-templates/:templateId/boards/:boardId/columns/reorder
func (h *TemplateHandler) ReorderColumns(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}

	var req dto.ReorderColumnsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}

	orders := make(map[uuid.UUID]int32)
	for _, o := range req.Orders {
		orders[o.ColumnID] = o.Order
	}

	err = h.service.ReorderColumns(c.Request.Context(), templateID, boardID, orders)
	if err != nil {
		if strings.Contains(err.Error(), "COLUMN_LOCKED") {
			writeError(c, http.StatusBadRequest, "COLUMN_LOCKED", "Нельзя перемещать заблокированные колонки")
			return
		}
		if strings.Contains(err.Error(), "INVALID_COLUMN_ORDER") {
			writeError(c, http.StatusBadRequest, "INVALID_COLUMN_ORDER", "Нарушен порядок типов колонок")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось изменить порядок колонок")
		return
	}

	writeSuccess(c, nil)
}

// POST /v1/admin/project-templates/:templateId/boards/:boardId/swimlanes
func (h *TemplateHandler) CreateSwimlane(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор шаблона")
		return
	}
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}

	var req dto.CreateTemplateBoardSwimlaneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}

	sw, err := h.service.CreateSwimlane(c.Request.Context(), templateID, boardID, req.Name, req.WipLimit, req.Order)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать дорожку")
		return
	}

	writeSuccess(c, mapSwimlaneToResponse(sw))
}

// PATCH /v1/admin/project-templates/:templateId/boards/:boardId/swimlanes/:swimlaneId
func (h *TemplateHandler) UpdateSwimlane(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	swimlaneID, err := uuid.Parse(c.Param("swimlaneId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}

	var raw map[string]json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}
	var req dto.UpdateTemplateBoardSwimlaneRequest
	rawBytes, _ := json.Marshal(raw)
	_ = json.Unmarshal(rawBytes, &req)

	clearWipLimit := false
	if rawVal, exists := raw["wip_limit"]; exists && string(rawVal) == "null" {
		clearWipLimit = true
	}
	clearNote := false
	if rawVal, exists := raw["note"]; exists && string(rawVal) == "null" {
		clearNote = true
	}

	sw, err := h.service.UpdateSwimlane(c.Request.Context(), templateID, boardID, swimlaneID, req.WipLimit, clearWipLimit, req.Note, clearNote)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Дорожка не найдена")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить дорожку")
		return
	}

	writeSuccess(c, mapSwimlaneToResponse(sw))
}

// DELETE /v1/admin/project-templates/:templateId/boards/:boardId/swimlanes/:swimlaneId
func (h *TemplateHandler) DeleteSwimlane(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	swimlaneID, err := uuid.Parse(c.Param("swimlaneId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}

	err = h.service.DeleteSwimlane(c.Request.Context(), templateID, boardID, swimlaneID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Дорожка не найдена")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить дорожку")
		return
	}

	writeSuccess(c, nil)
}

// PATCH /v1/admin/project-templates/:templateId/boards/:boardId/swimlanes/reorder
func (h *TemplateHandler) ReorderSwimlanes(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}

	var req dto.ReorderSwimlanesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}

	orders := make(map[uuid.UUID]int32)
	for _, o := range req.Orders {
		orders[o.SwimlaneID] = o.Order
	}

	if err := h.service.ReorderSwimlanes(c.Request.Context(), templateID, boardID, orders); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось изменить порядок дорожек")
		return
	}

	writeSuccess(c, nil)
}

// POST /v1/admin/project-templates/:templateId/boards/:boardId/custom-fields
func (h *TemplateHandler) CreateCustomField(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}

	var req dto.CreateTemplateBoardCustomFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}

	field, err := h.service.CreateCustomField(c.Request.Context(), templateID, boardID, req.Name, req.FieldType, req.IsRequired, req.Order, req.Options)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
			return
		}
		if err == domain.ErrInvalidFieldType {
			writeError(c, http.StatusBadRequest, "INVALID_FIELD_TYPE", "Тип поля недопустим для данного типа проекта")
			return
		}
		if err == domain.ErrConflict {
			writeError(c, http.StatusConflict, "DUPLICATE_NAME", "Поле с таким именем уже существует")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать поле")
		return
	}

	writeSuccess(c, mapCustomFieldToResponse(field))
}

// PATCH /v1/admin/project-templates/:templateId/boards/:boardId/custom-fields/:fieldId
func (h *TemplateHandler) UpdateCustomField(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	fieldID, err := uuid.Parse(c.Param("fieldId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}

	var req dto.UpdateTemplateBoardCustomFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}

	field, err := h.service.UpdateCustomField(c.Request.Context(), templateID, boardID, fieldID, req.Name, req.IsRequired, req.Options)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Название не может быть пустым")
			return
		}
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Поле не найдено")
			return
		}
		if err == domain.ErrSystemField {
			writeError(c, http.StatusBadRequest, "SYSTEM_FIELD", "Нельзя изменять системное поле")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить поле")
		return
	}

	writeSuccess(c, mapCustomFieldToResponse(field))
}

// DELETE /v1/admin/project-templates/:templateId/boards/:boardId/custom-fields/:fieldId
func (h *TemplateHandler) DeleteCustomField(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	fieldID, err := uuid.Parse(c.Param("fieldId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}

	err = h.service.DeleteCustomField(c.Request.Context(), templateID, boardID, fieldID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Поле не найдено")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить поле")
		return
	}

	writeSuccess(c, nil)
}

// PATCH /v1/admin/project-templates/:templateId/boards/:boardId/custom-fields/reorder
func (h *TemplateHandler) ReorderCustomFields(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}

	var req dto.ReorderFieldsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}

	orders := make(map[uuid.UUID]int32)
	for _, o := range req.Orders {
		orders[o.FieldID] = o.Order
	}

	if err := h.service.ReorderCustomFields(c.Request.Context(), templateID, boardID, orders); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось изменить порядок полей")
		return
	}

	writeSuccess(c, nil)
}

// --- Response mapping helpers ---

// --- Project Params handlers ---

func (h *TemplateHandler) CreateProjectParam(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	var req dto.CreateTemplateProjectParamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}
	p, err := h.service.CreateProjectParam(c.Request.Context(), templateID, req.Name, req.FieldType, req.IsRequired, req.Order, req.Options)
	if err != nil {
		if err == domain.ErrInvalidFieldType {
			writeError(c, http.StatusBadRequest, "INVALID_FIELD_TYPE", "Тип поля недопустим для параметров проекта")
			return
		}
		if err == domain.ErrConflict {
			writeError(c, http.StatusConflict, "DUPLICATE_NAME", "Параметр с таким именем уже существует")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать параметр")
		return
	}
	writeSuccess(c, mapProjectParamToResponse(p))
}

func (h *TemplateHandler) UpdateProjectParam(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	paramID, err := uuid.Parse(c.Param("paramId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	var req dto.UpdateTemplateProjectParamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}
	p, err := h.service.UpdateProjectParam(c.Request.Context(), templateID, paramID, req.Name, req.IsRequired, req.Options)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Название не может быть пустым")
			return
		}
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Параметр не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить параметр")
		return
	}
	writeSuccess(c, mapProjectParamToResponse(p))
}

func (h *TemplateHandler) DeleteProjectParam(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	paramID, err := uuid.Parse(c.Param("paramId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	if err := h.service.DeleteProjectParam(c.Request.Context(), templateID, paramID); err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Параметр не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить параметр")
		return
	}
	writeSuccess(c, nil)
}

func (h *TemplateHandler) ReorderProjectParams(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	var req dto.ReorderParamsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}
	orders := make(map[uuid.UUID]int32)
	for _, o := range req.Orders {
		orders[o.ParamID] = o.Order
	}
	if err := h.service.ReorderProjectParams(c.Request.Context(), templateID, orders); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось изменить порядок")
		return
	}
	writeSuccess(c, nil)
}

// --- Roles handlers ---

func (h *TemplateHandler) CreateRole(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	var req dto.CreateTemplateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}
	perms := make([]domain.TemplateRolePermission, 0, len(req.Permissions))
	for _, p := range req.Permissions {
		perms = append(perms, domain.TemplateRolePermission{Area: p.Area, Access: p.Access})
	}
	role, err := h.service.CreateRole(c.Request.Context(), templateID, req.Name, req.Description, perms)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Название не может быть пустым")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать роль")
		return
	}
	writeSuccess(c, mapRoleToResponse(role))
}

func (h *TemplateHandler) UpdateRole(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	roleID, err := uuid.Parse(c.Param("roleId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	var req dto.UpdateTemplateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные")
		return
	}
	var perms []domain.TemplateRolePermission
	if req.Permissions != nil {
		perms = make([]domain.TemplateRolePermission, 0, len(req.Permissions))
		for _, p := range req.Permissions {
			perms = append(perms, domain.TemplateRolePermission{Area: p.Area, Access: p.Access})
		}
	}
	role, err := h.service.UpdateRole(c.Request.Context(), templateID, roleID, req.Name, req.Description, perms)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Название не может быть пустым")
			return
		}
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Роль не найдена")
			return
		}
		if err == domain.ErrTemplateAdminRole {
			writeError(c, http.StatusBadRequest, "TEMPLATE_ADMIN_ROLE", "Нельзя изменять права доступа роли администратора проекта в шаблоне")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить роль")
		return
	}
	writeSuccess(c, mapRoleToResponse(role))
}

func (h *TemplateHandler) DeleteRole(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("templateId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	roleID, err := uuid.Parse(c.Param("roleId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор")
		return
	}
	if err := h.service.DeleteRole(c.Request.Context(), templateID, roleID); err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Роль не найдена")
			return
		}
		if err == domain.ErrTemplateAdminRole {
			writeError(c, http.StatusBadRequest, "TEMPLATE_ADMIN_ROLE", "Нельзя удалить роль администратора проекта из шаблона")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить роль")
		return
	}
	writeSuccess(c, nil)
}

// --- Response mapping helpers ---

func mapTemplateToResponse(tmpl *domain.ProjectTemplate, data services.TemplateFullData) dto.ProjectTemplateResponse {
	desc := ""
	if tmpl.Description != nil {
		desc = *tmpl.Description
	}
	boardResp := make([]dto.TemplateBoardResponse, 0, len(data.Boards))
	for _, b := range data.Boards {
		boardResp = append(boardResp, mapBoardToResponse(b))
	}
	paramResp := make([]dto.TemplateProjectParamResponse, 0, len(data.Params))
	for _, p := range data.Params {
		paramResp = append(paramResp, mapProjectParamToResponse(p))
	}
	roleResp := make([]dto.TemplateRoleResponse, 0, len(data.Roles))
	for _, r := range data.Roles {
		roleResp = append(roleResp, mapRoleToResponse(r))
	}
	return dto.ProjectTemplateResponse{
		ID:          tmpl.ID,
		Name:        tmpl.Name,
		Description: desc,
		ProjectType: string(tmpl.Type),
		Boards:      boardResp,
		Params:      paramResp,
		Roles:       roleResp,
	}
}

func mapProjectParamToResponse(p domain.TemplateProjectParam) dto.TemplateProjectParamResponse {
	return dto.TemplateProjectParamResponse{
		ID: p.ID, Name: p.Name, Description: p.Description, FieldType: p.FieldType,
		IsSystem: p.IsSystem, IsRequired: p.IsRequired, Order: p.Order, Options: p.Options,
	}
}

func mapRoleToResponse(r domain.TemplateRole) dto.TemplateRoleResponse {
	perms := make([]dto.TemplateRolePermissionResponse, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		perms = append(perms, dto.TemplateRolePermissionResponse{Area: p.Area, Access: p.Access})
	}
	return dto.TemplateRoleResponse{
		ID: r.ID, Name: r.Name, Description: r.Description,
		IsAdmin: r.IsAdmin, Permissions: perms,
	}
}

func mapBoardToResponse(b domain.TemplateBoard) dto.TemplateBoardResponse {
	var sgb *string
	if b.SwimlaneGroupBy != "" {
		sgb = &b.SwimlaneGroupBy
	}

	columns := make([]dto.TemplateBoardColumnResponse, 0, len(b.Columns))
	for _, col := range b.Columns {
		columns = append(columns, mapColumnToResponse(col))
	}

	swimlanes := make([]dto.TemplateBoardSwimlaneResponse, 0, len(b.Swimlanes))
	for _, sw := range b.Swimlanes {
		swimlanes = append(swimlanes, mapSwimlaneToResponse(sw))
	}

	fields := make([]dto.TemplateBoardCustomFieldResponse, 0, len(b.CustomFields))
	for _, f := range b.CustomFields {
		fields = append(fields, mapCustomFieldToResponse(f))
	}

	return dto.TemplateBoardResponse{
		ID:              b.ID,
		Name:            b.Name,
		Description:     b.Description,
		IsDefault:       b.IsDefault,
		Order:           b.Order,
		PriorityType:    b.PriorityType,
		EstimationUnit:  b.EstimationUnit,
		SwimlaneGroupBy: sgb,
		Columns:         columns,
		Swimlanes:       swimlanes,
		Fields:          fields,
	}
}

func mapColumnToResponse(col domain.TemplateBoardColumn) dto.TemplateBoardColumnResponse {
	return dto.TemplateBoardColumnResponse{
		ID:         col.ID,
		Name:       col.Name,
		SystemType: col.SystemType,
		WipLimit:   col.WipLimit,
		Order:      col.Order,
		IsLocked:   col.IsLocked,
		Note:       col.Note,
	}
}

func mapSwimlaneToResponse(sw domain.TemplateBoardSwimlane) dto.TemplateBoardSwimlaneResponse {
	return dto.TemplateBoardSwimlaneResponse{
		ID:       sw.ID,
		Name:     sw.Name,
		WipLimit: sw.WipLimit,
		Order:    sw.Order,
		Note:     sw.Note,
	}
}

func mapCustomFieldToResponse(f domain.TemplateBoardCustomField) dto.TemplateBoardCustomFieldResponse {
	return dto.TemplateBoardCustomFieldResponse{
		ID:          f.ID,
		Name:        f.Name,
		Description: f.Description,
		FieldType:   f.FieldType,
		IsSystem:    f.IsSystem,
		IsRequired:  f.IsRequired,
		Order:       f.Order,
		Options:     f.Options,
	}
}
