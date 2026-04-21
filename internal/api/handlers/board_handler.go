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

type BoardHandler struct {
	service        *services.BoardService
	projectService *services.ProjectService
	permissionSvc  *services.PermissionService
}

func NewBoardHandler(service *services.BoardService, projectService *services.ProjectService, permissionSvc *services.PermissionService) *BoardHandler {
	return &BoardHandler{service: service, projectService: projectService, permissionSvc: permissionSvc}
}

func (h *BoardHandler) ListBoards(c *gin.Context) {
	userID, ok := requireUserUUID(c)
	if !ok {
		return
	}

	projectIDStr := c.Query("projectId")
	if projectIDStr == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Необходимо указать projectId")
		return
	}
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	perms, _ := h.permissionSvc.GetMyPermissions(c.Request.Context(), userID, projectID)
	hasAccess := false
	for _, p := range perms {
		if p.Area == "project.boards" && p.Access != "none" {
			hasAccess = true
			break
		}
	}
	if !hasAccess {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "Недостаточно прав для работы с досками проекта")
		return
	}
	boards, err := h.service.ListBoards(c.Request.Context(), projectID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить список досок")
		return
	}
	resp := make([]dto.BoardResponse, 0, len(boards))
	for _, b := range boards {
		resp = append(resp, mapBoardToDTO(&b))
	}
	writeSuccess(c, resp)
}

func (h *BoardHandler) CreateBoard(c *gin.Context) {
	req, ok := bindJSON[dto.CreateBoardRequest](c)
	if !ok {
		return
	}

	userID, ok := requireUserUUID(c)
	if !ok {
		return
	}

	perms, _ := h.permissionSvc.GetMyPermissions(c.Request.Context(), userID, req.ProjectID)
	hasAccess := false
	for _, p := range perms {
		if p.Area == "project.boards" && p.Access == "full" {
			hasAccess = true
			break
		}
	}
	if !hasAccess {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "Недостаточно прав для работы с досками проекта")
		return
	}

	project, err := h.projectService.GetProject(c.Request.Context(), req.ProjectID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить проект")
		return
	}

	swimlaneGroupBy := ""
	if req.SwimlaneGroupBy != nil {
		swimlaneGroupBy = *req.SwimlaneGroupBy
	}
	board, err := h.service.CreateBoard(c.Request.Context(), req.ProjectID, string(project.Type), req.Name, req.Description, int16(req.Order), req.PriorityType, req.EstimationUnit, swimlaneGroupBy)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать доску")
		return
	}
	writeSuccess(c, mapBoardToDTO(board))
}

func (h *BoardHandler) GetBoard(c *gin.Context) {
	id, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	board, err := h.service.GetBoard(c.Request.Context(), id)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить доску")
		return
	}
	writeSuccess(c, mapBoardToDTO(board))
}

func (h *BoardHandler) UpdateBoard(c *gin.Context) {
	id, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.UpdateBoardRequest](c)
	if !ok {
		return
	}
	board, err := h.service.GetBoard(c.Request.Context(), id)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить доску")
		return
	}
	if req.Name != nil {
		board.Name = *req.Name
	}
	if req.Description.Set {
		board.Description = req.Description.Ptr()
	}
	if req.IsDefault != nil {
		board.IsDefault = *req.IsDefault
	}
	if req.Order != nil {
		board.Order = int16(*req.Order)
	}
	if req.PriorityType != nil {
		board.PriorityType = *req.PriorityType
	}
	if req.EstimationUnit != nil {
		board.EstimationUnit = *req.EstimationUnit
	}
	if req.SwimlaneGroupBy.Set {
		if req.SwimlaneGroupBy.Null {
			board.SwimlaneGroupBy = ""
		} else {
			board.SwimlaneGroupBy = req.SwimlaneGroupBy.Value
		}
	}
	if req.PriorityOptions != nil {
		board.PriorityOptions = req.PriorityOptions
	}
	updated, err := h.service.UpdateBoard(c.Request.Context(), board)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить доску")
		return
	}
	writeSuccess(c, mapBoardToDTO(updated))
}

