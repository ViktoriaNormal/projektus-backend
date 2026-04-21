package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
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
		respondInternal(c, err, "Не удалось загрузить справочники")
		return
	}

	resp := dto.ReferencesResponse{
		ColumnSystemTypes:   make([]dto.ReferenceColumnType, 0, len(refs.ColumnSystemTypes)),
		FieldTypes:          make([]dto.ReferenceFieldType, 0, len(refs.FieldTypes)),
		EstimationUnits:     make([]dto.ReferenceAvailable, 0, len(refs.EstimationUnits)),
		PriorityTypeOptions: make([]dto.ReferencePriorityType, 0, len(refs.PriorityTypeOptions)),
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
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить шаблоны проектов")
		return
	}
	resp := make([]dto.ProjectTemplateResponse, 0, len(templates))
	for i, t := range templates {
		injectSystemFields(&t, &allData[i])
		resp = append(resp, mapTemplateToResponse(&t, allData[i]))
	}
	writeSuccess(c, resp)
}

// POST /v1/admin/project-templates
func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	req, ok := bindJSON[dto.CreateTemplateRequest](c)
	if !ok {
		return
	}

	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}

	tmpl, data, err := h.service.Create(c.Request.Context(), req.Name, desc, req.ProjectType)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать шаблон")
		return
	}

	injectSystemFields(tmpl, &data)
	writeSuccess(c, mapTemplateToResponse(tmpl, data))
}

// GET /v1/admin/project-templates/:templateId
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}

	tmpl, data, err := h.service.GetByID(c.Request.Context(), templateID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить шаблон")
		return
	}

	injectSystemFields(tmpl, &data)
	writeSuccess(c, mapTemplateToResponse(tmpl, data))
}

// PATCH /v1/admin/project-templates/:templateId
func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.UpdateTemplateRequest](c)
	if !ok {
		return
	}

	tmpl, data, err := h.service.Update(c.Request.Context(), templateID, req.Name, req.Description)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить шаблон")
		return
	}

	injectSystemFields(tmpl, &data)
	writeSuccess(c, mapTemplateToResponse(tmpl, data))
}

// DELETE /v1/admin/project-templates/:templateId
func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}

	err := h.service.Delete(c.Request.Context(), templateID)
	if err != nil {
		if strings.Contains(err.Error(), "TEMPLATE_IN_USE") {
			writeError(c, http.StatusBadRequest, "TEMPLATE_IN_USE", "Шаблон используется проектами")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить шаблон")
		return
	}

	writeSuccess(c, nil)
}

// POST /v1/admin/project-templates/:templateId/boards
func (h *TemplateHandler) CreateBoard(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.CreateTemplateBoardRequest](c)
	if !ok {
		return
	}

	board, err := h.service.CreateBoard(c.Request.Context(), templateID, req.Name, req.Description, req.IsDefault, req.PriorityType, req.EstimationUnit, req.SwimlaneGroupBy)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать доску")
		return
	}

	writeSuccess(c, mapBoardToResponse(board))
}

// PATCH /v1/admin/project-templates/:templateId/boards/:boardId
func (h *TemplateHandler) UpdateBoard(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.UpdateTemplateBoardRequest](c)
	if !ok {
		return
	}

	// Convert NullableField[string] → *string (empty string signals "set to NULL" in service).
	var desc *string
	if req.Description.Set {
		if req.Description.Null {
			empty := ""
			desc = &empty
		} else {
			desc = &req.Description.Value
		}
	}

	// Convert NullableField → (*string, clearFlag) for swimlaneGroupBy.
	clearSwimlaneGroup := false
	var swimlaneGroupBy *string
	if req.SwimlaneGroupBy.Set {
		if req.SwimlaneGroupBy.Null {
			clearSwimlaneGroup = true
		} else {
			swimlaneGroupBy = &req.SwimlaneGroupBy.Value
		}
	}

	board, err := h.service.UpdateBoard(c.Request.Context(), templateID, boardID, req.Name, desc, req.IsDefault, req.Order, req.PriorityType, req.EstimationUnit, swimlaneGroupBy, clearSwimlaneGroup, req.PriorityOptions)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить доску")
		return
	}

	writeSuccess(c, mapBoardToResponse(board))
}

