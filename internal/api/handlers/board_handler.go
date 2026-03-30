package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type BoardHandler struct {
	service       *services.BoardService
	projectService *services.ProjectService
}

func NewBoardHandler(service *services.BoardService, projectService *services.ProjectService) *BoardHandler {
	return &BoardHandler{service: service, projectService: projectService}
}

func (h *BoardHandler) ListBoards(c *gin.Context) {
	userIDStr := c.GetString("userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
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

	project, err := h.projectService.GetProject(c.Request.Context(), projectID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Проект не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить проект")
		return
	}
	if project.OwnerID != userID {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "Недостаточно прав для работы с досками проекта")
		return
	}
	boards, err := h.service.ListBoards(c.Request.Context(), projectID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить список досок")
		return
	}
	resp := make([]dto.BoardResponse, 0, len(boards))
	for _, b := range boards {
		resp = append(resp, mapBoardToDTO(&b))
	}
	writeSuccess(c, resp)
}

func (h *BoardHandler) CreateBoard(c *gin.Context) {
	var req dto.CreateBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	userIDStr := c.GetString("userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}

	project, err := h.projectService.GetProject(c.Request.Context(), req.ProjectID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Проект не найден")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить проект")
		return
	}
	if project.OwnerID != userID {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "Недостаточно прав для работы с досками проекта")
		return
	}

	swimlaneGroupBy := ""
	if req.SwimlaneGroupBy != nil {
		swimlaneGroupBy = *req.SwimlaneGroupBy
	}
	board, err := h.service.CreateBoard(c.Request.Context(), req.ProjectID, req.Name, req.Description, int16(req.Order), req.PriorityType, req.EstimationUnit, swimlaneGroupBy)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать доску")
		return
	}
	writeSuccess(c, mapBoardToDTO(board))
}

func (h *BoardHandler) GetBoard(c *gin.Context) {
	idStr := c.Param("boardId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	board, err := h.service.GetBoard(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить доску")
		return
	}
	writeSuccess(c, mapBoardToDTO(board))
}

func (h *BoardHandler) UpdateBoard(c *gin.Context) {
	idStr := c.Param("boardId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	var req dto.UpdateBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	board, err := h.service.GetBoard(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Доска не найдена")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить доску")
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
	if req.SwimlaneGroupBy != nil {
		board.SwimlaneGroupBy = *req.SwimlaneGroupBy
	}
	updated, err := h.service.UpdateBoard(c.Request.Context(), board)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить доску")
		return
	}
	writeSuccess(c, mapBoardToDTO(updated))
}

func (h *BoardHandler) DeleteBoard(c *gin.Context) {
	idStr := c.Param("boardId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	if err := h.service.DeleteBoard(c.Request.Context(), id); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить доску")
		return
	}
	writeSuccess(c, gin.H{"message": "Доска удалена"})
}

func (h *BoardHandler) ListColumns(c *gin.Context) {
	boardIDStr := c.Param("boardId")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	cols, err := h.service.ListColumns(c.Request.Context(), boardID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить список колонок")
		return
	}
	resp := make([]dto.ColumnResponse, 0, len(cols))
	for _, col := range cols {
		resp = append(resp, mapColumnToDTO(&col))
	}
	writeSuccess(c, resp)
}

func (h *BoardHandler) CreateColumn(c *gin.Context) {
	boardIDStr := c.Param("boardId")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	var req dto.CreateColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
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
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Недопустимое значение systemType для колонки")
			return
		}
		if err == domain.ErrCompletedColumnWip {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "WIP-лимит нельзя установить для колонок с типом \"Завершено\"")
			return
		}
		if strings.Contains(err.Error(), "INVALID_COLUMN_ORDER") {
			writeError(c, http.StatusBadRequest, "INVALID_COLUMN_ORDER", "Нарушен порядок типов колонок")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать колонку")
		return
	}
	writeSuccess(c, mapColumnToDTO(col))
}

func (h *BoardHandler) ListSwimlanes(c *gin.Context) {
	boardIDStr := c.Param("boardId")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	sw, err := h.service.ListSwimlanes(c.Request.Context(), boardID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить список дорожек")
		return
	}
	resp := make([]dto.SwimlaneResponse, 0, len(sw))
	for _, s := range sw {
		resp = append(resp, mapSwimlaneToDTO(&s))
	}
	writeSuccess(c, resp)
}

func (h *BoardHandler) CreateSwimlane(c *gin.Context) {
	boardIDStr := c.Param("boardId")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	var req dto.CreateSwimlaneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	var swWipLimit16 *int16
	if req.WipLimit != nil {
		v := int16(*req.WipLimit)
		swWipLimit16 = &v
	}
	sw, err := h.service.CreateSwimlane(c.Request.Context(), boardID, req.Name, swWipLimit16, int16(req.Order))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать дорожку")
		return
	}
	writeSuccess(c, mapSwimlaneToDTO(sw))
}

