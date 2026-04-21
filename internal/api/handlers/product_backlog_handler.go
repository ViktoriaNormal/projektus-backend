package handlers

import (
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
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}
	tasks, err := h.service.GetProductBacklog(c.Request.Context(), projectID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить бэклог продукта")
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
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}
	req, ok := bindJSON[reorderRequest](c)
	if !ok {
		return
	}
	orders := make(map[uuid.UUID]int32, len(req.Orders))
	for _, o := range req.Orders {
		orders[o.TaskID] = int32(o.Order)
	}
	if err := h.service.ReorderProductBacklog(c.Request.Context(), projectID, orders); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось изменить порядок бэклога продукта")
		return
	}
	writeSuccess(c, gin.H{"message": "Порядок обновлён"})
}

func (h *ProductBacklogHandler) AddTaskToBacklog(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}
	type addBody struct {
		TaskID uuid.UUID `json:"task_id" binding:"required"`
		Order  int32     `json:"order"`
	}
	body, ok := bindJSON[addBody](c)
	if !ok {
		return
	}
	if err := h.service.AddToProductBacklog(c.Request.Context(), projectID, body.TaskID, body.Order); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось добавить задачу в бэклог продукта")
		return
	}
	writeSuccess(c, gin.H{"message": "Задача добавлена в бэклог продукта"})
}

func (h *ProductBacklogHandler) RemoveTaskFromBacklog(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	if err := h.service.RemoveFromProductBacklog(c.Request.Context(), projectID, taskID); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить задачу из бэклога продукта")
		return
	}
	writeSuccess(c, gin.H{"message": "Задача удалена из бэклога продукта"})
}
