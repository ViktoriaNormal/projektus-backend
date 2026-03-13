package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type TaskHandler struct {
	service *services.TaskService
}

func NewTaskHandler(service *services.TaskService) *TaskHandler {
	return &TaskHandler{service: service}
}

func (h *TaskHandler) SearchTasks(c *gin.Context) {
	var req dto.SearchTasksRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные параметры фильтрации")
		return
	}
	tasks, err := h.service.SearchTasks(c.Request.Context(), req.ProjectID, req.OwnerID, req.ExecutorID, req.ColumnID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить список задач")
		return
	}
	resp := make([]dto.TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, mapTaskToDTO(&t))
	}
	writeSuccess(c, resp)
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	var deadline *time.Time
	if req.Deadline != nil {
		d := *req.Deadline
		deadline = &d
	}

	task, err := h.service.CreateTask(c.Request.Context(), req.ProjectID, req.OwnerMemberID, req.Name, req.Description, req.ExecutorMemberID, req.ColumnID, req.SwimlaneID, deadline)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные параметры задачи")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать задачу")
		return
	}
	writeSuccess(c, mapTaskToDTO(task))
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	idStr := c.Param("taskId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	task, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Задача не найдена")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить задачу")
		return
	}
	writeSuccess(c, mapTaskToDTO(task))
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
	idStr := c.Param("taskId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	var req dto.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	task, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Задача не найдена")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить задачу")
		return
	}

	if req.Name != nil {
		task.Name = *req.Name
	}
	if req.Description != nil {
		task.Description = req.Description
	}
	if req.Deadline != nil {
		d := *req.Deadline
		task.Deadline = &d
	}
	if req.ExecutorMemberID != nil {
		idStr := req.ExecutorMemberID.String()
		task.ExecutorID = &idStr
	}
	if req.ColumnID != nil {
		task.ColumnID = req.ColumnID.String()
	}
	if req.SwimlaneID != nil {
		idStr := req.SwimlaneID.String()
		task.SwimlaneID = &idStr
	}

	updated, err := h.service.UpdateTask(c.Request.Context(), task)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные параметры задачи")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить задачу")
		return
	}
	writeSuccess(c, mapTaskToDTO(updated))
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
	idStr := c.Param("taskId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	var req dto.DeleteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	if err := h.service.DeleteTask(c.Request.Context(), id, req.Reason); err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Для удаления задачи требуется указать причину")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить задачу")
		return
	}
	writeSuccess(c, gin.H{"message": "Задача удалена"})
}

func mapTaskToDTO(t *domain.Task) dto.TaskResponse {
	projectID := uuid.MustParse(t.ProjectID)
	ownerID := uuid.MustParse(t.OwnerID)
	var execID *uuid.UUID
	if t.ExecutorID != nil {
		id := uuid.MustParse(*t.ExecutorID)
		execID = &id
	}
	var desc *string
	if t.Description != nil {
		desc = t.Description
	}
	var swimlaneID *uuid.UUID
	if t.SwimlaneID != nil {
		id := uuid.MustParse(*t.SwimlaneID)
		swimlaneID = &id
	}
	var deadline *time.Time
	if t.Deadline != nil {
		d := *t.Deadline
		deadline = &d
	}

	return dto.TaskResponse{
		ID:         uuid.MustParse(t.ID),
		Key:        t.Key,
		ProjectID:  projectID,
		OwnerID:    ownerID,
		ExecutorID: execID,
		Name:       t.Name,
		Description: desc,
		Deadline:   deadline,
		ColumnID:   uuid.MustParse(t.ColumnID),
		SwimlaneID: swimlaneID,
	}
}