func (h *BoardHandler) ListNotes(c *gin.Context) {
	boardIDStr := c.Param("boardId")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	notes, err := h.service.ListNotes(c.Request.Context(), boardID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить список заметок")
		return
	}
	resp := make([]dto.NoteResponse, 0, len(notes))
	for _, n := range notes {
		resp = append(resp, mapNoteToDTO(&n))
	}
	writeSuccess(c, resp)
}

func (h *BoardHandler) CreateNoteForColumn(c *gin.Context) {
	columnIDStr := c.Param("columnId")
	columnID, err := uuid.Parse(columnIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор колонки")
		return
	}
	var req dto.CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	note, err := h.service.CreateNoteForColumn(c.Request.Context(), columnID, req.Content)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать заметку")
		return
	}
	writeSuccess(c, mapNoteToDTO(note))
}

func (h *BoardHandler) CreateNoteForSwimlane(c *gin.Context) {
	swimlaneIDStr := c.Param("swimlaneId")
	swimlaneID, err := uuid.Parse(swimlaneIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор дорожки")
		return
	}
	var req dto.CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	note, err := h.service.CreateNoteForSwimlane(c.Request.Context(), swimlaneID, req.Content)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать заметку")
		return
	}
	writeSuccess(c, mapNoteToDTO(note))
}

// --- Column PATCH/DELETE/reorder ---

func (h *BoardHandler) UpdateColumn(c *gin.Context) {
	columnID, err := uuid.Parse(c.Param("columnId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор колонки")
		return
	}
	var req dto.UpdateColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
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
		if err == domain.ErrCompletedColumnWip {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "WIP-лимит нельзя установить для колонок с типом \"Завершено\"")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить колонку")
		return
	}
	writeSuccess(c, mapColumnToDTO(updated))
}