func (h *BoardHandler) DeleteBoard(c *gin.Context) {
	id, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	if err := h.service.DeleteBoard(c.Request.Context(), id); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить доску")
		return
	}
	writeSuccess(c, gin.H{"message": "Доска удалена"})
}

func (h *BoardHandler) ListColumns(c *gin.Context) {
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	cols, err := h.service.ListColumns(c.Request.Context(), boardID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить список колонок")
		return
	}
	resp := make([]dto.ColumnResponse, 0, len(cols))
	for _, col := range cols {
		resp = append(resp, mapColumnToDTO(&col))
	}
	writeSuccess(c, resp)
}

func (h *BoardHandler) CreateColumn(c *gin.Context) {
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.CreateColumnRequest](c)
	if !ok {
		return
	}
	if req.SystemType == nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "При создании колонки необходимо указать systemType")
		return
	}
	var sysType *domain.SystemStatusType
	if req.SystemType != nil {
		st := domain.SystemStatusType(*req.SystemType)
		sysType = &st
	}
	var wipLimit16 *int16
	if req.WipLimit != nil {
		v := int16(*req.WipLimit)
		wipLimit16 = &v
	}
	col, err := h.service.CreateColumn(c.Request.Context(), boardID, req.Name, sysType, wipLimit16, int16(req.Order))
	if err != nil {
		// Специфичный текстовый маркер — до домен-маппинга
		if strings.Contains(err.Error(), "INVALID_COLUMN_ORDER") {
			writeError(c, http.StatusBadRequest, "INVALID_COLUMN_ORDER", "Нарушен порядок типов колонок")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать колонку")
		return
	}
	writeSuccess(c, mapColumnToDTO(col))
}

func (h *BoardHandler) ListSwimlanes(c *gin.Context) {
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	sw, err := h.service.ListSwimlanes(c.Request.Context(), boardID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить список дорожек")
		return
	}
	resp := make([]dto.SwimlaneResponse, 0, len(sw))
	for _, s := range sw {
		resp = append(resp, mapSwimlaneToDTO(&s))
	}
	writeSuccess(c, resp)
}

func (h *BoardHandler) CreateSwimlane(c *gin.Context) {
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.CreateSwimlaneRequest](c)
	if !ok {
		return
	}
	var swWipLimit16 *int16
	if req.WipLimit != nil {
		v := int16(*req.WipLimit)
		swWipLimit16 = &v
	}
	sw, err := h.service.CreateSwimlane(c.Request.Context(), boardID, req.Name, swWipLimit16, int16(req.Order))
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать дорожку")
		return
	}
	writeSuccess(c, mapSwimlaneToDTO(sw))
}

func (h *BoardHandler) ListNotes(c *gin.Context) {
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	notes, err := h.service.ListNotes(c.Request.Context(), boardID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить список заметок")
		return
	}
	resp := make([]dto.NoteResponse, 0, len(notes))
	for _, n := range notes {
		resp = append(resp, mapNoteToDTO(&n))
	}
	writeSuccess(c, resp)
}

func (h *BoardHandler) CreateNoteForColumn(c *gin.Context) {
	columnID, ok := paramUUID(c, "columnId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.CreateNoteRequest](c)
	if !ok {
		return
	}
	note, err := h.service.CreateNoteForColumn(c.Request.Context(), columnID, req.Content)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать заметку")
		return
	}
	writeSuccess(c, mapNoteToDTO(note))
}

func (h *BoardHandler) CreateNoteForSwimlane(c *gin.Context) {
	swimlaneID, ok := paramUUID(c, "swimlaneId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.CreateNoteRequest](c)
	if !ok {
		return
	}
	note, err := h.service.CreateNoteForSwimlane(c.Request.Context(), swimlaneID, req.Content)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать заметку")
		return
	}
	writeSuccess(c, mapNoteToDTO(note))
}