// DELETE /v1/admin/project-templates/:templateId/boards/:boardId
func (h *TemplateHandler) DeleteBoard(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}

	err := h.service.DeleteBoard(c.Request.Context(), templateID, boardID)
	if err != nil {
		if strings.Contains(err.Error(), "DEFAULT_BOARD_DELETE") {
			writeError(c, http.StatusBadRequest, "DEFAULT_BOARD_DELETE", "Нельзя удалить доску по умолчанию")
			return
		}
		if strings.Contains(err.Error(), "LAST_BOARD_DELETE") {
			writeError(c, http.StatusBadRequest, "LAST_BOARD_DELETE", "Нельзя удалить единственную доску")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить доску")
		return
	}

	writeSuccess(c, nil)
}

// PATCH /v1/admin/project-templates/:templateId/boards/reorder
func (h *TemplateHandler) ReorderBoards(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.ReorderRequest](c)
	if !ok {
		return
	}

	orders := make(map[uuid.UUID]int32)
	for _, o := range req.Orders {
		orders[o.ID] = o.Order
	}

	if err := h.service.ReorderBoards(c.Request.Context(), templateID, orders); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось изменить порядок досок")
		return
	}

	writeSuccess(c, nil)
}

// POST /v1/admin/project-templates/:templateId/boards/:boardId/columns
func (h *TemplateHandler) CreateColumn(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.CreateTemplateBoardColumnRequest](c)
	if !ok {
		return
	}

	note := ""
	if req.Note != nil {
		note = *req.Note
	}
	col, err := h.service.CreateColumn(c.Request.Context(), templateID, boardID, req.Name, req.SystemType, req.WipLimit, req.Order, note)
	if err != nil {
		// Сохраняем исторический маппинг — VALIDATION_ERROR вместо COMPLETED_COLUMN_WIP.
		if err == domain.ErrCompletedColumnWip {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "WIP-лимит нельзя установить для колонок с типом \"Завершено\"")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать колонку")
		return
	}

	writeSuccess(c, mapColumnToResponse(col))
}

// PATCH /v1/admin/project-templates/:templateId/boards/:boardId/columns/:columnId
func (h *TemplateHandler) UpdateColumn(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	columnID, ok := paramUUID(c, "columnId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.UpdateTemplateBoardColumnRequest](c)
	if !ok {
		return
	}

	// Convert NullableField → (*int32, clearFlag) for wipLimit.
	var wipLimit *int32
	clearWipLimit := false
	if req.WipLimit.Set {
		if req.WipLimit.Null {
			clearWipLimit = true
		} else {
			wipLimit = &req.WipLimit.Value
		}
	}
	// Convert NullableField → (*string, clearFlag) for note.
	var note *string
	clearNote := false
	if req.Note.Set {
		if req.Note.Null {
			clearNote = true
		} else {
			note = &req.Note.Value
		}
	}

	col, err := h.service.UpdateColumn(c.Request.Context(), templateID, boardID, columnID, req.Name, req.SystemType, wipLimit, clearWipLimit, note, clearNote)
	if err != nil {
		if strings.Contains(err.Error(), "COLUMN_LOCKED") {
			writeError(c, http.StatusBadRequest, "COLUMN_LOCKED", "Нельзя редактировать заблокированную колонку")
			return
		}
		if strings.Contains(err.Error(), "INVALID_COLUMN_ORDER") {
			writeError(c, http.StatusBadRequest, "INVALID_COLUMN_ORDER", "Нарушен порядок типов колонок")
			return
		}
		if err == domain.ErrCompletedColumnWip {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "WIP-лимит нельзя установить для колонок с типом \"Завершено\"")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить колонку")
		return
	}

	writeSuccess(c, mapColumnToResponse(col))
}

// DELETE /v1/admin/project-templates/:templateId/boards/:boardId/columns/:columnId
func (h *TemplateHandler) DeleteColumn(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	columnID, ok := paramUUID(c, "columnId")
	if !ok {
		return
	}

	err := h.service.DeleteColumn(c.Request.Context(), templateID, boardID, columnID)
	if err != nil {
		if strings.Contains(err.Error(), "COLUMN_LOCKED") {
			writeError(c, http.StatusBadRequest, "COLUMN_LOCKED", "Нельзя удалить заблокированную колонку")
			return
		}
		if strings.Contains(err.Error(), "INVALID_COLUMN_ORDER") {
			writeError(c, http.StatusBadRequest, "INVALID_COLUMN_ORDER", "После удаления нарушится порядок типов")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить колонку")
		return
	}

	writeSuccess(c, nil)
}

// PATCH /v1/admin/project-templates/:templateId/boards/:boardId/columns/reorder
func (h *TemplateHandler) ReorderColumns(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.ReorderColumnsRequest](c)
	if !ok {
		return
	}

	orders := make(map[uuid.UUID]int32)
	for _, o := range req.Orders {
		orders[o.ColumnID] = o.Order
	}

	err := h.service.ReorderColumns(c.Request.Context(), templateID, boardID, orders)
	if err != nil {
		if strings.Contains(err.Error(), "COLUMN_LOCKED") {
			writeError(c, http.StatusBadRequest, "COLUMN_LOCKED", "Нельзя перемещать заблокированные колонки")
			return
		}
		if strings.Contains(err.Error(), "INVALID_COLUMN_ORDER") {
			writeError(c, http.StatusBadRequest, "INVALID_COLUMN_ORDER", "Нарушен порядок типов колонок")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось изменить порядок колонок")
		return
	}

	writeSuccess(c, nil)
}

// POST /v1/admin/project-templates/:templateId/boards/:boardId/swimlanes
func (h *TemplateHandler) CreateSwimlane(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.CreateTemplateBoardSwimlaneRequest](c)
	if !ok {
		return
	}

	sw, err := h.service.CreateSwimlane(c.Request.Context(), templateID, boardID, req.Name, req.WipLimit, req.Order)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать дорожку")
		return
	}

	writeSuccess(c, mapSwimlaneToResponse(sw))
}

