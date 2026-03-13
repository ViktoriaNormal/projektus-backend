package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type CommentHandler struct {
	service *services.CommentService
}

func NewCommentHandler(service *services.CommentService) *CommentHandler {
	return &CommentHandler{service: service}
}

func (h *CommentHandler) ListTaskComments(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	comments, err := h.service.ListTaskComments(c.Request.Context(), taskID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить список комментариев")
		return
	}
	resp := make([]dto.CommentResponse, 0, len(comments))
	for _, cm := range comments {
		resp = append(resp, mapCommentToDTO(&cm))
	}
	writeSuccess(c, resp)
}

func (h *CommentHandler) CreateComment(c *gin.Context) {
	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	comment, err := h.service.CreateComment(c.Request.Context(), req.TaskID, req.AuthorMemberID, req.Content)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный текст комментария")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать комментарий")
		return
	}
	writeSuccess(c, mapCommentToDTO(comment))
}

func mapCommentToDTO(cmt *domain.Comment) dto.CommentResponse {
	return dto.CommentResponse{
		ID:        uuid.MustParse(cmt.ID),
		TaskID:    uuid.MustParse(cmt.TaskID),
		AuthorID:  uuid.MustParse(cmt.AuthorID),
		Content:   cmt.Content,
		CreatedAt: cmt.CreatedAt,
		UpdatedAt: cmt.UpdatedAt,
	}
}