// --- Column PATCH/DELETE/reorder ---

func (h *BoardHandler) UpdateColumn(c *gin.Context) {
	columnID, ok := paramUUID(c, "columnId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.UpdateColumnRequest](c)
	if !ok {
		return
	}

	col, err := h.service.GetColumnByID(c.Request.Context(), columnID)
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "Колонка не найдена")
		return
	}
	if col.IsLocked {
		if req.Name != nil || req.SystemType.Set {
			writeError(c, http.StatusBadRequest, "COLUMN_LOCKED", "Заблокированную колонку нельзя переименовывать или менять systemType. Допускается изменение wipLimit")
			return
		}
	}
	if req.Name != nil {
		col.Name = *req.Name
	}
	if req.SystemType.Set {
		if req.SystemType.Null {
			col.SystemType = nil
		} else {
			st := domain.SystemStatusType(req.SystemType.Value)
			col.SystemType = &st
		}
	}
	if req.WipLimit.Set {
		if req.WipLimit.Null {
			col.WipLimit = nil
		} else {
			v := int16(req.WipLimit.Value)
			col.WipLimit = &v
		}
	}
	if req.Order != nil {
		col.Order = int16(*req.Order)
	}
	updated, err := h.service.UpdateColumn(c.Request.Context(), col)
	if err != nil {
		if strings.Contains(err.Error(), "INVALID_COLUMN_ORDER") {
			writeError(c, http.StatusBadRequest, "INVALID_COLUMN_ORDER", "Нарушен порядок типов колонок")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить колонку")
		return
	}
	writeSuccess(c, mapColumnToDTO(updated))
}

func (h *BoardHandler) DeleteColumn(c *gin.Context) {
	columnID, ok := paramUUID(c, "columnId")
	if !ok {
		return
	}
	col, err := h.service.GetColumnByID(c.Request.Context(), columnID)
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "Колонка не найдена")
		return
	}
	if col.IsLocked {
		writeError(c, http.StatusBadRequest, "COLUMN_LOCKED", "Нельзя удалить заблокированную колонку")
		return
	}
	err = h.service.DeleteColumnSafe(c.Request.Context(), columnID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить колонку")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *BoardHandler) ReorderColumns(c *gin.Context) {
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.BoardReorderColumnsRequest](c)
	if !ok {
		return
	}
	orders := make(map[uuid.UUID]int16)
	for _, o := range req.Orders {
		orders[o.ColumnID] = int16(o.Order)
	}
	if err := h.service.ReorderColumns(c.Request.Context(), boardID, orders); err != nil {
		if strings.Contains(err.Error(), "COLUMN_LOCKED") {
			writeError(c, http.StatusBadRequest, "COLUMN_LOCKED", "Нельзя перемещать заблокированную колонку")
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
	c.Status(http.StatusNoContent)
}

// --- Swimlane PATCH/DELETE/reorder ---

func (h *BoardHandler) UpdateSwimlane(c *gin.Context) {
	swimlaneID, ok := paramUUID(c, "swimlaneId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.UpdateSwimlaneRequest](c)
	if !ok {
		return
	}

	sw, err := h.service.GetSwimlaneByID(c.Request.Context(), swimlaneID)
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "Дорожка не найдена")
		return
	}

	// Reject wipLimit for Scrum projects.
	if req.WipLimit.Set && !req.WipLimit.Null {
		board, err := h.service.GetBoard(c.Request.Context(), sw.BoardID)
		if err == nil && board.ProjectID != nil {
			project, err := h.projectService.GetProject(c.Request.Context(), *board.ProjectID)
			if err == nil && project.Type == domain.ProjectTypeScrum {
				writeError(c, http.StatusBadRequest, "SCRUM_WIP_NOT_ALLOWED", "WIP-лимиты дорожек не поддерживаются в Scrum")
				return
			}
		}
	}

	if req.Name != nil {
		sw.Name = *req.Name
	}
	if req.WipLimit.Set {
		if req.WipLimit.Null {
			sw.WipLimit = nil
		} else {
			v := int16(req.WipLimit.Value)
			sw.WipLimit = &v
		}
	}
	if req.Order != nil {
		sw.Order = int16(*req.Order)
	}
	updated, err := h.service.UpdateSwimlane(c.Request.Context(), sw)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить дорожку")
		return
	}
	writeSuccess(c, mapSwimlaneToDTO(updated))
}

