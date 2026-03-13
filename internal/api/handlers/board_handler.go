package handlers

import (
	"net/http"

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

	board, err := h.service.CreateBoard(c.Request.Context(), req.ProjectID, req.Name, req.Description, req.Order)
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
	if req.Description != nil {
		board.Description = req.Description
	}
	if req.Order != nil {
		board.Order = *req.Order
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
	col, err := h.service.CreateColumn(c.Request.Context(), boardID, req.Name, sysType, req.WipLimit, req.Order)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Недопустимое значение systemType для колонки")
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
	sw, err := h.service.CreateSwimlane(c.Request.Context(), boardID, req.Name, req.WipLimit, req.Order)
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

func mapBoardToDTO(b *domain.Board) dto.BoardResponse {
	var projectID *uuid.UUID
	if b.ProjectID != nil {
		if id, err := uuid.Parse(*b.ProjectID); err == nil {
			projectID = &id
		}
	}
	return dto.BoardResponse{
		ID:          uuid.MustParse(b.ID),
		ProjectID:   projectID,
		Name:        b.Name,
		Description: b.Description,
		Order:       b.Order,
	}
}

func mapColumnToDTO(c *domain.Column) dto.ColumnResponse {
	var st *string
	if c.SystemType != nil {
		s := string(*c.SystemType)
		st = &s
	}
	return dto.ColumnResponse{
		ID:         uuid.MustParse(c.ID),
		BoardID:    uuid.MustParse(c.BoardID),
		Name:       c.Name,
		SystemType: st,
		WipLimit:   c.WipLimit,
		Order:      c.Order,
	}
}

func mapSwimlaneToDTO(s *domain.Swimlane) dto.SwimlaneResponse {
	return dto.SwimlaneResponse{
		ID:       uuid.MustParse(s.ID),
		BoardID:  uuid.MustParse(s.BoardID),
		Name:     s.Name,
		WipLimit: s.WipLimit,
		Order:    s.Order,
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