func (h *BoardHandler) DeleteColumn(c *gin.Context) {
	columnID, err := uuid.Parse(c.Param("columnId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор колонки")
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
		if err == domain.ErrColumnHasTasks {
			writeError(c, http.StatusBadRequest, "COLUMN_HAS_TASKS", "Нельзя удалить колонку, в которой есть задачи")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить колонку")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *BoardHandler) ReorderColumns(c *gin.Context) {
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	var req dto.BoardReorderColumnsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
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
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось изменить порядок колонок")
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Swimlane PATCH/DELETE/reorder ---

func (h *BoardHandler) UpdateSwimlane(c *gin.Context) {
	swimlaneID, err := uuid.Parse(c.Param("swimlaneId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор дорожки")
		return
	}
	var req dto.UpdateSwimlaneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	sw, err := h.service.GetSwimlaneByID(c.Request.Context(), swimlaneID)
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "Дорожка не найдена")
		return
	}

	// Reject wipLimit for Scrum projects.
	if req.WipLimit.Set && !req.WipLimit.Null {
		board, err := h.service.GetBoard(c.Request.Context(), uuid.MustParse(sw.BoardID))
		if err == nil && board.ProjectID != nil {
			pid, _ := uuid.Parse(*board.ProjectID)
			project, err := h.projectService.GetProject(c.Request.Context(), pid)
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
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить дорожку")
		return
	}
	writeSuccess(c, mapSwimlaneToDTO(updated))
}

func (h *BoardHandler) DeleteSwimlane(c *gin.Context) {
	swimlaneID, err := uuid.Parse(c.Param("swimlaneId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор дорожки")
		return
	}
	err = h.service.DeleteSwimlaneSafe(c.Request.Context(), swimlaneID)
	if err != nil {
		if err == domain.ErrSwimlaneHasTasks {
			writeError(c, http.StatusBadRequest, "SWIMLANE_HAS_TASKS", "Нельзя удалить дорожку, в которой есть задачи")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить дорожку")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *BoardHandler) ReorderSwimlanes(c *gin.Context) {
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	var req dto.BoardReorderSwimlanesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	orders := make(map[uuid.UUID]int16)
	for _, o := range req.Orders {
		orders[o.SwimlaneID] = int16(o.Order)
	}
	if err := h.service.ReorderSwimlanes(c.Request.Context(), boardID, orders); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось изменить порядок дорожек")
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Note PATCH/DELETE ---

func (h *BoardHandler) UpdateNote(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("noteId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор заметки")
		return
	}
	var req dto.UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
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
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить заметку")
		return
	}
	writeSuccess(c, mapNoteToDTO(updated))
}

func (h *BoardHandler) DeleteNote(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("noteId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор заметки")
		return
	}
	if err := h.service.DeleteNote(c.Request.Context(), noteID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить заметку")
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Board reorder ---

func (h *BoardHandler) ReorderBoards(c *gin.Context) {
	var req dto.ReorderBoardsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	orders := make(map[uuid.UUID]int16)
	for _, o := range req.Orders {
		orders[o.BoardID] = int16(o.Order)
	}
	if err := h.service.ReorderBoards(c.Request.Context(), orders); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось изменить порядок досок")
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Custom Fields ---

func (h *BoardHandler) ListCustomFields(c *gin.Context) {
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	fields, err := h.service.ListCustomFields(c.Request.Context(), boardID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить кастомные поля")
		return
	}
	resp := make([]dto.BoardCustomFieldResponse, 0, len(fields))
	for _, f := range fields {
		resp = append(resp, mapCustomFieldToDTO(f))
	}
	writeSuccess(c, resp)
}

func (h *BoardHandler) CreateCustomField(c *gin.Context) {
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	var req dto.CreateBoardCustomFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	field, err := h.service.CreateCustomField(c.Request.Context(), boardID, req.Name, req.FieldType, req.IsRequired, req.Options)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать кастомное поле")
		return
	}
	writeSuccess(c, mapCustomFieldToDTO(*field))
}

func (h *BoardHandler) UpdateCustomField(c *gin.Context) {
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	fieldID, err := uuid.Parse(c.Param("fieldId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор поля")
		return
	}
	var req dto.UpdateBoardCustomFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	var desc *string
	if req.Description.Set {
		if req.Description.Null {
			empty := ""
			desc = &empty
		} else {
			desc = &req.Description.Value
		}
	}
	field, err := h.service.UpdateCustomField(c.Request.Context(), boardID, fieldID, req.Name, desc, req.IsRequired, req.Options)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Поле не найдено")
			return
		}
		if err == domain.ErrSystemField {
			writeError(c, http.StatusBadRequest, "SYSTEM_FIELD", "Нельзя изменять системное поле")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить кастомное поле")
		return
	}
	writeSuccess(c, mapCustomFieldToDTO(*field))
}

func (h *BoardHandler) DeleteCustomField(c *gin.Context) {
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	fieldID, err := uuid.Parse(c.Param("fieldId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор поля")
		return
	}
	err = h.service.DeleteCustomField(c.Request.Context(), boardID, fieldID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Поле не найдено")
			return
		}
		if err == domain.ErrSystemField {
			writeError(c, http.StatusBadRequest, "SYSTEM_FIELD", "Нельзя удалить системное поле")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить кастомное поле")
		return
	}
	c.Status(http.StatusNoContent)
}

// --- DTO mappers ---

func mapBoardToDTO(b *domain.Board) dto.BoardResponse {
	var projectID *uuid.UUID
	if b.ProjectID != nil {
		if id, err := uuid.Parse(*b.ProjectID); err == nil {
			projectID = &id
		}
	}
	var sgb *string
	if b.SwimlaneGroupBy != "" {
		sgb = &b.SwimlaneGroupBy
	}
	return dto.BoardResponse{
		ID:              uuid.MustParse(b.ID),
		ProjectID:       projectID,
		Name:            b.Name,
		Description:     b.Description,
		IsDefault:       b.IsDefault,
		Order:           int32(b.Order),
		PriorityType:    b.PriorityType,
		EstimationUnit:  b.EstimationUnit,
		SwimlaneGroupBy: sgb,
	}
}

func mapCustomFieldToDTO(f domain.BoardCustomField) dto.BoardCustomFieldResponse {
	opts := f.Options
	if opts == nil {
		opts = []string{}
	}
	return dto.BoardCustomFieldResponse{
		ID:          uuid.MustParse(f.ID),
		Name:        f.Name,
		Description: f.Description,
		FieldType:   f.FieldType,
		IsSystem:    f.IsSystem,
		IsRequired:  f.IsRequired,
		Options:     opts,
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
		ID:         uuid.MustParse(c.ID),
		BoardID:    uuid.MustParse(c.BoardID),
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
		ID:       uuid.MustParse(s.ID),
		BoardID:  uuid.MustParse(s.BoardID),
		Name:     s.Name,
		WipLimit: wl,
		Order:    int32(s.Order),
	}
}

func mapNoteToDTO(n *domain.Note) dto.NoteResponse {
	var colID *uuid.UUID
	if n.ColumnID != nil {
		if id, err := uuid.Parse(*n.ColumnID); err == nil {
			colID = &id
		}
	}
	var swID *uuid.UUID
	if n.SwimlaneID != nil {
		if id, err := uuid.Parse(*n.SwimlaneID); err == nil {
			swID = &id
		}
	}
	return dto.NoteResponse{
		ID:         uuid.MustParse(n.ID),
		ColumnID:   colID,
		SwimlaneID: swID,
		Content:    n.Content,
	}
}

