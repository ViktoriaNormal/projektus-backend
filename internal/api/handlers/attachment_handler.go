package handlers

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type AttachmentHandler struct {
	service *services.AttachmentService
}

func NewAttachmentHandler(service *services.AttachmentService) *AttachmentHandler {
	return &AttachmentHandler{service: service}
}

func (h *AttachmentHandler) UploadTaskAttachment(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Файл не найден в запросе")
		return
	}

	if file.Size > 10*1024*1024 {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Файл слишком большой (максимум 10 МБ)")
		return
	}

	ext := filepath.Ext(file.Filename)
	if ext == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Недопустимый тип файла")
		return
	}

	userIDStr := c.GetString("userID")
	uploadedBy, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return
	}

	// Для примера сохраняем в ./uploads.
	savePath := filepath.Join("uploads", uuid.New().String()+ext)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось сохранить файл")
		return
	}

	att, err := h.service.CreateTaskAttachment(c.Request.Context(), taskID, file.Filename, savePath, uploadedBy)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Недопустимый тип файла")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать вложение")
		return
	}

	writeSuccess(c, mapAttachmentToDTO(att))
}

func (h *AttachmentHandler) ListTaskAttachments(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}

	attachments, err := h.service.ListTaskAttachments(c.Request.Context(), taskID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить список вложений")
		return
	}

	resp := make([]dto.AttachmentResponse, 0, len(attachments))
	for _, a := range attachments {
		resp = append(resp, mapAttachmentToDTO(&a))
	}
	writeSuccess(c, resp)
}

func mapAttachmentToDTO(a *domain.Attachment) dto.AttachmentResponse {
	var taskID *uuid.UUID
	if a.TaskID != nil {
		if id, err := uuid.Parse(*a.TaskID); err == nil {
			taskID = &id
		}
	}
	var commentID *uuid.UUID
	if a.CommentID != nil {
		if id, err := uuid.Parse(*a.CommentID); err == nil {
			commentID = &id
		}
	}
	return dto.AttachmentResponse{
		ID:         uuid.MustParse(a.ID),
		TaskID:     taskID,
		CommentID:  commentID,
		FileName:   a.FileName,
		FilePath:   a.FilePath,
		UploadedBy: uuid.MustParse(a.UploadedBy),
	}
}