// PATCH /v1/admin/project-templates/:templateId/boards/:boardId/swimlanes/:swimlaneId
func (h *TemplateHandler) UpdateSwimlane(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	swimlaneID, ok := paramUUID(c, "swimlaneId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.UpdateTemplateBoardSwimlaneRequest](c)
	if !ok {
		return
	}

	var wipLimit *int32
	clearWipLimit := false
	if req.WipLimit.Set {
		if req.WipLimit.Null {
			clearWipLimit = true
		} else {
			wipLimit = &req.WipLimit.Value
		}
	}
	var note *string
	clearNote := false
	if req.Note.Set {
		if req.Note.Null {
			clearNote = true
		} else {
			note = &req.Note.Value
		}
	}

	sw, err := h.service.UpdateSwimlane(c.Request.Context(), templateID, boardID, swimlaneID, wipLimit, clearWipLimit, note, clearNote)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить дорожку")
		return
	}

	writeSuccess(c, mapSwimlaneToResponse(sw))
}

// DELETE /v1/admin/project-templates/:templateId/boards/:boardId/swimlanes/:swimlaneId
func (h *TemplateHandler) DeleteSwimlane(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	swimlaneID, ok := paramUUID(c, "swimlaneId")
	if !ok {
		return
	}

	err := h.service.DeleteSwimlane(c.Request.Context(), templateID, boardID, swimlaneID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить дорожку")
		return
	}

	writeSuccess(c, nil)
}

// PATCH /v1/admin/project-templates/:templateId/boards/:boardId/swimlanes/reorder
func (h *TemplateHandler) ReorderSwimlanes(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.ReorderSwimlanesRequest](c)
	if !ok {
		return
	}

	orders := make(map[uuid.UUID]int32)
	for _, o := range req.Orders {
		orders[o.SwimlaneID] = o.Order
	}

	if err := h.service.ReorderSwimlanes(c.Request.Context(), templateID, boardID, orders); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось изменить порядок дорожек")
		return
	}

	writeSuccess(c, nil)
}

// POST /v1/admin/project-templates/:templateId/boards/:boardId/custom-fields
func (h *TemplateHandler) CreateCustomField(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.CreateTemplateBoardCustomFieldRequest](c)
	if !ok {
		return
	}

	field, err := h.service.CreateCustomField(c.Request.Context(), templateID, boardID, req.Name, req.FieldType, req.IsRequired, req.Options)
	if err != nil {
		// Сохраняем специфичный код DUPLICATE_NAME (не в таблице).
		if err == domain.ErrConflict {
			writeError(c, http.StatusConflict, "DUPLICATE_NAME", "Поле с таким именем уже существует")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать поле")
		return
	}

	writeSuccess(c, mapCustomFieldToResponse(field))
}

// PATCH /v1/admin/project-templates/:templateId/boards/:boardId/custom-fields/:fieldId
func (h *TemplateHandler) UpdateCustomField(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	fieldID, ok := paramUUID(c, "fieldId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.UpdateTemplateBoardCustomFieldRequest](c)
	if !ok {
		return
	}

	// Особый кейс: системные поля «Приоритизация» и «Оценка трудозатрат».
	// У них нет строк в таблице template_board_fields — они генерируются
	// из констант в runtime. Обновление options таких полей перенаправляется
	// на соответствующие поля доски.
	if field, handled, err := h.updateSystemFieldIfAny(c, templateID, boardID, fieldID, req); handled {
		if err != nil {
			if respondDomainErr(c, err) {
				return
			}
			respondInternal(c, err, "Не удалось обновить системное поле")
			return
		}
		writeSuccess(c, mapCustomFieldToResponse(field))
		return
	}

	field, err := h.service.UpdateCustomField(c.Request.Context(), templateID, boardID, fieldID, req.Name, req.IsRequired, req.Options)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить поле")
		return
	}

	writeSuccess(c, mapCustomFieldToResponse(field))
}

