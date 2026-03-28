package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type TagHandler struct {
	service *services.TagService
}

func NewTagHandler(service *services.TagService) *TagHandler {
	return &TagHandler{service: service}
}

// GET /boards/:boardId/tags — список тегов доски (для автокомплита)
func (h *TagHandler) ListBoardTags(c *gin.Context) {
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}

	tags, err := h.service.ListBoardTags(c.Request.Context(), boardID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить теги")
		return
	}

	writeSuccess(c, mapTagsToDTO(tags))
}

// GET /tasks/:taskId/tags — теги задачи
func (h *TagHandler) ListTaskTags(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("taskId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}

	tags, err := h.service.ListTaskTags(c.Request.Context(), taskID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить теги задачи")
		return
	}

	writeSuccess(c, mapTagsToDTO(tags))
}

// POST /boards/:boardId/tasks/:taskId/tags — добавить тег к задаче
func (h *TagHandler) AddTagToTask(c *gin.Context) {
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	taskID, err := uuid.Parse(c.Param("taskId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}

	var req dto.AddTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	tag, err := h.service.AddTagToTask(c.Request.Context(), boardID, taskID, req.Name)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Имя тега не может быть пустым")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось добавить тег")
		return
	}

	writeSuccess(c, mapTagToDTO(*tag))
}

// DELETE /tasks/:taskId/tags/:tagId — убрать тег с задачи
func (h *TagHandler) RemoveTagFromTask(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("taskId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	tagID, err := uuid.Parse(c.Param("tagId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор тега")
		return
	}

	if err := h.service.RemoveTagFromTask(c.Request.Context(), taskID, tagID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить тег")
		return
	}

	c.Status(http.StatusNoContent)
}

// PUT /boards/:boardId/tasks/:taskId/tags — заменить все теги задачи
func (h *TagHandler) SetTaskTags(c *gin.Context) {
	boardID, err := uuid.Parse(c.Param("boardId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}
	taskID, err := uuid.Parse(c.Param("taskId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}

	var req dto.SetTaskTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	tags, err := h.service.SetTaskTags(c.Request.Context(), boardID, taskID, req.Tags)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить теги")
		return
	}

	writeSuccess(c, mapTagsToDTO(tags))
}

func mapTagToDTO(t domain.Tag) dto.TagResponse {
	return dto.TagResponse{ID: t.ID, BoardID: t.BoardID, Name: t.Name}
}

func mapTagsToDTO(tags []domain.Tag) []dto.TagResponse {
	result := make([]dto.TagResponse, len(tags))
	for i, t := range tags {
		result[i] = mapTagToDTO(t)
	}
	return result
}
