package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/services"
)

type ProductBacklogHandler struct {
	service *services.ProductBacklogService
}

func NewProductBacklogHandler(service *services.ProductBacklogService) *ProductBacklogHandler {
	return &ProductBacklogHandler{service: service}
}

func (h *ProductBacklogHandler) GetProductBacklog(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}
	tasks, err := h.service.GetProductBacklog(c.Request.Context(), projectID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить бэклог продукта")
		return
	}
	resp := make([]dto.TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, mapTaskToDTO(&t))
	}
	writeSuccess(c, resp)
}

type reorderRequest struct {
	Orders []dto.TaskOrder `json:"orders" binding:"required"`
}

func (h *ProductBacklogHandler) ReorderProductBacklog(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}
	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	orders := make(map[uuid.UUID]int32, len(req.Orders))
	for _, o := range req.Orders {
		orders[o.TaskID] = int32(o.Order)
	}
	if err := h.service.ReorderProductBacklog(c.Request.Context(), projectID, orders); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось изменить порядок бэклога продукта")
		return
	}
	writeSuccess(c, gin.H{"message": "Порядок обновлён"})
}

func (h *ProductBacklogHandler) AddTaskToBacklog(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}
	var body struct {
		TaskID uuid.UUID `json:"taskId" binding:"required"`
		Order  int32     `json:"order"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	if err := h.service.AddToProductBacklog(c.Request.Context(), projectID, body.TaskID, body.Order); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось добавить задачу в бэклог продукта")
		return
	}
	writeSuccess(c, gin.H{"message": "Задача добавлена в бэклог продукта"})
}

func (h *ProductBacklogHandler) RemoveTaskFromBacklog(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	if err := h.service.RemoveFromProductBacklog(c.Request.Context(), projectID, taskID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить задачу из бэклога продукта")
		return
	}
	writeSuccess(c, gin.H{"message": "Задача удалена из бэклога продукта"})
}