// updateSystemFieldIfAny обрабатывает PATCH на системные поля доски
// (Приоритизация, Оценка трудозатрат), которые хранятся не в отдельной
// таблице, а в полях самой доски. Возвращает (field, handled, err).
// Когда handled=false — это обычное кастомное поле, дальше работает
// стандартный путь через service.UpdateCustomField.
func (h *TemplateHandler) updateSystemFieldIfAny(c *gin.Context, templateID, boardID, fieldID uuid.UUID, req dto.UpdateTemplateBoardCustomFieldRequest) (domain.TemplateBoardCustomField, bool, error) {
	priorityID := domain.SystemBoardFieldIDs["priority"]
	estimationID := domain.SystemBoardFieldIDs["estimation"]

	switch fieldID {
	case priorityID:
		// Меняем priority_options на уровне доски — именно оттуда они
		// подтягиваются в системное поле через GenerateSystemBoardFields.
		// Передаём опции в UpdateBoard как *[]string; остальные параметры nil.
		var opts *[]string
		if req.Options != nil {
			v := req.Options
			opts = &v
		}
		if _, err := h.service.UpdateBoard(c.Request.Context(), templateID, boardID,
			nil, nil, nil, nil, nil, nil, nil, false, opts); err != nil {
			return domain.TemplateBoardCustomField{}, true, err
		}
		// Перечитываем доску, чтобы отдать актуальный набор опций обратно.
		board, err := h.service.GetBoardByID(c.Request.Context(), templateID, boardID)
		if err != nil {
			return domain.TemplateBoardCustomField{}, true, err
		}
		options := board.PriorityOptions
		if options == nil {
			options = []string{}
		}
		return domain.TemplateBoardCustomField{
			ID: priorityID, Name: "Приоритизация", FieldType: "priority",
			IsSystem: true, IsRequired: false, Options: options,
		}, true, nil

	case estimationID:
		// «Оценка трудозатрат» — это выбор единицы (story_points/time),
		// хранится в board.estimation_unit. Здесь массив options не имеет
		// смысла как таковой, поэтому просто возвращаем текущее состояние
		// идемпотентно — не ломая контракт.
		board, err := h.service.GetBoardByID(c.Request.Context(), templateID, boardID)
		if err != nil {
			return domain.TemplateBoardCustomField{}, true, err
		}
		return domain.TemplateBoardCustomField{
			ID: estimationID, Name: "Оценка трудозатрат", FieldType: "estimation",
			IsSystem: true, IsRequired: false, Options: []string{board.EstimationUnit},
		}, true, nil
	}
	return domain.TemplateBoardCustomField{}, false, nil
}

// DELETE /v1/admin/project-templates/:templateId/boards/:boardId/custom-fields/:fieldId
func (h *TemplateHandler) DeleteCustomField(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	fieldID, ok := paramUUID(c, "fieldId")
	if !ok {
		return
	}

	err := h.service.DeleteCustomField(c.Request.Context(), templateID, boardID, fieldID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить поле")
		return
	}

	writeSuccess(c, nil)
}

// --- Project Params handlers ---

func (h *TemplateHandler) CreateProjectParam(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.CreateTemplateProjectParamRequest](c)
	if !ok {
		return
	}
	p, err := h.service.CreateProjectParam(c.Request.Context(), templateID, req.Name, req.FieldType, req.IsRequired, req.Options)
	if err != nil {
		// Сохраняем специфичный код DUPLICATE_NAME (не в таблице).
		if err == domain.ErrConflict {
			writeError(c, http.StatusConflict, "DUPLICATE_NAME", "Параметр с таким именем уже существует")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать параметр")
		return
	}
	writeSuccess(c, mapProjectParamToResponse(p))
}

func (h *TemplateHandler) UpdateProjectParam(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	paramID, ok := paramUUID(c, "paramId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.UpdateTemplateProjectParamRequest](c)
	if !ok {
		return
	}
	p, err := h.service.UpdateProjectParam(c.Request.Context(), templateID, paramID, req.Name, req.IsRequired, req.Options)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить параметр")
		return
	}
	writeSuccess(c, mapProjectParamToResponse(p))
}