func (h *BoardHandler) DeleteSwimlane(c *gin.Context) {
	swimlaneID, ok := paramUUID(c, "swimlaneId")
	if !ok {
		return
	}
	err := h.service.DeleteSwimlaneSafe(c.Request.Context(), swimlaneID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить дорожку")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *BoardHandler) ReorderSwimlanes(c *gin.Context) {
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.BoardReorderSwimlanesRequest](c)
	if !ok {
		return
	}
	orders := make(map[uuid.UUID]int16)
	for _, o := range req.Orders {
		orders[o.SwimlaneID] = int16(o.Order)
	}
	if err := h.service.ReorderSwimlanes(c.Request.Context(), boardID, orders); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось изменить порядок дорожек")
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Note PATCH/DELETE ---

func (h *BoardHandler) UpdateNote(c *gin.Context) {
	noteID, ok := paramUUID(c, "noteId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.UpdateNoteRequest](c)
	if !ok {
		return
	}
	note, err := h.service.GetNoteByID(c.Request.Context(), noteID)
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "Заметка не найдена")
		return
	}
	if req.Content != nil {
		note.Content = *req.Content
	}
	updated, err := h.service.UpdateNote(c.Request.Context(), note)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить заметку")
		return
	}
	writeSuccess(c, mapNoteToDTO(updated))
}

func (h *BoardHandler) DeleteNote(c *gin.Context) {
	noteID, ok := paramUUID(c, "noteId")
	if !ok {
		return
	}
	if err := h.service.DeleteNote(c.Request.Context(), noteID); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить заметку")
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Board reorder ---

func (h *BoardHandler) ReorderBoards(c *gin.Context) {
	req, ok := bindJSON[dto.ReorderBoardsRequest](c)
	if !ok {
		return
	}
	orders := make(map[uuid.UUID]int16)
	for _, o := range req.Orders {
		orders[o.BoardID] = int16(o.Order)
	}
	if err := h.service.ReorderBoards(c.Request.Context(), orders); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось изменить порядок досок")
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Custom Fields ---

func (h *BoardHandler) ListCustomFields(c *gin.Context) {
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}

	// Get board to determine project type and priority options.
	board, err := h.service.GetBoard(c.Request.Context(), boardID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить доску")
		return
	}
	projectType := "kanban"
	if board.ProjectID != nil {
		project, err := h.projectService.GetProject(c.Request.Context(), *board.ProjectID)
		if err == nil {
			projectType = string(project.Type)
		}
	}

	// Generate system fields from constants.
	systemFields := domain.GenerateSystemBoardFields(
		projectType, board.PriorityType, board.EstimationUnit,
		board.PriorityOptions, repositories.DefaultBoardFields,
	)

	// Get custom fields from DB.
	customFields, err := h.service.ListCustomFields(c.Request.Context(), boardID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить кастомные поля")
		return
	}

	// Merge: system first, then custom.
	allFields := append(systemFields, customFields...)
	resp := make([]dto.BoardCustomFieldResponse, 0, len(allFields))
	for _, f := range allFields {
		resp = append(resp, mapCustomFieldToDTO(f))
	}
	writeSuccess(c, resp)
}

func (h *BoardHandler) CreateCustomField(c *gin.Context) {
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.CreateBoardCustomFieldRequest](c)
	if !ok {
		return
	}
	field, err := h.service.CreateCustomField(c.Request.Context(), boardID, req.Name, req.FieldType, req.IsRequired, req.Options)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать кастомное поле")
		return
	}
	writeSuccess(c, mapCustomFieldToDTO(*field))
}

