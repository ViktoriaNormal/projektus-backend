package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

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
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	tags, err := h.service.ListBoardTags(c.Request.Context(), boardID)
	if err != nil {
		respondInternal(c, err, "Не удалось получить теги")
		return
	}
	writeSuccess(c, mapTagsToDTO(tags))
}

// GET /tasks/:taskId/tags — теги задачи
func (h *TagHandler) ListTaskTags(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	tags, err := h.service.ListTaskTags(c.Request.Context(), taskID)
	if err != nil {
		respondInternal(c, err, "Не удалось получить теги задачи")
		return
	}
	writeSuccess(c, mapTagsToDTO(tags))
}

// POST /boards/:boardId/tasks/:taskId/tags — добавить тег к задаче
func (h *TagHandler) AddTagToTask(c *gin.Context) {
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.AddTagRequest](c)
	if !ok {
		return
	}

	tag, err := h.service.AddTagToTask(c.Request.Context(), boardID, taskID, req.Name)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось добавить тег")
		return
	}
	writeSuccess(c, mapTagToDTO(*tag))
}

// DELETE /tasks/:taskId/tags/:tagId — убрать тег с задачи
func (h *TagHandler) RemoveTagFromTask(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	tagID, ok := paramUUID(c, "tagId")
	if !ok {
		return
	}
	if err := h.service.RemoveTagFromTask(c.Request.Context(), taskID, tagID); err != nil {
		respondInternal(c, err, "Не удалось удалить тег")
		return
	}
	c.Status(http.StatusNoContent)
}

// PUT /boards/:boardId/tasks/:taskId/tags — заменить все теги задачи
func (h *TagHandler) SetTaskTags(c *gin.Context) {
	boardID, ok := paramUUID(c, "boardId")
	if !ok {
		return
	}
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.SetTaskTagsRequest](c)
	if !ok {
		return
	}

	tags, err := h.service.SetTaskTags(c.Request.Context(), boardID, taskID, req.Tags)
	if err != nil {
		respondInternal(c, err, "Не удалось обновить теги")
		return
	}
	writeSuccess(c, mapTagsToDTO(tags))
}

func mapTagToDTO(t domain.Tag) dto.TagResponse {
	return dto.TagResponse{ID: t.ID.String(), BoardID: t.BoardID.String(), Name: t.Name}
}

func mapTagsToDTO(tags []domain.Tag) []dto.TagResponse {
	result := make([]dto.TagResponse, len(tags))
	for i, t := range tags {
		result[i] = mapTagToDTO(t)
	}
	return result
}