func (h *TemplateHandler) DeleteProjectParam(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	paramID, ok := paramUUID(c, "paramId")
	if !ok {
		return
	}
	if err := h.service.DeleteProjectParam(c.Request.Context(), templateID, paramID); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить параметр")
		return
	}
	writeSuccess(c, nil)
}

// --- Roles handlers ---

func (h *TemplateHandler) CreateRole(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.CreateTemplateRoleRequest](c)
	if !ok {
		return
	}
	perms := make([]domain.TemplateRolePermission, 0, len(req.Permissions))
	for _, p := range req.Permissions {
		perms = append(perms, domain.TemplateRolePermission{Area: p.Area, Access: p.Access})
	}
	role, err := h.service.CreateRole(c.Request.Context(), templateID, req.Name, req.Description, perms)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать роль")
		return
	}
	writeSuccess(c, mapRoleToResponse(role))
}

func (h *TemplateHandler) UpdateRole(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	roleID, ok := paramUUID(c, "roleId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.UpdateTemplateRoleRequest](c)
	if !ok {
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
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить роль")
		return
	}
	writeSuccess(c, mapRoleToResponse(role))
}

// ReorderRoles PATCH /v1/admin/project-templates/:templateId/roles/reorder
// Принимает массив { role_id, order } — применяет новые позиции одной батч-операцией.
func (h *TemplateHandler) ReorderRoles(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.ReorderTemplateRolesRequest](c)
	if !ok {
		return
	}
	orders := make(map[uuid.UUID]int32, len(req.Orders))
	for _, o := range req.Orders {
		orders[o.RoleID] = o.Order
	}
	if err := h.service.ReorderRoles(c.Request.Context(), templateID, orders); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось изменить порядок ролей")
		return
	}
	writeSuccess(c, nil)
}

func (h *TemplateHandler) DeleteRole(c *gin.Context) {
	templateID, ok := paramUUID(c, "templateId")
	if !ok {
		return
	}
	roleID, ok := paramUUID(c, "roleId")
	if !ok {
		return
	}
	if err := h.service.DeleteRole(c.Request.Context(), templateID, roleID); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить роль")
		return
	}
	writeSuccess(c, nil)
}

// --- Response mapping helpers ---

// injectSystemFields adds system board fields and project params into template data.
func injectSystemFields(tmpl *domain.ProjectTemplate, data *services.TemplateFullData) {
	// Inject system fields into each board's custom fields.
	for i, b := range data.Boards {
		systemFields := domain.GenerateSystemBoardFields(
			string(tmpl.Type), b.PriorityType, b.EstimationUnit,
			b.PriorityOptions, repositories.DefaultBoardFields,
		)
		sysTemplateFields := make([]domain.TemplateBoardCustomField, 0, len(systemFields))
		for _, sf := range systemFields {
			opts := sf.Options
			if opts == nil {
				opts = []string{}
			}
			sysTemplateFields = append(sysTemplateFields, domain.TemplateBoardCustomField{
				ID: sf.ID, Name: sf.Name,
				FieldType: sf.FieldType, IsSystem: true, IsRequired: sf.IsRequired, Options: opts,
			})
		}
		data.Boards[i].CustomFields = append(sysTemplateFields, b.CustomFields...)
	}

	// Inject system project params.
	sysParams := domain.GenerateSystemProjectParamsForTemplate()
	sysTemplateParams := make([]domain.TemplateProjectParam, 0, len(sysParams))
	for _, sp := range sysParams {
		sysTemplateParams = append(sysTemplateParams, domain.TemplateProjectParam{
			ID: sp.ID, Name: sp.Name,
			FieldType: sp.FieldType, IsSystem: true, IsRequired: sp.IsRequired,
			Options: []string{},
		})
	}
	data.Params = append(sysTemplateParams, data.Params...)
}

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
		ID: p.ID, Name: p.Name, FieldType: p.FieldType,
		IsSystem: p.IsSystem, IsRequired: p.IsRequired, Options: p.Options,
	}
}

func mapRoleToResponse(r domain.TemplateRole) dto.TemplateRoleResponse {
	perms := make([]dto.TemplateRolePermissionResponse, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		perms = append(perms, dto.TemplateRolePermissionResponse{Area: p.Area, Access: p.Access})
	}
	return dto.TemplateRoleResponse{
		ID: r.ID, Name: r.Name, Description: r.Description,
		IsAdmin: r.IsAdmin, Order: r.Order, Permissions: perms,
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
		ID:         f.ID,
		Name:       f.Name,
		FieldType:  f.FieldType,
		IsSystem:   f.IsSystem,
		IsRequired: f.IsRequired,
		Options:    f.Options,
	}
}
