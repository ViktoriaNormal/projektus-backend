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

type ClassOfServiceHandler struct {
	service *services.ClassOfServiceService
}

func NewClassOfServiceHandler(service *services.ClassOfServiceService) *ClassOfServiceHandler {
	return &ClassOfServiceHandler{service: service}
}

// GetClassesOfService возвращает список доступных классов обслуживания.
func (h *ClassOfServiceHandler) GetClassesOfService(c *gin.Context) {
	_ = c.Param("projectId") // пока используется только для маршрутизации

	classes := h.service.GetDefaultClasses()
	resp := make([]dto.ClassOfServiceResponse, 0, len(classes))
	for _, class := range classes {
		value := string(class)
		name := strings.Title(strings.ReplaceAll(value, "_", " "))
		resp = append(resp, dto.ClassOfServiceResponse{
			Value:       value,
			Name:        name,
			Description: "",
		})
	}
	writeSuccess(c, resp)
}

// UpdateTaskClass обновляет класс обслуживания задачи.
func (h *ClassOfServiceHandler) UpdateTaskClass(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}

	var req dto.UpdateTaskClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	if err := h.service.SetTaskClass(c.Request.Context(), taskID, req.ClassOfService); err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Недопустимый класс обслуживания")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить класс обслуживания задачи")
		return
	}

	writeSuccess(c, gin.H{"message": "Класс обслуживания задачи обновлён"})
}

// ConfigureSwimlanes настраивает источник дорожек для доски.
func (h *ClassOfServiceHandler) ConfigureSwimlanes(c *gin.Context) {
	boardIDStr := c.Param("boardId")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}

	var req dto.SwimlaneConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	if err := h.service.ConfigureSwimlanes(c.Request.Context(), boardID, req.SourceType, req.CustomFieldID, req.ValueMappings); err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректная конфигурация дорожек")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить конфигурацию дорожек")
		return
	}

	writeSuccess(c, gin.H{"message": "Конфигурация дорожек обновлена"})
}