func (h *BoardHandler) UpdateCustomField(c *gin.Context) {
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	fieldID, ok := paramUUID(c, "fieldId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.UpdateBoardCustomFieldRequest](c)
	if !ok {
		return
	}
	field, err := h.service.UpdateCustomField(c.Request.Context(), boardID, fieldID, req.Name, req.IsRequired, req.Options)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить кастомное поле")
		return
	}
	writeSuccess(c, mapCustomFieldToDTO(*field))
}

func (h *BoardHandler) DeleteCustomField(c *gin.Context) {
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	fieldID, ok := paramUUID(c, "fieldId")
	if !ok {
		return
	}
	err := h.service.DeleteCustomField(c.Request.Context(), boardID, fieldID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить кастомное поле")
		return
	}
	c.Status(http.StatusNoContent)
}

// --- DTO mappers ---

func mapBoardToDTO(b *domain.Board) dto.BoardResponse {
	var projectID *uuid.UUID
	if b.ProjectID != nil {
		id := *b.ProjectID
		projectID = &id
	}
	var sgb *string
	if b.SwimlaneGroupBy != "" {
		sgb = &b.SwimlaneGroupBy
	}
	priorityOptions := b.PriorityOptions
	if priorityOptions == nil {
		priorityOptions = []string{}
	}
	return dto.BoardResponse{
		ID:              b.ID,
		ProjectID:       projectID,
		Name:            b.Name,
		Description:     b.Description,
		IsDefault:       b.IsDefault,
		Order:           int32(b.Order),
		PriorityType:    b.PriorityType,
		EstimationUnit:  b.EstimationUnit,
		SwimlaneGroupBy: sgb,
		PriorityOptions: priorityOptions,
	}
}

func mapCustomFieldToDTO(f domain.BoardCustomField) dto.BoardCustomFieldResponse {
	opts := f.Options
	if opts == nil {
		opts = []string{}
	}
	return dto.BoardCustomFieldResponse{
		ID:         f.ID,
		Name:       f.Name,
		FieldType:  f.FieldType,
		IsSystem:   f.IsSystem,
		IsRequired: f.IsRequired,
		Options:    opts,
	}
}

func mapColumnToDTO(c *domain.Column) dto.ColumnResponse {
	var st *string
	if c.SystemType != nil {
		s := string(*c.SystemType)
		st = &s
	}
	var wl *int32
	if c.WipLimit != nil {
		v := int32(*c.WipLimit)
		wl = &v
	}
	return dto.ColumnResponse{
		ID:         c.ID,
		BoardID:    c.BoardID,
		Name:       c.Name,
		SystemType: st,
		WipLimit:   wl,
		Order:      int32(c.Order),
		IsLocked:   c.IsLocked,
	}
}

func mapSwimlaneToDTO(s *domain.Swimlane) dto.SwimlaneResponse {
	var wl *int32
	if s.WipLimit != nil {
		v := int32(*s.WipLimit)
		wl = &v
	}
	return dto.SwimlaneResponse{
		ID:       s.ID,
		BoardID:  s.BoardID,
		Name:     s.Name,
		WipLimit: wl,
		Order:    int32(s.Order),
	}
}

func mapNoteToDTO(n *domain.Note) dto.NoteResponse {
	var colID *uuid.UUID
	if n.ColumnID != nil {
		id := *n.ColumnID
		colID = &id
	}
	var swID *uuid.UUID
	if n.SwimlaneID != nil {
		id := *n.SwimlaneID
		swID = &id
	}
	return dto.NoteResponse{
		ID:         n.ID,
		ColumnID:   colID,
		SwimlaneID: swID,
		Content:    n.Content,
	}
}
